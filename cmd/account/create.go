package account

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/omnistrate-oss/omnistrate-ctl/cmd/common"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/config"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/dataaccess"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/utils"
	openapiclient "github.com/omnistrate-oss/omnistrate-sdk-go/v1"
	"github.com/spf13/cobra"
)

const (
	createExample = `# Create aws account
omnistrate-ctl account create [account-name] --aws-account-id=[account-id]

# Create gcp account
omnistrate-ctl account create [account-name] --gcp-project-id=[project-id] --gcp-project-number=[project-number]

# Create azure account
omnistrate-ctl account create [account-name] --azure-subscription-id=[subscription-id] --azure-tenant-id=[tenant-id]

# Create Nebius account
omnistrate-ctl account create [account-name] --nebius-tenant-id=[tenant-id] --nebius-bindings-file=[bindings-file]`
)

var createCmd = &cobra.Command{
	Use:          "create [account-name] [--aws-account-id=account-id] [--gcp-project-id=project-id] [--gcp-project-number=project-number] [--azure-subscription-id=subscription-id] [--azure-tenant-id=tenant-id] [--nebius-tenant-id=tenant-id] [--nebius-bindings-file=file]",
	Short:        "Create a Cloud Provider Account",
	Long:         `This command helps you create a Cloud Provider Account in your account list.`,
	Example:      createExample,
	RunE:         runCreate,
	SilenceUsage: true,
}

func init() {
	createCmd.Args = cobra.ExactArgs(1) // Require exactly one argument

	createCmd.Flags().String("aws-account-id", "", "AWS account ID")
	createCmd.Flags().String("gcp-project-id", "", "GCP project ID")
	createCmd.Flags().String("gcp-project-number", "", "GCP project number")
	createCmd.Flags().String("azure-subscription-id", "", "Azure subscription ID")
	createCmd.Flags().String("azure-tenant-id", "", "Azure tenant ID")
	createCmd.Flags().String("nebius-tenant-id", "", "Nebius tenant ID")
	createCmd.Flags().String("nebius-bindings-file", "", "Path to a YAML file describing Nebius bindings")
	createCmd.Flags().Bool("skip-wait", false, "Skip waiting for account to become READY")

	// Add validation to the flags
	createCmd.MarkFlagsOneRequired("aws-account-id", "gcp-project-id", "azure-subscription-id", "nebius-tenant-id")
	createCmd.MarkFlagsRequiredTogether("gcp-project-id", "gcp-project-number")
	createCmd.MarkFlagsRequiredTogether("azure-subscription-id", "azure-tenant-id")
	createCmd.MarkFlagsRequiredTogether("nebius-tenant-id", "nebius-bindings-file")
	createCmd.MarkFlagFilename("nebius-bindings-file")
}

func runCreate(cmd *cobra.Command, args []string) error {
	defer config.CleanupArgsAndFlags(cmd, &args)

	// Retrieve args
	var name string
	if len(args) > 0 {
		name = args[0]
	}

	// Retrieve flags
	awsAccountID, _ := cmd.Flags().GetString("aws-account-id")
	gcpProjectID, _ := cmd.Flags().GetString("gcp-project-id")
	gcpProjectNumber, _ := cmd.Flags().GetString("gcp-project-number")
	azureSubscriptionID, _ := cmd.Flags().GetString("azure-subscription-id")
	azureTenantID, _ := cmd.Flags().GetString("azure-tenant-id")
	nebiusTenantID, _ := cmd.Flags().GetString("nebius-tenant-id")
	nebiusBindingsFile, _ := cmd.Flags().GetString("nebius-bindings-file")
	output, _ := cmd.Flags().GetString("output")
	skipWait, _ := cmd.Flags().GetBool("skip-wait")

	var nebiusBindings []openapiclient.NebiusAccountBindingInput
	var err error
	if nebiusBindingsFile != "" {
		nebiusBindings, err = parseNebiusBindingsFile(nebiusBindingsFile)
		if err != nil {
			return err
		}
	}

	params := CloudAccountParams{
		Name:                name,
		AwsAccountID:        awsAccountID,
		GcpProjectID:        gcpProjectID,
		GcpProjectNumber:    gcpProjectNumber,
		AzureSubscriptionID: azureSubscriptionID,
		AzureTenantID:       azureTenantID,
		NebiusTenantID:      nebiusTenantID,
		NebiusBindings:      nebiusBindings,
	}

	if err := validateCloudAccountParams(params); err != nil {
		return err
	}

	// Validate user login
	token, err := common.GetTokenWithLogin()
	if err != nil {
		utils.PrintError(err)
		return err
	}

	// Initialize spinner if output is not JSON
	var sm utils.SpinnerManager
	var spinner *utils.Spinner
	if output != "json" {
		sm = utils.NewSpinnerManager()
		msg := "Creating account..."
		spinner = sm.AddSpinner(msg)
		sm.Start()
	}

	account, err := CreateCloudAccount(cmd.Context(), token, params, spinner, sm)
	if err != nil {
		return err
	}
	utils.HandleSpinnerSuccess(spinner, sm, "Successfully created account")

	// Print output
	accountOutput, err := buildCreateAccountOutput(output, account)
	if err != nil {
		utils.PrintError(err)
		return err
	}
	err = utils.PrintTextTableJsonOutput(output, accountOutput)
	if err != nil {
		utils.PrintError(err)
		return err
	}

	// Print next step
	if output != "json" {
		dataaccess.PrintNextStepVerifyAccountMsg(account)
	}

	// Wait for account to become READY (poll up to 10 min)
	if !skipWait {
		var waitSpinner *utils.Spinner
		if output != "json" {
			fmt.Printf("\n")
			sm = utils.NewSpinnerManager()
			waitSpinner = sm.AddSpinner("Waiting for account to become READY (may take up to 10 minutes)...")
			sm.Start()
		}

		err = WaitForAccountReady(cmd.Context(), token, account.Id)
		if err != nil {
			utils.HandleSpinnerError(waitSpinner, sm, err)
			utils.PrintError(fmt.Errorf("account did not become READY: %v", err))
			return err
		}
		utils.HandleSpinnerSuccess(waitSpinner, sm, "Account is now READY")
	}

	return nil
}

// CloudAccountParams holds the parameters for creating a cloud account
type CloudAccountParams struct {
	Name                string
	AwsAccountID        string
	GcpProjectID        string
	GcpProjectNumber    string
	AzureSubscriptionID string
	AzureTenantID       string
	NebiusTenantID      string
	NebiusBindings      []openapiclient.NebiusAccountBindingInput
}

// CreateCloudAccount creates a cloud provider account and returns the account config ID and account details
// This function is reusable across different commands that need to create accounts
func CreateCloudAccount(ctx context.Context, token string, params CloudAccountParams, spinner *utils.Spinner, sm utils.SpinnerManager) (account *openapiclient.DescribeAccountConfigResult, err error) {
	// Prepare request
	request := openapiclient.CreateAccountConfigRequest2{
		Name: params.Name,
	}

	if params.AwsAccountID != "" {
		// Get aws cloud provider id
		cloudProviderID, err := dataaccess.GetCloudProviderByName(ctx, token, "aws")
		if err != nil {
			utils.HandleSpinnerError(spinner, sm, err)
			return nil, err
		}

		request.CloudProviderId = cloudProviderID
		request.AwsAccountID = &params.AwsAccountID
		request.AwsBootstrapRoleARN = utils.ToPtr("arn:aws:iam::" + params.AwsAccountID + ":role/omnistrate-bootstrap-role")
		request.Description = "AWS Account " + params.AwsAccountID
	} else if params.GcpProjectID != "" {
		// Get organization id
		user, err := dataaccess.DescribeUser(ctx, token)
		if err != nil {
			utils.HandleSpinnerError(spinner, sm, err)
			return nil, err
		}

		// Get gcp cloud provider id
		cloudProviderID, err := dataaccess.GetCloudProviderByName(ctx, token, "gcp")
		if err != nil {
			utils.HandleSpinnerError(spinner, sm, err)
			return nil, err
		}

		request.CloudProviderId = cloudProviderID
		request.GcpProjectID = &params.GcpProjectID
		request.GcpProjectNumber = &params.GcpProjectNumber
		request.GcpServiceAccountEmail = utils.ToPtr(fmt.Sprintf("bootstrap-%s@%s.iam.gserviceaccount.com", *user.OrgId, params.GcpProjectID))
		request.Description = "GCP Account " + params.GcpProjectID
	} else if params.AzureSubscriptionID != "" {
		// Get azure cloud provider id
		cloudProviderID, err := dataaccess.GetCloudProviderByName(ctx, token, "azure")
		if err != nil {
			utils.HandleSpinnerError(spinner, sm, err)
			return nil, err
		}

		request.CloudProviderId = cloudProviderID
		request.AzureSubscriptionID = &params.AzureSubscriptionID
		request.AzureTenantID = &params.AzureTenantID
		request.Description = "Azure Account " + params.AzureSubscriptionID
	} else if params.NebiusTenantID != "" {
		cloudProviderID, err := dataaccess.GetCloudProviderByName(ctx, token, "nebius")
		if err != nil {
			utils.HandleSpinnerError(spinner, sm, err)
			return nil, err
		}

		request.CloudProviderId = cloudProviderID
		request.NebiusTenantID = &params.NebiusTenantID
		request.NebiusBindings = params.NebiusBindings
		request.Description = "Nebius Account " + params.NebiusTenantID
	} else {
		return nil, fmt.Errorf("no cloud provider credentials provided")
	}

	// Create account
	accountConfigID, err := dataaccess.CreateAccount(ctx, token, request)
	if err != nil {
		utils.HandleSpinnerError(spinner, sm, err)
		return nil, err
	}

	// Describe account
	account, err = dataaccess.DescribeAccount(ctx, token, accountConfigID)
	if err != nil {
		return nil, err
	}

	return account, nil
}

func buildCreateAccountOutput(
	output string,
	account *openapiclient.DescribeAccountConfigResult,
) (any, error) {
	if output == "json" {
		return account, nil
	}

	return formatAccount(account)
}

func validateCloudAccountParams(params CloudAccountParams) error {
	providerCount := 0

	if params.AwsAccountID != "" {
		providerCount++
	}
	if params.GcpProjectID != "" || params.GcpProjectNumber != "" {
		providerCount++
	}
	if params.AzureSubscriptionID != "" || params.AzureTenantID != "" {
		providerCount++
	}
	if params.NebiusTenantID != "" || len(params.NebiusBindings) > 0 {
		providerCount++
	}

	if providerCount == 0 {
		return fmt.Errorf("one cloud provider account configuration must be provided")
	}
	if providerCount > 1 {
		return fmt.Errorf("only one of --aws-account-id, --gcp-project-id, --azure-subscription-id, or --nebius-tenant-id can be used at a time")
	}

	if (params.GcpProjectID != "" && params.GcpProjectNumber == "") || (params.GcpProjectID == "" && params.GcpProjectNumber != "") {
		return fmt.Errorf("both --gcp-project-id and --gcp-project-number must be provided together")
	}
	if (params.AzureSubscriptionID != "" && params.AzureTenantID == "") || (params.AzureSubscriptionID == "" && params.AzureTenantID != "") {
		return fmt.Errorf("both --azure-subscription-id and --azure-tenant-id must be provided together")
	}
	if (params.NebiusTenantID != "" && len(params.NebiusBindings) == 0) || (params.NebiusTenantID == "" && len(params.NebiusBindings) > 0) {
		return fmt.Errorf("both --nebius-tenant-id and --nebius-bindings-file must be provided together")
	}

	return nil
}

// waitForAccountReady polls for account status to become READY, up to 10 minutes
func WaitForAccountReady(ctx context.Context, token, accountID string) error {
	timeout := time.After(10 * time.Minute)
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-timeout:
			fmt.Fprintf(os.Stderr, "\n⚠️  Warning: Account did not become READY after 10 minutes. Please check account status with 'omnistrate-ctl account describe %s'\n", accountID)
			return fmt.Errorf("account %s did not become READY after 10 minutes", accountID)
		case <-ticker.C:
			account, err := dataaccess.DescribeAccount(ctx, token, accountID)
			if err != nil {
				return err
			}
			if account.Status == "READY" {
				return nil
			}
		}
	}
}
