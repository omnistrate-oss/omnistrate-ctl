package deploymentcell

import (
	"github.com/omnistrate-oss/omnistrate-ctl/internal/model"
	openapiclientfleet "github.com/omnistrate-oss/omnistrate-sdk-go/fleet"
	"github.com/spf13/cobra"
)

var validEnvironments = []string{"PROD", "PRIVATE", "CANARY", "STAGING", "QA", "DEV", "GLOBAL"}
var validCloudProviders = []string{"aws", "azure", "gcp"}

var Cmd = &cobra.Command{
	Use:          "deployment-cell [operation] [flags]",
	Short:        "Manage Deployment Cells",
	Long:         `This command helps you manage Deployment Cells.`,
	Run:          run,
	SilenceUsage: true,
}

func init() {
	Cmd.AddCommand(adoptCmd)
	Cmd.AddCommand(statusCmd)
	Cmd.AddCommand(listCmd)
	Cmd.AddCommand(deleteCmd)
	Cmd.AddCommand(updateKubeConfigCmd)
	Cmd.AddCommand(applyPendingChangesCmd)
	Cmd.AddCommand(generateTemplateCmd)
	Cmd.AddCommand(describeTemplateCmd)
	Cmd.AddCommand(updateTemplateCmd)
	Cmd.AddCommand(debugCmd)
}

func isValidCloudProvider(provider string) bool {
	for _, valid := range validCloudProviders {
		if provider == valid {
			return true
		}
	}
	return false
}

func isValidEnvironment(env string) bool {
	for _, valid := range validEnvironments {
		if env == valid {
			return true
		}
	}
	return false
}

func run(cmd *cobra.Command, args []string) {
	err := cmd.Help()
	if err != nil {
		return
	}
}

func formatDeploymentCell(cluster *openapiclientfleet.HostCluster) model.DeploymentCell {
	return model.DeploymentCell{
		// Basic identification
		ID:          cluster.GetId(),
		Key:         cluster.GetKey(),
		Status:      cluster.GetStatus(),
		Type:        cluster.GetType(),
		Description: cluster.GetDescription(),

		// Infrastructure details
		CloudProvider:      cluster.GetCloudProvider(),
		Region:             cluster.GetRegion(),
		RegionID:           cluster.GetRegionId(),
		AccountID:          cluster.GetAccountID(),
		AccountConfigID:    cluster.GetAccountConfigId(),
		IsCustomDeployment: cluster.GetIsCustomDeployment(),

		// Deployment information
		CurrentNumberOfDeployments: cluster.GetCurrentNumberOfDeployments(),

		// Health status
		HealthStatus: formatHealthStatus(cluster.HealthStatus),

		// Network configuration
		CustomNetwork: formatCustomNetwork(cluster.CustomNetworkDetail),

		// Kubernetes details
		KubernetesDashboardEndpoint: cluster.KubernetesDashboardEndpoint,

		// Helm packages
		HelmPackages: formatHelmPackages(cluster.GetHelmPackages()),

		// Additional metadata
		Role:      cluster.Role,
		ModelType: cluster.ModelType,

		// Customer metadata
		CustomerEmail:            cluster.CustomerEmail,
		CustomerOrganizationName: cluster.CustomerOrganizationName,
	}
}

func formatHealthStatus(healthStatus *openapiclientfleet.HostClusterHealthStatus) model.DeploymentCellHealthStatus {
	if healthStatus == nil {
		return model.DeploymentCellHealthStatus{
			OverallStatus: "Unknown",
		}
	}

	var failedEntityDetails []model.EntityHealthInfo
	for _, entity := range healthStatus.GetFailedEntities() {
		failedEntityDetails = append(failedEntityDetails, model.EntityHealthInfo{
			Identifier: entity.GetIdentifier(),
			Type:       entity.GetType(),
			SyncStatus: entity.GetSyncStatus(),
			Error:      entity.Error,
		})
	}

	return model.DeploymentCellHealthStatus{
		OverallStatus:                 healthStatus.GetOverallStatus(),
		StatusMessage:                 healthStatus.StatusMessage,
		TotalEntities:                 healthStatus.GetTotalNumberOfEntities(),
		HealthyEntities:               healthStatus.GetTotalNumberOfHealthyEntities(),
		FailedEntities:                healthStatus.GetTotalNumberOfFailedEntities(),
		EntitiesByType:                healthStatus.GetTotalNumberOfEntitiesByType(),
		HealthyEntitiesByType:         healthStatus.GetTotalNumberOfHealthyEntitiesByType(),
		FailedEntitiesByType:          healthStatus.GetTotalNumberOfFailedEntitiesByType(),
		FailedEntityDetails:           failedEntityDetails,
		KubernetesControlPlaneVersion: healthStatus.KubernetesControlPlaneVersion,
	}
}

func formatCustomNetwork(customNetwork *openapiclientfleet.CustomNetworkFleetDetail) *model.CustomNetworkInfo {
	if customNetwork == nil {
		return nil
	}

	return &model.CustomNetworkInfo{
		ID:      customNetwork.Id,
		Name:    customNetwork.Name,
		CIDR:    customNetwork.Cidr,
		OrgName: customNetwork.OrgName,
	}
}

func formatHelmPackages(helmPackages []openapiclientfleet.HelmPackage) []model.HelmPackageInfo {
	var packages []model.HelmPackageInfo
	for _, pkg := range helmPackages {
		packages = append(packages, model.HelmPackageInfo{
			ChartName:     pkg.GetChartName(),
			ChartVersion:  pkg.GetChartVersion(),
			ChartRepoName: pkg.GetChartRepoName(),
			ChartRepoURL:  pkg.GetChartRepoUrl(),
			Namespace:     pkg.GetNamespace(),
			ChartValues:   pkg.GetChartValues(),
		})
	}
	return packages
}
