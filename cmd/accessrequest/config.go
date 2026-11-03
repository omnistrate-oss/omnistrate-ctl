package accessrequest

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/omnistrate-oss/omnistrate-ctl/cmd/common"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/config"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/dataaccess"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/model"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/utils"
	"github.com/spf13/cobra"
)

var configCmd = &cobra.Command{
	Use:          "config [operation] [flags]",
	Short:        "View or update JIT access approval configuration",
	Args:         cobra.NoArgs,
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, _ []string) error {
		return cmd.Help()
	},
}

var configGetCmd = &cobra.Command{
	Use:          "get",
	Short:        "View JIT access approval configuration",
	Args:         cobra.NoArgs,
	RunE:         runConfigGet,
	SilenceUsage: true,
}

var configSetCmd = &cobra.Command{
	Use:   "set",
	Short: "Update JIT access approval configuration",
	Long:  "Update JIT access approval configuration and display the resulting configuration.",
	Example: `# Require manual approval using provider-user emails
omnistrate-ctl access-request config set --approval-mode MANUAL --approver-email approver@example.com

# Restore automatic approval
omnistrate-ctl access-request config set --approval-mode AUTO`,
	Args:         cobra.NoArgs,
	RunE:         runConfigSet,
	SilenceUsage: true,
}

func init() {
	configGetCmd.Flags().String("environment-type", "", "Environment type to inspect (GLOBAL or PROD; defaults from organization configuration)")
	configSetCmd.Flags().String("environment-type", "", "Environment type to configure (GLOBAL or PROD; defaults from organization configuration)")
	configSetCmd.Flags().String("approval-mode", "", "Approval mode (AUTO or MANUAL)")
	configSetCmd.Flags().StringSlice("approver-email", nil, "Provider-user email allowed to approve, deny, or revoke (repeatable; required for MANUAL)")
	configCmd.AddCommand(configGetCmd, configSetCmd)
}

func runConfigGet(cmd *cobra.Command, args []string) error {
	defer config.CleanupArgsAndFlags(cmd, &args)

	environment, err := cmd.Flags().GetString("environment-type")
	if err != nil {
		return err
	}
	output, err := outputFormat(cmd)
	if err != nil {
		return err
	}
	token, err := common.GetTokenWithLogin()
	if err != nil {
		return fmt.Errorf("failed to get user token: %w", err)
	}
	configuration, err := dataaccess.GetAccessRequestApprovalConfiguration(cmd.Context(), token, environment)
	if err != nil {
		return fmt.Errorf("failed to get JIT access approval configuration: %w", err)
	}
	return printAccessRequestApprovalConfiguration(output, configuration)
}

func runConfigSet(cmd *cobra.Command, args []string) error {
	defer config.CleanupArgsAndFlags(cmd, &args)

	environment, err := cmd.Flags().GetString("environment-type")
	if err != nil {
		return err
	}
	mode, err := cmd.Flags().GetString("approval-mode")
	if err != nil {
		return err
	}
	approverEmails, err := cmd.Flags().GetStringSlice("approver-email")
	if err != nil {
		return err
	}
	mode, approverEmails, err = validateAccessRequestConfiguration(mode, approverEmails)
	if err != nil {
		return err
	}
	output, err := outputFormat(cmd)
	if err != nil {
		return err
	}

	token, err := common.GetTokenWithLogin()
	if err != nil {
		return fmt.Errorf("failed to get user token: %w", err)
	}
	organization, err := dataaccess.GetServiceProviderOrganization(cmd.Context(), token)
	if err != nil {
		return fmt.Errorf("failed to get service-provider organization: %w", err)
	}
	environment, err = dataaccess.ResolveAccessRequestEnvironment(organization, environment)
	if err != nil {
		return err
	}

	var approverUserIDs []string
	if mode == dataaccess.AccessApprovalModeManual {
		users, err := dataaccess.GetOrganizationUsers(cmd.Context(), token)
		if err != nil {
			return fmt.Errorf("failed to list organization users: %w", err)
		}
		approverUserIDs, err = dataaccess.ResolveAccessRequestApproverEmails(users, approverEmails)
		if err != nil {
			return err
		}
	}
	configuration, err := configureAndReadAccessRequestApproval(
		cmd.Context(), token, environment, mode, approverUserIDs)
	if err != nil {
		return err
	}
	return printAccessRequestApprovalConfiguration(output, configuration)
}

func configureAndReadAccessRequestApproval(
	ctx context.Context,
	token string,
	environment string,
	mode string,
	approverUserIDs []string,
) (*model.AccessRequestApprovalConfiguration, error) {
	if err := dataaccess.ConfigureAccessRequestApproval(
		ctx, token, environment, mode, approverUserIDs); err != nil {
		return nil, fmt.Errorf("failed to configure JIT access approval: %w", err)
	}

	configuration, err := dataaccess.GetAccessRequestApprovalConfiguration(ctx, token, environment)
	if err != nil {
		return nil, fmt.Errorf(
			"JIT access approval configuration was updated successfully, but verification failed: %w", err)
	}
	if err = verifyAccessRequestApprovalConfiguration(configuration, mode, approverUserIDs); err != nil {
		return nil, fmt.Errorf(
			"JIT access approval configuration was updated successfully, but verification failed: %w", err)
	}
	return configuration, nil
}

func verifyAccessRequestApprovalConfiguration(
	configuration *model.AccessRequestApprovalConfiguration,
	mode string,
	approverUserIDs []string,
) error {
	if configuration == nil {
		return fmt.Errorf("readback returned no configuration")
	}

	requestedMode := strings.ToUpper(strings.TrimSpace(mode))
	persistedMode := strings.ToUpper(strings.TrimSpace(configuration.ApprovalMode))
	if persistedMode != requestedMode {
		return fmt.Errorf(
			"persisted approval mode %q does not match requested mode %q",
			persistedMode,
			requestedMode,
		)
	}

	persistedApproverUserIDs := make([]string, 0, len(configuration.Approvers))
	for _, approver := range configuration.Approvers {
		persistedApproverUserIDs = append(persistedApproverUserIDs, approver.UserID)
	}
	requestedApproverUserIDs := normalizedAccessRequestApproverUserIDs(approverUserIDs)
	persistedApproverUserIDs = normalizedAccessRequestApproverUserIDs(persistedApproverUserIDs)
	if !slices.Equal(persistedApproverUserIDs, requestedApproverUserIDs) {
		return fmt.Errorf(
			"persisted approver user IDs %v do not match requested user IDs %v",
			persistedApproverUserIDs,
			requestedApproverUserIDs,
		)
	}
	return nil
}

func normalizedAccessRequestApproverUserIDs(userIDs []string) []string {
	unique := make(map[string]struct{}, len(userIDs))
	for _, userID := range userIDs {
		userID = strings.TrimSpace(userID)
		if userID != "" {
			unique[userID] = struct{}{}
		}
	}

	normalized := make([]string, 0, len(unique))
	for userID := range unique {
		normalized = append(normalized, userID)
	}
	slices.Sort(normalized)
	return normalized
}

func validateAccessRequestConfiguration(mode string, approverEmails []string) (string, []string, error) {
	mode = strings.ToUpper(strings.TrimSpace(mode))
	if mode != dataaccess.AccessApprovalModeAuto && mode != dataaccess.AccessApprovalModeManual {
		return "", nil, fmt.Errorf("approval mode must be AUTO or MANUAL")
	}

	cleanApprovers := make([]string, 0, len(approverEmails))
	seen := make(map[string]bool, len(approverEmails))
	for _, input := range approverEmails {
		email := strings.TrimSpace(input)
		normalized := strings.ToLower(email)
		if email != "" && !seen[normalized] {
			seen[normalized] = true
			cleanApprovers = append(cleanApprovers, email)
		}
	}
	if mode == dataaccess.AccessApprovalModeManual && len(cleanApprovers) == 0 {
		return "", nil, fmt.Errorf("MANUAL approval requires at least one --approver-email")
	}
	if mode == dataaccess.AccessApprovalModeAuto && len(cleanApprovers) > 0 {
		return "", nil, fmt.Errorf("--approver-email is only valid with MANUAL approval")
	}
	return mode, cleanApprovers, nil
}

type accessRequestApprovalConfigurationOutput struct {
	EnvironmentType string   `json:"environmentType"`
	ApprovalMode    string   `json:"approvalMode"`
	ApproverEmails  []string `json:"approverEmails,omitempty"`
}

func printAccessRequestApprovalConfiguration(
	output string,
	configuration *model.AccessRequestApprovalConfiguration,
) error {
	formatted := formatAccessRequestApprovalConfiguration(configuration)
	if output == "table" {
		newAccessRequestApprovalConfigurationTable(formatted).Print()
		return nil
	}
	return utils.PrintTextTableJsonOutput(output, formatted)
}

func newAccessRequestApprovalConfigurationTable(
	configuration accessRequestApprovalConfigurationOutput,
) *utils.Table {
	columns := []string{"environmentType", "approvalMode"}
	row := []any{
		configuration.EnvironmentType,
		configuration.ApprovalMode,
	}
	if len(configuration.ApproverEmails) > 0 {
		columns = append(columns, "approverEmails")
		row = append(row, strings.Join(configuration.ApproverEmails, ", "))
	}
	table := utils.NewTableWithColumnOrder(columns)
	table.AddRow(row)
	return table
}

func formatAccessRequestApprovalConfiguration(
	configuration *model.AccessRequestApprovalConfiguration,
) accessRequestApprovalConfigurationOutput {
	approverEmails := make([]string, 0, len(configuration.Approvers))
	for _, approver := range configuration.Approvers {
		if approver.Email != "" {
			approverEmails = append(approverEmails, approver.Email)
		} else if approver.UserID != "" {
			approverEmails = append(approverEmails, approver.UserID)
		}
	}
	return accessRequestApprovalConfigurationOutput{
		EnvironmentType: configuration.EnvironmentType,
		ApprovalMode:    configuration.ApprovalMode,
		ApproverEmails:  approverEmails,
	}
}
