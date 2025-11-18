package account

import (
	"context"
	"fmt"

	"github.com/omnistrate-oss/omnistrate-ctl/cmd/common"

	"github.com/chelnak/ysmrr"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/config"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/dataaccess"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/utils"
	openapiclient "github.com/omnistrate-oss/omnistrate-sdk-go/v1"
	"github.com/spf13/cobra"
)

const (
	createExample = `# Create aws account
omctl account create [account-name] --aws-account-id=[account-id]

# Create gcp account
omctl account create [account-name] --gcp-project-id=[project-id] --gcp-project-number=[project-number]

# Create azure account
omctl account create [account-name] --azure-subscription-id=[subscription-id] --azure-tenant-id=[tenant-id]`
)

var createCmd = &cobra.Command{
	Use:          "create [account-name] [--aws-account-id=account-id] [--gcp-project-id=project-id] [--gcp-project-number=project-number] [--azure-subscription-id=subscription-id] [--azure-tenant-id=tenant-id]",
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

	// Add validation to the flags
	createCmd.MarkFlagsOneRequired("aws-account-id", "gcp-project-id", "azure-subscription-id")
	createCmd.MarkFlagsRequiredTogether("gcp-project-id", "gcp-project-number")
	createCmd.MarkFlagsRequiredTogether("azure-subscription-id", "azure-tenant-id")
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
	output, _ := cmd.Flags().GetString("output")
	if (awsAccountID != "" && gcpProjectID != "") ||
		(awsAccountID != "" && azureSubscriptionID != "") ||
		(gcpProjectID != "" && azureSubscriptionID != "") {
		return fmt.Errorf("only one of --aws-account-id, --gcp-project-id, or --azure-subscription-id can be used at a time")
	}
	// Validate user login
	token, err := common.GetTokenWithLogin()
	if err != nil {
		utils.PrintError(err)
		return err
	}

	// Initialize spinner if output is not JSON
	var sm ysmrr.SpinnerManager
	var spinner *ysmrr.Spinner
	if output != "json" {
		sm = ysmrr.NewSpinnerManager()
		msg := "Creating account..."
		spinner = sm.AddSpinner(msg)
		sm.Start()
	}

	// Create account using helper function
	params := CloudAccountParams{
		Name:                name,
		AwsAccountID:        awsAccountID,
		GcpProjectID:        gcpProjectID,
		GcpProjectNumber:    gcpProjectNumber,
		AzureSubscriptionID: azureSubscriptionID,
		AzureTenantID:       azureTenantID,
	}

	account, err := CreateCloudAccount(cmd.Context(), token, params, spinner, sm)
	if err != nil {
		return err
	}
	utils.HandleSpinnerSuccess(spinner, sm, "Successfully created account")

	// Print output
	err = utils.PrintTextTableJsonOutput(output, account)
	if err != nil {
		utils.PrintError(err)
		return err
	}

	// Print next step
	if output != "json" {
		dataaccess.PrintNextStepVerifyAccountMsg(account)
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
}

// CreateCloudAccount creates a cloud provider account and returns the account config ID and account details
// This function is reusable across different commands that need to create accounts
func CreateCloudAccount(ctx context.Context, token string, params CloudAccountParams, spinner *ysmrr.Spinner, sm ysmrr.SpinnerManager) (account *openapiclient.DescribeAccountConfigResult, err error) {
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
