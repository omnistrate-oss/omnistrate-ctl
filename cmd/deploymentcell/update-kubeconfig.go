package deploymentcell

import (
	"context"
	"encoding/base64"
	"fmt"
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
omctl deployment-cell update-kubeconfig deployment-cell-id-123

# Update kubeconfig with custom kubeconfig path
omctl deployment-cell update-kubeconfig deployment-cell-id-123 --kubeconfig ~/.kube/my-config`
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
	updateKubeConfigCmd.Flags().String("kubeconfig", "", "Path to kubeconfig file (default: $HOME/.kube/config)")
}

func runUpdateKubeConfig(cmd *cobra.Command, args []string) error {
	defer config.CleanupArgsAndFlags(cmd, &args)

	deploymentCellID := args[0]

	kubeconfigPath, err := cmd.Flags().GetString("kubeconfig")
	if err != nil {
		utils.PrintError(err)
		return err
	}

	// Default kubeconfig path
	if kubeconfigPath == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			utils.PrintError(fmt.Errorf("failed to get home directory: %w", err))
			return err
		}
		kubeconfigPath = filepath.Join(homeDir, ".kube", "config")
	}

	ctx := context.Background()
	token, err := common.GetTokenWithLogin()
	if err != nil {
		utils.PrintError(err)
		return err
	}

	// Get kubeconfig data from the API
	kubeConfigResult, err := dataaccess.GetKubeConfigForHostCluster(ctx, token, deploymentCellID)
	if err != nil {
		utils.PrintError(fmt.Errorf("failed to get kubeconfig for deployment cell %s: %w", deploymentCellID, err))
		return err
	}

	// Validate base64 encoded data
	if _, err := base64.StdEncoding.DecodeString(kubeConfigResult.GetCaDataBase64()); err != nil {
		utils.PrintError(fmt.Errorf("invalid CA data: %w", err))
		return err
	}
	if _, err := base64.StdEncoding.DecodeString(kubeConfigResult.GetClientCertificateDataBase64()); err != nil {
		utils.PrintError(fmt.Errorf("invalid client certificate data: %w", err))
		return err
	}
	if _, err := base64.StdEncoding.DecodeString(kubeConfigResult.GetClientKeyDataBase64()); err != nil {
		utils.PrintError(fmt.Errorf("invalid client key data: %w", err))
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
			utils.PrintError(fmt.Errorf("failed to read kubeconfig file: %w", err))
			return err
		}
		if err := yaml.Unmarshal(data, &kubeConfig); err != nil {
			utils.PrintError(fmt.Errorf("failed to parse kubeconfig file: %w", err))
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
	if err := os.MkdirAll(filepath.Dir(kubeconfigPath), 0755); err != nil {
		utils.PrintError(fmt.Errorf("failed to create kubeconfig directory: %w", err))
		return err
	}

	// Write kubeconfig file
	data, err := yaml.Marshal(&kubeConfig)
	if err != nil {
		utils.PrintError(fmt.Errorf("failed to marshal kubeconfig: %w", err))
		return err
	}

	if err := os.WriteFile(kubeconfigPath, data, 0600); err != nil {
		utils.PrintError(fmt.Errorf("failed to write kubeconfig file: %w", err))
		return err
	}

	fmt.Printf("Successfully updated kubeconfig at %s\n", kubeconfigPath)
	fmt.Printf("Current context set to: %s\n", contextName)
	fmt.Printf("You can now use kubectl to interact with your deployment cell.\n")

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
