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
	skipWaitFlag            = "skip-wait"
	privateLinkFlag         = "private-link"
)

func addCloudAccountProviderFlags(cmd *cobra.Command) {
	cmd.Flags().String(awsAccountIDFlag, "", "AWS account ID")
	cmd.Flags().String(gcpProjectIDFlag, "", "GCP project ID")
	cmd.Flags().String(gcpProjectNumberFlag, "", "GCP project number")
	cmd.Flags().String(azureSubscriptionIDFlag, "", "Azure subscription ID")
	cmd.Flags().String(azureTenantIDFlag, "", "Azure tenant ID")
	cmd.Flags().String(nebiusTenantIDFlag, "", "Nebius tenant ID")
	cmd.Flags().String(nebiusBindingsFileFlag, "", "Path to a YAML file describing Nebius bindings")
	cmd.Flags().Bool(skipWaitFlag, false, "Skip waiting for the account to become READY")
	cmd.Flags().Bool(privateLinkFlag, false, "Enable AWS PrivateLink connectivity for services deployed in this account")

	cmd.MarkFlagsOneRequired(
		awsAccountIDFlag,
		gcpProjectIDFlag,
		azureSubscriptionIDFlag,
		nebiusTenantIDFlag,
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
	privateLink, _ := cmd.Flags().GetBool(privateLinkFlag)

	params := CloudAccountParams{
		Name:                name,
		AwsAccountID:        awsAccountID,
		GcpProjectID:        gcpProjectID,
		GcpProjectNumber:    gcpProjectNumber,
		AzureSubscriptionID: azureSubscriptionID,
		AzureTenantID:       azureTenantID,
		NebiusTenantID:      nebiusTenantID,
		PrivateLink:         privateLink,
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
