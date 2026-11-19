package account

import (
	"fmt"
	"strings"

	"github.com/omnistrate-oss/omnistrate-ctl/cmd/common"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/config"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/dataaccess"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/utils"
	openapiclient "github.com/omnistrate-oss/omnistrate-sdk-go/v1"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

const (
	privateArtifactPublicationEnabledFlag = "private-artifact-publication-enabled"

	updateExample = `# Update account name and description
omnistrate-ctl account update [account-name or account-id] --name=[new-name] --description=[new-description]

# Replace Nebius bindings on an existing Nebius account
omnistrate-ctl account update [account-name or account-id] --nebius-bindings-file=[bindings-file]

# Enable private managed-artifact publication for an AWS non-BYOA account
omnistrate-ctl account update [account-name or account-id] --private-artifact-publication-enabled=true

# Disable private managed-artifact publication (use = with false)
omnistrate-ctl account update [account-name or account-id] --private-artifact-publication-enabled=false`

	nextStepPrivateArtifactPublicationMsgTemplate = `
Next step:
Apply the private artifact publication setting to account %s.
Update the existing AWS CloudFormation stack or Terraform deployment for this account; do not create a new one.%s
Track progress with: omnistrate-ctl account describe %s`
)

var updateCmd = &cobra.Command{
	Use:          "update [account-name or account-id] [flags]",
	Short:        "Update a Cloud Provider Account",
	Long:         "This command helps you update mutable fields on an existing cloud provider account.",
	Example:      updateExample,
	RunE:         runUpdate,
	SilenceUsage: true,
}

func init() {
	updateCmd.Args = cobra.ExactArgs(1)

	updateCmd.Flags().String("name", "", "Updated account name")
	updateCmd.Flags().String("description", "", "Updated account description")
	updateCmd.Flags().String("nebius-bindings-file", "", "Path to a YAML file describing the full replacement Nebius bindings")
	updateCmd.Flags().String(tagsFlag, "", `Full replacement set of custom tags for the account in key=value format, separated by commas (e.g. env=prod,team=platform). Escape commas inside values with \,`)
	updateCmd.Flags().Bool(privateArtifactPublicationEnabledFlag, false, "Whether an AWS non-BYOA account may publish managed artifacts to private ECR")
	updateCmd.Flags().Bool("skip-wait", false, "Skip waiting for account to become READY after replacing Nebius bindings")
	_ = updateCmd.MarkFlagFilename("nebius-bindings-file")
}

type UpdateCloudAccountParams struct {
	ID                                string
	Name                              *string
	Description                       *string
	NebiusBindings                    []openapiclient.NebiusAccountBindingInput
	CustomTags                        []openapiclient.CustomTag
	PrivateArtifactPublicationEnabled *bool
}

func runUpdate(cmd *cobra.Command, args []string) error {
	defer config.CleanupArgsAndFlags(cmd, &args)

	output, err := cmd.Flags().GetString("output")
	if err != nil {
		utils.PrintError(err)
		return err
	}

	if err = validateDescribeArguments(args, output); err != nil {
		utils.PrintError(err)
		return err
	}

	name, _ := cmd.Flags().GetString("name")
	description, _ := cmd.Flags().GetString("description")
	nebiusBindingsFile, _ := cmd.Flags().GetString("nebius-bindings-file")
	skipWait, _ := cmd.Flags().GetBool("skip-wait")
	privateArtifactPublicationEnabled, err := optionalBoolFlag(cmd, privateArtifactPublicationEnabledFlag)
	if err != nil {
		utils.PrintError(err)
		return err
	}

	params, err := buildUpdateAccountParams(
		cmd, args[0], name, description, nebiusBindingsFile, privateArtifactPublicationEnabled,
	)
	if err != nil {
		utils.PrintError(err)
		return err
	}

	token, err := common.GetTokenWithLogin()
	if err != nil {
		utils.PrintError(err)
		return err
	}

	var sm utils.SpinnerManager
	var spinner *utils.Spinner
	if output != "json" {
		sm = utils.NewSpinnerManager()
		spinner = sm.AddSpinner("Resolving account...")
		sm.Start()
	}

	params.ID, err = getAccountID(cmd.Context(), token, args[0])
	if err != nil {
		utils.HandleSpinnerError(spinner, sm, err)
		return err
	}

	// Validate provider-specific fields before the API persists any metadata changes.
	if len(params.NebiusBindings) > 0 || params.PrivateArtifactPublicationEnabled != nil {
		existingAccount, describeErr := dataaccess.DescribeAccount(cmd.Context(), token, params.ID)
		if describeErr != nil {
			utils.HandleSpinnerError(spinner, sm, describeErr)
			return describeErr
		}
		if err = validateUpdateTargetAccount(params, existingAccount); err != nil {
			utils.HandleSpinnerError(spinner, sm, err)
			return err
		}
	}

	if output != "json" {
		utils.HandleSpinnerSuccess(spinner, sm, "Resolved account")
		sm = utils.NewSpinnerManager()
		spinner = sm.AddSpinner("Updating account...")
		sm.Start()
	}

	accountConfigID, err := dataaccess.UpdateAccount(cmd.Context(), token, dataaccess.UpdateAccountParams{
		AccountConfigID:                   params.ID,
		Name:                              params.Name,
		Description:                       params.Description,
		NebiusBindings:                    params.NebiusBindings,
		CustomTags:                        params.CustomTags,
		PrivateArtifactPublicationEnabled: params.PrivateArtifactPublicationEnabled,
	})
	if err != nil {
		utils.HandleSpinnerError(spinner, sm, err)
		return err
	}
	utils.HandleSpinnerSuccess(spinner, sm, "Successfully updated account")

	if len(params.NebiusBindings) > 0 && !skipWait {
		var waitSpinner *utils.Spinner
		if output != "json" {
			fmt.Printf("\n")
			sm = utils.NewSpinnerManager()
			waitSpinner = sm.AddSpinner("Waiting for account to become READY after binding replacement...")
			sm.Start()
		}

		if err = WaitForAccountReady(cmd.Context(), token, accountConfigID); err != nil {
			utils.HandleSpinnerError(waitSpinner, sm, err)
			utils.PrintError(fmt.Errorf("account did not become READY: %v", err))
			return err
		}

		utils.HandleSpinnerSuccess(waitSpinner, sm, "Account is now READY")
	}

	account, err := dataaccess.DescribeAccount(cmd.Context(), token, accountConfigID)
	if err != nil {
		utils.PrintError(err)
		return err
	}

	if err = printDescribeOutput(output, account); err != nil {
		utils.PrintError(err)
		return err
	}

	if output != "json" && account.Status != "READY" {
		if params.PrivateArtifactPublicationEnabled != nil {
			printNextStepPrivateArtifactPublicationMsg(account)
		} else {
			dataaccess.PrintNextStepVerifyAccountMsg(account)
		}
	}

	return nil
}

// optionalBoolFlag returns nil when the flag was not supplied so explicit false values are preserved.
func optionalBoolFlag(cmd *cobra.Command, name string) (*bool, error) {
	if !cmd.Flags().Changed(name) {
		return nil, nil
	}

	value, err := cmd.Flags().GetBool(name)
	if err != nil {
		return nil, err
	}

	return &value, nil
}

func validateUpdateTargetAccount(params UpdateCloudAccountParams, account *openapiclient.DescribeAccountConfigResult) error {
	if len(params.NebiusBindings) > 0 && account.NebiusTenantID == nil {
		return errors.New("Nebius bindings can only be updated for Nebius accounts")
	}

	if params.PrivateArtifactPublicationEnabled != nil {
		if account.AwsAccountID == nil {
			return errors.Errorf("--%s is only supported for AWS accounts", privateArtifactPublicationEnabledFlag)
		}
		if len(account.ByoaInstanceIDs) > 0 {
			return errors.Errorf("--%s is not supported for BYOA accounts", privateArtifactPublicationEnabledFlag)
		}
	}

	return nil
}

func printNextStepPrivateArtifactPublicationMsg(account *openapiclient.DescribeAccountConfigResult) {
	name := account.Name
	if name == "" {
		name = "Unnamed Account"
	}

	var details string
	if account.StatusMessage != "" {
		details += fmt.Sprintf("\nStatus: %s (%s)", account.Status, account.StatusMessage)
	}
	if updateURL := utils.FromPtr(account.AwsCloudFormationUpdateURL); updateURL != "" {
		details += fmt.Sprintf("\nCloudFormation stack update URL: %s", updateURL)
	}

	fmt.Printf(nextStepPrivateArtifactPublicationMsgTemplate+"\n", name, details, account.Id)
}

func buildUpdateAccountParams(
	cmd *cobra.Command,
	accountNameOrID string,
	name string,
	description string,
	nebiusBindingsFile string,
	privateArtifactPublicationEnabled *bool,
) (UpdateCloudAccountParams, error) {
	params := UpdateCloudAccountParams{
		ID:                                strings.TrimSpace(accountNameOrID),
		PrivateArtifactPublicationEnabled: privateArtifactPublicationEnabled,
	}

	if cmd.Flags().Changed("name") {
		trimmedName := strings.TrimSpace(name)
		params.Name = &trimmedName
	}

	if cmd.Flags().Changed("description") {
		trimmedDescription := strings.TrimSpace(description)
		params.Description = &trimmedDescription
	}

	if nebiusBindingsFile != "" {
		bindings, err := parseNebiusBindingsFile(nebiusBindingsFile)
		if err != nil {
			return UpdateCloudAccountParams{}, err
		}
		params.NebiusBindings = bindings
	}

	customTags, _, err := parseAccountTags(cmd)
	if err != nil {
		return UpdateCloudAccountParams{}, err
	}
	params.CustomTags = customTags

	if err := validateUpdateAccountParams(params); err != nil {
		return UpdateCloudAccountParams{}, err
	}

	return params, nil
}

func validateUpdateAccountParams(params UpdateCloudAccountParams) error {
	if strings.TrimSpace(params.ID) == "" {
		return errors.New("account name or ID must be provided")
	}

	if params.Name != nil && strings.TrimSpace(*params.Name) == "" {
		return errors.New("name cannot be empty")
	}

	if params.Description != nil && strings.TrimSpace(*params.Description) == "" {
		return errors.New("description cannot be empty")
	}

	if params.Name == nil && params.Description == nil && len(params.NebiusBindings) == 0 && len(params.CustomTags) == 0 &&
		params.PrivateArtifactPublicationEnabled == nil {
		return errors.New("at least one of --name, --description, --nebius-bindings-file, --tags, or --private-artifact-publication-enabled must be provided")
	}

	return nil
}
