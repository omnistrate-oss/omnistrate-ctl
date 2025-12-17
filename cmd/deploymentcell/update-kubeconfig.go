package deploymentcell

import (
	"context"
	"encoding/base64"
	"fmt"
	"github.com/chelnak/ysmrr"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/model"
	openapiclientfleet "github.com/omnistrate-oss/omnistrate-sdk-go/fleet"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"github.com/omnistrate-oss/omnistrate-ctl/cmd/common"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/config"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/dataaccess"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/utils"
)

const (
	updateKubeConfigExample = `# Update kubeconfig for a deployment cell
omnistrate-ctl deployment-cell update-kubeconfig deployment-cell-id-123

# Update kubeconfig with custom kubeconfig path
omnistrate-ctl deployment-cell update-kubeconfig deployment-cell-id-123 --kubeconfig ~/.kube/my-config`
)

var updateKubeConfigCmd = &cobra.Command{
	Use:          "update-kubeconfig [deployment-cell-id]",
	Short:        "Update kubeconfig for a deployment cell",
	Long:         `Update your local kubeconfig with the configuration for the specified deployment cell and set it as the default context.`,
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

	// Default kubeconfig path
	if kubeconfigPath == "" {
		kubeconfigPath = "/tmp/kubeconfig"
	}

	ctx := context.Background()
	token, err := common.GetTokenWithLogin()
	if err != nil {
		utils.PrintError(err)
		return err
	}

	sm := ysmrr.NewSpinnerManager()
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
	spinner.UpdateMessage("Fetching kubeconfig for deployment cell (this may take a couple of minutes)...")
	kubeConfigResult, err := dataaccess.GetKubeConfigForHostCluster(ctx, token, deploymentCellID, role)
	if err != nil {
		utils.HandleSpinnerError(spinner, sm, fmt.Errorf("failed to get kubeconfig for deployment cell %s: %w", deploymentCellID, err))
		return err
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
