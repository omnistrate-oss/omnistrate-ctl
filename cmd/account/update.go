package account

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/omnistrate-oss/omnistrate-ctl/cmd/common"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/config"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/dataaccess"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/utils"
	openapiclient "github.com/omnistrate-oss/omnistrate-sdk-go/v1"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

const (
	privateArtifactPushEnabledFlag = "private-artifact-push-enabled"

	updateExample = `# Update account name and description
omnistrate-ctl account update [account-name or account-id] --name=[new-name] --description=[new-description]

# Replace Nebius bindings on an existing Nebius account
omnistrate-ctl account update [account-name or account-id] --nebius-bindings-file=[bindings-file]

# Enable private managed-artifact push for an AWS non-BYOA account
omnistrate-ctl account update [account-name or account-id] --private-artifact-push-enabled=true

# Disable private managed-artifact push (use = with false)
omnistrate-ctl account update [account-name or account-id] --private-artifact-push-enabled=false`

	nextStepPrivateArtifactPushStackUpdateMsgTemplate = `
Next step:
Update the existing AWS CloudFormation stack for account %s to apply the private artifact push setting; do not create a new one.%s
Track progress with: omnistrate-ctl account describe %s`

	privateArtifactPushPendingMsgTemplate = `
The private artifact push setting for account %s is pending verification.%s
If verification reports that the setting is not applied, update the existing AWS CloudFormation stack; do not create a new one.
Track progress with: omnistrate-ctl account describe %s`

	privateArtifactPushUnverifiedMsgTemplate = `
Account %s finished verification, but it does not report the requested private artifact push value, so the setting could not be confirmed.%s
If the account stack predates this setting, update the existing AWS CloudFormation stack; do not create a new one.
Check the applied value with: omnistrate-ctl account describe %s`

	// Matches commons accountconfig settings.SettingsPendingStatusMessage, which the
	// server reports when the account is verified but its stack must still be updated.
	accountSettingsPendingStatusMessage = "account verified; waiting for requested settings"

	privateArtifactPushWaitTimeout  = 10 * time.Minute
	privateArtifactPushPollInterval = 10 * time.Second
)

type privateArtifactPushState int

const (
	privateArtifactPushPending privateArtifactPushState = iota
	privateArtifactPushApplied
	privateArtifactPushStackUpdateRequired
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
	updateCmd.Flags().Bool(privateArtifactPushEnabledFlag, false, "Whether an AWS non-BYOA account may push managed artifacts to private ECR")
	updateCmd.Flags().Bool("skip-wait", false, "Skip waiting for account to become READY after replacing Nebius bindings or changing private artifact push")
	_ = updateCmd.MarkFlagFilename("nebius-bindings-file")
}

type UpdateCloudAccountParams struct {
	ID                         string
	Name                       *string
	Description                *string
	NebiusBindings             []openapiclient.NebiusAccountBindingInput
	CustomTags                 []openapiclient.CustomTag
	PrivateArtifactPushEnabled *bool
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
	privateArtifactPushEnabled, err := optionalBoolFlag(cmd, privateArtifactPushEnabledFlag)
	if err != nil {
		utils.PrintError(err)
		return err
	}

	params, err := buildUpdateAccountParams(
		cmd, args[0], name, description, nebiusBindingsFile, privateArtifactPushEnabled,
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
	if len(params.NebiusBindings) > 0 || params.PrivateArtifactPushEnabled != nil {
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
		AccountConfigID:            params.ID,
		Name:                       params.Name,
		Description:                params.Description,
		NebiusBindings:             params.NebiusBindings,
		CustomTags:                 params.CustomTags,
		PrivateArtifactPushEnabled: params.PrivateArtifactPushEnabled,
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

	if params.PrivateArtifactPushEnabled != nil && !skipWait {
		var waitSpinner *utils.Spinner
		if output != "json" {
			fmt.Printf("\n")
			sm = utils.NewSpinnerManager()
			waitSpinner = sm.AddSpinner("Waiting for the private artifact push setting to be verified...")
			sm.Start()
		}

		var state privateArtifactPushState
		state, err = waitForPrivateArtifactPushSetting(
			cmd.Context(), token, accountConfigID, *params.PrivateArtifactPushEnabled,
		)
		if err != nil {
			err = errors.Wrap(err, "account update was submitted, but the private artifact push setting was not verified")
			utils.HandleSpinnerError(waitSpinner, sm, err)
			return err
		}

		switch state {
		case privateArtifactPushApplied:
			utils.HandleSpinnerSuccess(waitSpinner, sm, "Private artifact push setting is applied")
		case privateArtifactPushStackUpdateRequired:
			utils.HandleSpinnerSuccess(waitSpinner, sm, "Account verified; existing stack update required")
		default:
			utils.HandleSpinnerSuccess(waitSpinner, sm, "Account verified; requested value not reported")
		}
	}

	account, err := dataaccess.DescribeAccount(cmd.Context(), token, accountConfigID)
	if err != nil {
		utils.PrintError(err)
		return err
	}

	if params.PrivateArtifactPushEnabled != nil {
		return printPrivateArtifactPushResult(output, account, *params.PrivateArtifactPushEnabled)
	}

	if err = printDescribeOutput(output, account); err != nil {
		utils.PrintError(err)
		return err
	}

	if output != "json" && account.Status != "READY" {
		dataaccess.PrintNextStepVerifyAccountMsg(account)
	}

	return nil
}

func printPrivateArtifactPushResult(
	output string, account *openapiclient.DescribeAccountConfigResult, enabled bool,
) error {
	state, err := getPrivateArtifactPushState(account, enabled)
	if err != nil {
		err = errors.Wrap(err, "account update was submitted, but the private artifact push setting was not applied")
		utils.PrintError(err)
		return err
	}

	nextStepWriter := io.Writer(os.Stdout)
	if output == "json" {
		if err = printDescribeOutput(output, account); err != nil {
			utils.PrintError(err)
			return err
		}
		// Keep stdout valid JSON while still warning about an unconfirmed setting.
		nextStepWriter = os.Stderr
	} else {
		// Print a static snapshot so the interactive view cannot hide the next step.
		snapshot := renderAccountDescribeSnapshot(account)
		utils.LastPrintedString = snapshot
		fmt.Println(snapshot)
	}

	if state != privateArtifactPushApplied {
		printPrivateArtifactPushNextStepMsg(nextStepWriter, account, state)
	}

	return nil
}

func waitForPrivateArtifactPushSetting(
	ctx context.Context, token, accountID string, enabled bool,
) (privateArtifactPushState, error) {
	// The deadline also cancels an in-flight describe request.
	waitCtx, cancel := context.WithTimeout(ctx, privateArtifactPushWaitTimeout)
	defer cancel()
	ticker := time.NewTicker(privateArtifactPushPollInterval)
	defer ticker.Stop()

	waitErr := func() error {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		return errors.Errorf(
			"account %s did not finish verifying the setting after %s; check it with 'omnistrate-ctl account describe %s'",
			accountID, privateArtifactPushWaitTimeout, accountID)
	}

	for {
		select {
		case <-waitCtx.Done():
			return privateArtifactPushPending, waitErr()
		case <-ticker.C:
			account, err := dataaccess.DescribeAccount(waitCtx, token, accountID)
			if err != nil {
				if waitCtx.Err() != nil {
					return privateArtifactPushPending, waitErr()
				}
				return privateArtifactPushPending, err
			}

			state, err := getPrivateArtifactPushState(account, enabled)
			// The update request moves the account out of READY, so READY means verification finished.
			if err != nil || state != privateArtifactPushPending || account.Status == "READY" {
				return state, err
			}
		}
	}
}

func getPrivateArtifactPushState(
	account *openapiclient.DescribeAccountConfigResult, enabled bool,
) (privateArtifactPushState, error) {
	switch account.Status {
	case "FAILED":
		return privateArtifactPushPending, errors.Errorf("account settings update failed: %s", account.StatusMessage)
	case "READY":
		// Success requires an observed value; applied settings are never defaulted.
		if applied, ok := appliedPrivateArtifactPushEnabled(account); ok && applied == enabled {
			return privateArtifactPushApplied, nil
		}
	case "PENDING_UPDATE":
		if account.StatusMessage == accountSettingsPendingStatusMessage {
			return privateArtifactPushStackUpdateRequired, nil
		}
	}

	return privateArtifactPushPending, nil
}

// appliedPrivateArtifactPushEnabled reports whether the account observed the setting.
func appliedPrivateArtifactPushEnabled(account *openapiclient.DescribeAccountConfigResult) (bool, bool) {
	if account.AppliedSettings == nil || account.AppliedSettings.AccountGovernanceControls == nil {
		return false, false
	}

	enabled, ok := account.AppliedSettings.AccountGovernanceControls.
		AdditionalProperties[dataaccess.PrivateArtifactPushEnabledSettingKey].(bool)
	return enabled, ok
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

	if params.PrivateArtifactPushEnabled != nil {
		if account.AwsAccountID == nil {
			return errors.Errorf("--%s is only supported for AWS accounts", privateArtifactPushEnabledFlag)
		}
		if len(account.ByoaInstanceIDs) > 0 {
			return errors.Errorf("--%s is not supported for BYOA accounts", privateArtifactPushEnabledFlag)
		}
	}

	return nil
}

func printPrivateArtifactPushNextStepMsg(
	w io.Writer, account *openapiclient.DescribeAccountConfigResult, state privateArtifactPushState,
) {
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

	template := privateArtifactPushPendingMsgTemplate
	switch {
	case state == privateArtifactPushStackUpdateRequired:
		template = nextStepPrivateArtifactPushStackUpdateMsgTemplate
	case account.Status == "READY":
		template = privateArtifactPushUnverifiedMsgTemplate
	}

	_, _ = fmt.Fprintf(w, template+"\n", name, details, account.Id)
}

func buildUpdateAccountParams(
	cmd *cobra.Command,
	accountNameOrID string,
	name string,
	description string,
	nebiusBindingsFile string,
	privateArtifactPushEnabled *bool,
) (UpdateCloudAccountParams, error) {
	params := UpdateCloudAccountParams{
		ID:                         strings.TrimSpace(accountNameOrID),
		PrivateArtifactPushEnabled: privateArtifactPushEnabled,
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
		params.PrivateArtifactPushEnabled == nil {
		return errors.New("at least one of --name, --description, --nebius-bindings-file, --tags, or --private-artifact-push-enabled must be provided")
	}

	return nil
}
