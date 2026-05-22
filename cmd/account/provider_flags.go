package account

import (
	"github.com/spf13/cobra"
)

const (
	awsAccountIDFlag        = "aws-account-id"
	gcpProjectIDFlag        = "gcp-project-id"
	gcpProjectNumberFlag    = "gcp-project-number"
	azureSubscriptionIDFlag = "azure-subscription-id"
	azureTenantIDFlag       = "azure-tenant-id"
	nebiusTenantIDFlag      = "nebius-tenant-id"
	nebiusBindingsFileFlag  = "nebius-bindings-file"
	clusterNameFlag         = "cluster-name"
	clusterRegionFlag       = "cluster-region"
	clusterDescriptionFlag  = "cluster-description"
	skipWaitFlag            = "skip-wait"
	privateLinkFlag         = "private-link"
	allowCreateNewFlag      = "allow-create-new-cloud-native-network"
)

func addCloudAccountProviderFlags(cmd *cobra.Command) {
	addBaseCloudAccountProviderFlags(cmd)
	cmd.MarkFlagsOneRequired(
		awsAccountIDFlag,
		gcpProjectIDFlag,
		azureSubscriptionIDFlag,
		nebiusTenantIDFlag,
	)
}

func addBaseCloudAccountProviderFlags(cmd *cobra.Command) {
	cmd.Flags().String(awsAccountIDFlag, "", "AWS account ID")
	cmd.Flags().String(gcpProjectIDFlag, "", "GCP project ID")
	cmd.Flags().String(gcpProjectNumberFlag, "", "GCP project number")
	cmd.Flags().String(azureSubscriptionIDFlag, "", "Azure subscription ID")
	cmd.Flags().String(azureTenantIDFlag, "", "Azure tenant ID")
	cmd.Flags().String(nebiusTenantIDFlag, "", "Nebius tenant ID")
	cmd.Flags().String(nebiusBindingsFileFlag, "", "Path to a YAML file describing Nebius bindings")
	cmd.Flags().Bool(skipWaitFlag, false, "Skip waiting for account onboarding to become READY")

	cmd.MarkFlagsRequiredTogether(gcpProjectIDFlag, gcpProjectNumberFlag)
	cmd.MarkFlagsRequiredTogether(azureSubscriptionIDFlag, azureTenantIDFlag)
	cmd.MarkFlagsRequiredTogether(nebiusTenantIDFlag, nebiusBindingsFileFlag)
	_ = cmd.MarkFlagFilename(nebiusBindingsFileFlag)
}

func addCustomerAccountProviderFlags(cmd *cobra.Command) {
	addBaseCloudAccountProviderFlags(cmd)
	cmd.Flags().String(clusterNameFlag, "", "Name of the customer-provided Kubernetes cluster for BYOC On-Premise")
	cmd.Flags().String(clusterRegionFlag, "", "Optional region or location label for the BYOC On-Premise cluster")
	cmd.Flags().String(clusterDescriptionFlag, "", "Optional description for the BYOC On-Premise cluster")

	cmd.MarkFlagsOneRequired(
		awsAccountIDFlag,
		gcpProjectIDFlag,
		azureSubscriptionIDFlag,
		nebiusTenantIDFlag,
		clusterNameFlag,
	)
}

func cloudAccountParamsFromFlags(cmd *cobra.Command, name string) (CloudAccountParams, error) {
	awsAccountID, _ := cmd.Flags().GetString(awsAccountIDFlag)
	gcpProjectID, _ := cmd.Flags().GetString(gcpProjectIDFlag)
	gcpProjectNumber, _ := cmd.Flags().GetString(gcpProjectNumberFlag)
	azureSubscriptionID, _ := cmd.Flags().GetString(azureSubscriptionIDFlag)
	azureTenantID, _ := cmd.Flags().GetString(azureTenantIDFlag)
	nebiusTenantID, _ := cmd.Flags().GetString(nebiusTenantIDFlag)
	nebiusBindingsFile, _ := cmd.Flags().GetString(nebiusBindingsFileFlag)
	var clusterName, clusterRegion, clusterDescription string
	if cmd.Flags().Lookup(clusterNameFlag) != nil {
		clusterName, _ = cmd.Flags().GetString(clusterNameFlag)
	}
	if cmd.Flags().Lookup(clusterRegionFlag) != nil {
		clusterRegion, _ = cmd.Flags().GetString(clusterRegionFlag)
	}
	if cmd.Flags().Lookup(clusterDescriptionFlag) != nil {
		clusterDescription, _ = cmd.Flags().GetString(clusterDescriptionFlag)
	}

	params := CloudAccountParams{
		Name:                       name,
		AwsAccountID:               awsAccountID,
		GcpProjectID:               gcpProjectID,
		GcpProjectNumber:           gcpProjectNumber,
		AzureSubscriptionID:        azureSubscriptionID,
		AzureTenantID:              azureTenantID,
		NebiusTenantID:             nebiusTenantID,
		CustomerClusterName:        clusterName,
		CustomerClusterRegion:      clusterRegion,
		CustomerClusterDescription: clusterDescription,
	}

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
