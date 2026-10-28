package deploymentcell

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/omnistrate-oss/omnistrate-ctl/cmd/common"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/config"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/dataaccess"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/model"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/utils"
	openapiclientfleet "github.com/omnistrate-oss/omnistrate-sdk-go/fleet"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

const (
	updateKubeConfigExample = `# Update kubeconfig for a deployment cell
omnistrate-ctl deployment-cell update-kubeconfig deployment-cell-id-123

# Update kubeconfig with custom kubeconfig path
omnistrate-ctl deployment-cell update-kubeconfig deployment-cell-id-123 --kubeconfig ~/.kube/my-config

# Request access when manual approval is enabled
omnistrate-ctl deployment-cell update-kubeconfig deployment-cell-id-123 --reason "Investigate deployment health"`

	accessRequestApprovalWaitTimeout  = 5 * time.Minute
	accessRequestApprovalPollInterval = 2 * time.Second
)

var updateKubeConfigCmd = &cobra.Command{
	Use:          "update-kubeconfig [deployment-cell-id]",
	Short:        "Update kubeconfig for a deployment cell",
	Long:         `Create or reuse a deployment-cell access request, wait for automatic approval, then update the local kubeconfig and set it as the default context. Manual requests print approval instructions without writing credentials.`,
	Example:      updateKubeConfigExample,
	Args:         cobra.ExactArgs(1),
	RunE:         runUpdateKubeConfig,
	SilenceUsage: true,
}

type KubeConfig struct {
	APIVersion     string              `yaml:"apiVersion"`
	Kind           string              `yaml:"kind"`
	Clusters       []KubeConfigCluster `yaml:"clusters"`
	Contexts       []KubeConfigContext `yaml:"contexts"`
	CurrentContext string              `yaml:"current-context"`
	Users          []KubeConfigUser    `yaml:"users"`
}

type KubeConfigCluster struct {
	Name    string                `yaml:"name"`
	Cluster KubeConfigClusterData `yaml:"cluster"`
}

type KubeConfigClusterData struct {
	Server                   string `yaml:"server"`
	CertificateAuthorityData string `yaml:"certificate-authority-data"`
}

type KubeConfigContext struct {
	Name    string                `yaml:"name"`
	Context KubeConfigContextData `yaml:"context"`
}

type KubeConfigContextData struct {
	Cluster string `yaml:"cluster"`
	User    string `yaml:"user"`
}

type KubeConfigUser struct {
	Name string             `yaml:"name"`
	User KubeConfigUserData `yaml:"user"`
}

type KubeConfigUserData struct {
	ClientCertificateData string `yaml:"client-certificate-data"`
	ClientKeyData         string `yaml:"client-key-data"`
	Token                 string `yaml:"token,omitempty"`
}

func init() {
	updateKubeConfigCmd.Flags().String("kubeconfig", "", "Path to kubeconfig file (default: /tmp/kubeconfig)")
	updateKubeConfigCmd.Flags().String("customer-email", "", "Customer email to filter by (optional)")
	updateKubeConfigCmd.Flags().String("role", "", "Access role for the kube context (optional, default: 'cluster-reader')")
	updateKubeConfigCmd.Flags().String("reason", "", "Reason for requesting access when manual approval is enabled")
}

type kubeConfigAccessDependencies struct {
	getKubeConfig          func(context.Context, string, string, string) (*openapiclientfleet.KubeConfigHostClusterResult, error)
	createRequest          func(context.Context, string, dataaccess.CreateAccessRequestRequest) (*model.AccessRequest, error)
	waitForApproval        func(context.Context, string, *model.AccessRequest) (*model.AccessRequest, error)
	kubeConfigWaitTimeout  time.Duration
	kubeConfigPollInterval time.Duration
	promptReason           func() (string, error)
}

var defaultKubeConfigAccessDependencies = kubeConfigAccessDependencies{
	getKubeConfig: dataaccess.GetKubeConfigForHostCluster,
	createRequest: dataaccess.CreateAccessRequest,
	waitForApproval: func(ctx context.Context, token string, request *model.AccessRequest) (*model.AccessRequest, error) {
		return waitForAccessRequestApproval(
			ctx,
			token,
			request,
			dataaccess.DescribeAccessRequest,
			accessRequestApprovalWaitTimeout,
			accessRequestApprovalPollInterval,
		)
	},
	kubeConfigWaitTimeout:  accessRequestApprovalWaitTimeout,
	kubeConfigPollInterval: accessRequestApprovalPollInterval,
	promptReason: func() (string, error) {
		return utils.PromptInput(
			"Reason for requesting deployment-cell access:",
			"Investigate deployment health",
			func(value string) error {
				if strings.TrimSpace(value) == "" {
					return fmt.Errorf("access reason is required")
				}
				return nil
			},
		)
	},
}

func runUpdateKubeConfig(cmd *cobra.Command, args []string) error {
	defer config.CleanupArgsAndFlags(cmd, &args)

	deploymentCellID := args[0]

	kubeconfigPath, err := cmd.Flags().GetString("kubeconfig")
	if err != nil {
		utils.PrintError(err)
		return err
	}

	customerEmail, err := cmd.Flags().GetString("customer-email")
	if err != nil {
		utils.PrintError(err)
		return err
	}

	role, err := cmd.Flags().GetString("role")
	if err != nil {
		utils.PrintError(err)
		return err
	}

	reason, err := cmd.Flags().GetString("reason")
	if err != nil {
		utils.PrintError(err)
		return err
	}

	// Default kubeconfig path
	if kubeconfigPath == "" {
		kubeconfigPath = "/tmp/kubeconfig"
	}

	ctx := cmd.Context()
	token, err := common.GetTokenWithLogin()
	if err != nil {
		utils.PrintError(err)
		return err
	}

	sm := utils.NewSpinnerManager()
	spinner := sm.AddSpinner("Looking up deployment cell...")
	sm.Start()

	var hostClusters *openapiclientfleet.ListHostClustersResult
	if hostClusters, err = dataaccess.ListHostClusters(ctx, token, nil, nil); err != nil {
		utils.PrintError(err)
		return err
	}

	// Convert to model structure and filter by ID / key
	var deploymentCells []model.DeploymentCell
	for _, cluster := range hostClusters.GetHostClusters() {
		if cluster.GetId() != deploymentCellID && cluster.GetKey() != deploymentCellID {
			continue // Skip if ID or key does not match
		}

		if customerEmail != "" && cluster.GetCustomerEmail() != customerEmail {
			continue // Skip if customer email does not match
		}

		deploymentCell := formatDeploymentCell(&cluster)
		deploymentCells = append(deploymentCells, deploymentCell)
	}

	if len(deploymentCells) > 1 {
		utils.HandleSpinnerError(spinner, sm, fmt.Errorf("multiple deployment cells found for ID %s, please specify a unique ID or a customer email to filter by", deploymentCellID))
		return fmt.Errorf("multiple deployment cells found for ID %s", deploymentCellID)
	} else if len(deploymentCells) == 0 {
		utils.HandleSpinnerError(spinner, sm, fmt.Errorf("no deployment cell found for ID %s", deploymentCellID))
		return fmt.Errorf("no deployment cell found for ID %s", deploymentCellID)
	} else {
		deploymentCellID = deploymentCells[0].ID
	}

	// Get kubeconfig data from the API
	spinner.UpdateMessage("Requesting access and fetching kubeconfig (this may take a couple of minutes)...")
	kubeConfigResult, accessRequest, err := resolveKubeConfigAccess(
		ctx,
		token,
		deploymentCellID,
		role,
		reason,
		defaultKubeConfigAccessDependencies,
	)
	if err != nil {
		utils.HandleSpinnerError(spinner, sm, fmt.Errorf("failed to get kubeconfig for deployment cell %s: %w", deploymentCellID, err))
		return err
	}
	if kubeConfigResult == nil {
		utils.HandleSpinnerSuccess(spinner, sm, "Access request recorded")
		printAccessRequestInstructions(accessRequest, deploymentCellID, role)
		return nil
	}

	// Validate base64 encoded data
	if _, err := base64.StdEncoding.DecodeString(kubeConfigResult.GetCaDataBase64()); err != nil {
		utils.HandleSpinnerError(spinner, sm, fmt.Errorf("invalid CA data: %w", err))
		return err
	}
	if _, err := base64.StdEncoding.DecodeString(kubeConfigResult.GetClientCertificateDataBase64()); err != nil {
		utils.HandleSpinnerError(spinner, sm, fmt.Errorf("invalid client certificate data: %w", err))
		return err
	}
	if _, err := base64.StdEncoding.DecodeString(kubeConfigResult.GetClientKeyDataBase64()); err != nil {
		utils.HandleSpinnerError(spinner, sm, fmt.Errorf("invalid client key data: %w", err))
		return err
	}

	// Create context name from deployment cell ID
	contextName := fmt.Sprintf("omnistrate-%s", deploymentCellID)
	clusterName := fmt.Sprintf("omnistrate-%s", deploymentCellID)
	userName := fmt.Sprintf("omnistrate-%s", kubeConfigResult.GetUserName())

	// Load existing kubeconfig or create new one
	var kubeConfig KubeConfig
	if _, err := os.Stat(kubeconfigPath); err == nil {
		// File exists, load it
		data, err := os.ReadFile(kubeconfigPath)
		if err != nil {
			utils.HandleSpinnerError(spinner, sm, fmt.Errorf("failed to read kubeconfig file: %w", err))
			return err
		}
		if err := yaml.Unmarshal(data, &kubeConfig); err != nil {
			utils.HandleSpinnerError(spinner, sm, fmt.Errorf("failed to parse kubeconfig file: %w", err))
			return err
		}
	} else {
		// File doesn't exist, create new kubeconfig
		kubeConfig = KubeConfig{
			APIVersion: "v1",
			Kind:       "Config",
			Clusters:   []KubeConfigCluster{},
			Contexts:   []KubeConfigContext{},
			Users:      []KubeConfigUser{},
		}
	}

	// Remove existing cluster, context, and user if they exist
	kubeConfig.Clusters = removeCluster(kubeConfig.Clusters, clusterName)
	kubeConfig.Contexts = removeContext(kubeConfig.Contexts, contextName)
	kubeConfig.Users = removeUser(kubeConfig.Users, userName)

	// Add new cluster
	kubeConfig.Clusters = append(kubeConfig.Clusters, KubeConfigCluster{
		Name: clusterName,
		Cluster: KubeConfigClusterData{
			Server:                   "https://" + kubeConfigResult.GetApiServerEndpoint(),
			CertificateAuthorityData: kubeConfigResult.GetCaDataBase64(),
		},
	})

	// Add new user
	kubeConfig.Users = append(kubeConfig.Users, KubeConfigUser{
		Name: userName,
		User: KubeConfigUserData{
			ClientCertificateData: kubeConfigResult.GetClientCertificateDataBase64(),
			ClientKeyData:         kubeConfigResult.GetClientKeyDataBase64(),
			Token:                 kubeConfigResult.GetServiceAccountToken(),
		},
	})

	// Add new context
	kubeConfig.Contexts = append(kubeConfig.Contexts, KubeConfigContext{
		Name: contextName,
		Context: KubeConfigContextData{
			Cluster: clusterName,
			User:    userName,
		},
	})

	// Set as current context
	kubeConfig.CurrentContext = contextName

	// Ensure directory exists
	if err = os.MkdirAll(filepath.Dir(kubeconfigPath), 0755); err != nil {
		utils.HandleSpinnerError(spinner, sm, fmt.Errorf("failed to create kubeconfig directory: %w", err))
		return err
	}

	// Write kubeconfig file
	data, err := yaml.Marshal(&kubeConfig)
	if err != nil {
		utils.HandleSpinnerError(spinner, sm, fmt.Errorf("failed to marshal kubeconfig: %w", err))
		return err
	}

	if err := os.WriteFile(kubeconfigPath, data, 0600); err != nil {
		utils.HandleSpinnerError(spinner, sm, fmt.Errorf("failed to write kubeconfig file: %w", err))
		return err
	}

	utils.HandleSpinnerSuccess(spinner, sm, fmt.Sprintf("Successfully updated kubeconfig at %s\n", kubeconfigPath))
	utils.PrintInfo(fmt.Sprintf("Current context set to: %s", contextName))

	return nil
}

func resolveKubeConfigAccess(
	ctx context.Context,
	token string,
	deploymentCellID string,
	role string,
	reason string,
	dependencies kubeConfigAccessDependencies,
) (*openapiclientfleet.KubeConfigHostClusterResult, *model.AccessRequest, error) {
	requestReason := strings.TrimSpace(reason)
	accessRequest, requestErr := dependencies.createRequest(ctx, token, dataaccess.CreateAccessRequestRequest{
		DeploymentCellID: deploymentCellID,
		Role:             normalizedKubeConfigRole(role),
		Reason:           optionalString(requestReason),
	})
	if requestErr != nil && requestReason == "" && dataaccess.IsManualAccessReasonRequiredError(requestErr) {
		requestReason, requestErr = dependencies.promptReason()
		requestReason = strings.TrimSpace(requestReason)
		if requestErr != nil {
			return nil, nil, requestErr
		}
		if requestReason == "" {
			return nil, nil, fmt.Errorf("access reason is required")
		}
		accessRequest, requestErr = dependencies.createRequest(ctx, token, dataaccess.CreateAccessRequestRequest{
			DeploymentCellID: deploymentCellID,
			Role:             normalizedKubeConfigRole(role),
			Reason:           &requestReason,
		})
	}
	if requestErr != nil {
		return nil, nil, requestErr
	}
	if accessRequest == nil {
		return nil, nil, fmt.Errorf("access request API returned no request")
	}

	if strings.EqualFold(accessRequest.Status, "APPROVING") {
		if dependencies.waitForApproval == nil {
			return nil, accessRequest, fmt.Errorf("access request approval waiter is not configured")
		}
		accessRequest, requestErr = dependencies.waitForApproval(ctx, token, accessRequest)
		if requestErr != nil {
			return nil, accessRequest, requestErr
		}
		if accessRequest == nil {
			return nil, nil, fmt.Errorf("access request approval returned no request")
		}
	}

	switch strings.ToUpper(accessRequest.Status) {
	case "APPROVED":
		waitTimeout := dependencies.kubeConfigWaitTimeout
		if waitTimeout <= 0 {
			waitTimeout = accessRequestApprovalWaitTimeout
		}
		pollInterval := dependencies.kubeConfigPollInterval
		if pollInterval <= 0 {
			pollInterval = accessRequestApprovalPollInterval
		}
		kubeConfig, stillPending, err := waitForKubeConfigAvailability(
			ctx,
			token,
			deploymentCellID,
			role,
			dependencies.getKubeConfig,
			waitTimeout,
			pollInterval,
		)
		if err != nil {
			return nil, accessRequest, err
		}
		if stillPending {
			return nil, accessRequest, nil
		}
		return kubeConfig, accessRequest, nil
	case "PENDING", "APPROVING":
		return nil, accessRequest, nil
	default:
		return nil, accessRequest, fmt.Errorf("access request %s is %s", accessRequest.ID, accessRequest.Status)
	}
}

func waitForKubeConfigAvailability(
	ctx context.Context,
	token string,
	deploymentCellID string,
	role string,
	getKubeConfig func(context.Context, string, string, string) (*openapiclientfleet.KubeConfigHostClusterResult, error),
	timeout time.Duration,
	pollInterval time.Duration,
) (*openapiclientfleet.KubeConfigHostClusterResult, bool, error) {
	if getKubeConfig == nil {
		return nil, false, fmt.Errorf("kubeconfig operation is not configured")
	}

	waitCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	for {
		result, err := getKubeConfig(waitCtx, token, deploymentCellID, role)
		if err == nil {
			return result, false, nil
		}
		if !dataaccess.IsKubeConfigAccessRequestPendingError(err) {
			return nil, false, err
		}

		select {
		case <-waitCtx.Done():
			if waitCtx.Err() == context.DeadlineExceeded {
				return nil, true, nil
			}
			return nil, false, waitCtx.Err()
		case <-time.After(pollInterval):
		}
	}
}

func waitForAccessRequestApproval(
	ctx context.Context,
	token string,
	request *model.AccessRequest,
	describeRequest func(context.Context, string, string) (*model.AccessRequest, error),
	timeout time.Duration,
	pollInterval time.Duration,
) (*model.AccessRequest, error) {
	if request == nil {
		return nil, fmt.Errorf("access request is required")
	}
	if describeRequest == nil {
		return request, fmt.Errorf("access request describe operation is not configured")
	}

	waitCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	current := request
	for {
		described, err := describeRequest(waitCtx, token, current.ID)
		if err != nil {
			if waitCtx.Err() == context.DeadlineExceeded {
				return current, nil
			}
			return current, err
		}
		if described == nil {
			return current, fmt.Errorf("access request API returned no request while waiting for approval")
		}
		current = described

		switch strings.ToUpper(current.Status) {
		case "APPROVED", "PENDING":
			return current, nil
		case "APPROVING":
			// Continue polling. A timeout returns the reusable request so the
			// caller can print rerun instructions instead of losing its ID.
		default:
			return current, nil
		}

		select {
		case <-waitCtx.Done():
			if waitCtx.Err() == context.DeadlineExceeded {
				return current, nil
			}
			return current, waitCtx.Err()
		case <-time.After(pollInterval):
		}
	}
}

func normalizedKubeConfigRole(role string) string {
	if strings.TrimSpace(role) == "" {
		return "cluster-reader"
	}
	return strings.TrimSpace(role)
}

func optionalString(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}

func printAccessRequestInstructions(request *model.AccessRequest, deploymentCellID, role string) {
	if request == nil {
		return
	}
	switch strings.ToUpper(request.Status) {
	case "PENDING":
		utils.PrintInfo(fmt.Sprintf("Access request %s is pending approval.", request.ID))
	case "APPROVING":
		utils.PrintInfo(fmt.Sprintf("Access request %s is being approved.", request.ID))
	case "APPROVED":
		utils.PrintInfo(fmt.Sprintf("Access request %s is approved; kubeconfig credentials are still being prepared.", request.ID))
	default:
		utils.PrintInfo(fmt.Sprintf("Access request %s has status %s.", request.ID, request.Status))
	}
	if approvalURL := accessRequestApprovalURL(request); approvalURL != "" {
		utils.PrintURL("Approval URL", approvalURL)
	}
	command := fmt.Sprintf("omnistrate-ctl deployment-cell update-kubeconfig %s", deploymentCellID)
	if normalizedKubeConfigRole(role) != "cluster-reader" {
		command += " --role " + normalizedKubeConfigRole(role)
	}
	if strings.EqualFold(request.Status, "APPROVED") {
		utils.PrintInfo("Retry with: " + command)
	} else {
		utils.PrintInfo("After approval, rerun: " + command)
	}
}

func accessRequestApprovalURL(request *model.AccessRequest) string {
	if request == nil {
		return ""
	}
	if !strings.EqualFold(request.Status, "PENDING") {
		return ""
	}
	if strings.TrimSpace(request.ApprovalURL) != "" {
		return strings.TrimSpace(request.ApprovalURL)
	}
	if request.ID != "" {
		// Temporary console route until the backend returns the final approval URL.
		return fmt.Sprintf("https://%s/access-requests/%s", config.GetRootDomain(), request.ID)
	}
	return ""
}

func removeCluster(clusters []KubeConfigCluster, name string) []KubeConfigCluster {
	var result []KubeConfigCluster
	for _, cluster := range clusters {
		if cluster.Name != name {
			result = append(result, cluster)
		}
	}
	return result
}

func removeContext(contexts []KubeConfigContext, name string) []KubeConfigContext {
	var result []KubeConfigContext
	for _, context := range contexts {
		if context.Name != name {
			result = append(result, context)
		}
	}
	return result
}

func removeUser(users []KubeConfigUser, name string) []KubeConfigUser {
	var result []KubeConfigUser
	for _, user := range users {
		if user.Name != name {
			result = append(result, user)
		}
	}
	return result
}
