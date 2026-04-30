package account

import (
	"github.com/spf13/cobra"
)

const (
	awsAccountIDFlag           = "aws-account-id"
	gcpProjectIDFlag           = "gcp-project-id"
	gcpProjectNumberFlag       = "gcp-project-number"
	azureSubscriptionIDFlag    = "azure-subscription-id"
	azureTenantIDFlag          = "azure-tenant-id"
	nebiusTenantIDFlag         = "nebius-tenant-id"
	nebiusBindingsFileFlag     = "nebius-bindings-file"
	clusterNameFlag            = "cluster-name"
	clusterRegionFlag          = "cluster-region"
	clusterDescriptionFlag     = "cluster-description"
	skipWaitFlag               = "skip-wait"
	privateLinkFlag            = "private-link"
	allowCreateNewFlag         = "allow-create-new-cloud-native-network"
)

func addCloudAccountProviderFlags(cmd *cobra.Command) {
	cmd.Flags().String(awsAccountIDFlag, "", "AWS account ID")
	cmd.Flags().String(gcpProjectIDFlag, "", "GCP project ID")
	cmd.Flags().String(gcpProjectNumberFlag, "", "GCP project number")
	cmd.Flags().String(azureSubscriptionIDFlag, "", "Azure subscription ID")
	cmd.Flags().String(azureTenantIDFlag, "", "Azure tenant ID")
	cmd.Flags().String(nebiusTenantIDFlag, "", "Nebius tenant ID")
	cmd.Flags().String(nebiusBindingsFileFlag, "", "Path to a YAML file describing Nebius bindings")
	cmd.Flags().String(clusterNameFlag, "", "Name of the customer-provided Kubernetes cluster (byoc-anywhere)")
	cmd.Flags().String(clusterRegionFlag, "", "Region or location label for the cluster (byoc-anywhere)")
	cmd.Flags().String(clusterDescriptionFlag, "", "Free-form description of the cluster (byoc-anywhere)")
	cmd.Flags().Bool(skipWaitFlag, false, "Skip waiting for the account to become READY")

	cmd.MarkFlagsOneRequired(
		awsAccountIDFlag,
		gcpProjectIDFlag,
		azureSubscriptionIDFlag,
		nebiusTenantIDFlag,
		clusterNameFlag,
	)
	cmd.MarkFlagsRequiredTogether(gcpProjectIDFlag, gcpProjectNumberFlag)
	cmd.MarkFlagsRequiredTogether(azureSubscriptionIDFlag, azureTenantIDFlag)
	cmd.MarkFlagsRequiredTogether(nebiusTenantIDFlag, nebiusBindingsFileFlag)
	_ = cmd.MarkFlagFilename(nebiusBindingsFileFlag)
}

func cloudAccountParamsFromFlags(cmd *cobra.Command, name string) (CloudAccountParams, error) {
	awsAccountID, _ := cmd.Flags().GetString(awsAccountIDFlag)
	gcpProjectID, _ := cmd.Flags().GetString(gcpProjectIDFlag)
	gcpProjectNumber, _ := cmd.Flags().GetString(gcpProjectNumberFlag)
	azureSubscriptionID, _ := cmd.Flags().GetString(azureSubscriptionIDFlag)
	azureTenantID, _ := cmd.Flags().GetString(azureTenantIDFlag)
	nebiusTenantID, _ := cmd.Flags().GetString(nebiusTenantIDFlag)
	nebiusBindingsFile, _ := cmd.Flags().GetString(nebiusBindingsFileFlag)

	clusterName, _ := cmd.Flags().GetString(clusterNameFlag)
	clusterRegion, _ := cmd.Flags().GetString(clusterRegionFlag)
	clusterDescription, _ := cmd.Flags().GetString(clusterDescriptionFlag)

	params := CloudAccountParams{
		Name:                name,
		AwsAccountID:        awsAccountID,
		GcpProjectID:        gcpProjectID,
		GcpProjectNumber:    gcpProjectNumber,
		AzureSubscriptionID: azureSubscriptionID,
		AzureTenantID:       azureTenantID,
		NebiusTenantID:      nebiusTenantID,
		ClusterName:         clusterName,
		ClusterRegion:       clusterRegion,
		ClusterDescription:  clusterDescription,
	}

	// --private-link / --allow-create-new are only meaningful for the BYOA
	// customer onboarding flow (they map to injected input parameters on the
	// account-config resource). They are registered conditionally by the
	// caller, so only read them when present on the command.
	if cmd.Flags().Lookup(privateLinkFlag) != nil {
		privateLink, _ := cmd.Flags().GetBool(privateLinkFlag)
		params.PrivateLink = privateLink
	}
	if cmd.Flags().Lookup(allowCreateNewFlag) != nil {
		allowCreateNew, _ := cmd.Flags().GetBool(allowCreateNewFlag)
		params.AllowCreateNew = allowCreateNew
	}

	if nebiusBindingsFile != "" {
		bindings, err := parseNebiusBindingsFile(nebiusBindingsFile)
		if err != nil {
			return CloudAccountParams{}, err
		}
		params.NebiusBindings = bindings
	}

	if err := validateCloudAccountParams(params); err != nil {
		return CloudAccountParams{}, err
	}

	return params, nil
}
