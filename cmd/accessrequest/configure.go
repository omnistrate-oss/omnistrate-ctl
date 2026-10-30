package accessrequest

import (
	"fmt"
	"strings"

	"github.com/omnistrate-oss/omnistrate-ctl/cmd/common"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/config"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/dataaccess"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/utils"
	"github.com/spf13/cobra"
)

const (
	accessApprovalModeAuto   = "AUTO"
	accessApprovalModeManual = "MANUAL"
	accessEnvironmentGlobal  = "GLOBAL"
	accessEnvironmentProd    = "PROD"
)

var configureCmd = &cobra.Command{
	Use:   "configure",
	Short: "Configure deployment-cell access approval",
	Long:  "Configure AUTO or MANUAL deployment-cell access approval.",
	Example: `# Require manual approval in PROD
omnistrate-ctl access-request configure --environment-type PROD --approval-mode MANUAL --approver-user-id user-123 --approver-user-id user-456`,
	Args:         cobra.NoArgs,
	RunE:         runConfigure,
	SilenceUsage: true,
}

func init() {
	configureCmd.Flags().String("environment-type", "", "Environment type to configure")
	configureCmd.Flags().String("approval-mode", "", "Approval mode (AUTO or MANUAL)")
	configureCmd.Flags().StringSlice("approver-user-id", nil, "Provider user ID allowed to approve, deny, or revoke (repeatable; required for MANUAL)")
}

func runConfigure(cmd *cobra.Command, args []string) error {
	defer config.CleanupArgsAndFlags(cmd, &args)

	environment, err := cmd.Flags().GetString("environment-type")
	if err != nil {
		return err
	}
	mode, err := cmd.Flags().GetString("approval-mode")
	if err != nil {
		return err
	}
	approverUserIDs, err := cmd.Flags().GetStringSlice("approver-user-id")
	if err != nil {
		return err
	}
	environment, mode, approverUserIDs, err = validateAccessRequestConfiguration(environment, mode, approverUserIDs)
	if err != nil {
		return err
	}

	token, err := common.GetTokenWithLogin()
	if err != nil {
		return fmt.Errorf("failed to get user token: %w", err)
	}
	if err = dataaccess.ConfigureAccessRequestApproval(
		cmd.Context(), token, environment, mode, approverUserIDs); err != nil {
		return fmt.Errorf("failed to configure access request approval: %w", err)
	}
	utils.PrintSuccess(fmt.Sprintf("Configured %s deployment-cell access for %s approval.", environment, mode))
	return nil
}

func validateAccessRequestConfiguration(environment, mode string, approverUserIDs []string) (string, string, []string, error) {
	environment = strings.ToUpper(strings.TrimSpace(environment))
	if environment != accessEnvironmentGlobal && environment != accessEnvironmentProd {
		return "", "", nil, fmt.Errorf("unsupported environment type")
	}
	mode = strings.ToUpper(strings.TrimSpace(mode))
	if mode != accessApprovalModeAuto && mode != accessApprovalModeManual {
		return "", "", nil, fmt.Errorf("approval mode must be AUTO or MANUAL")
	}

	cleanApprovers := make([]string, 0, len(approverUserIDs))
	seen := make(map[string]bool)
	for _, userID := range approverUserIDs {
		userID = strings.TrimSpace(userID)
		if userID != "" && !seen[userID] {
			seen[userID] = true
			cleanApprovers = append(cleanApprovers, userID)
		}
	}
	if mode == accessApprovalModeManual && len(cleanApprovers) == 0 {
		return "", "", nil, fmt.Errorf("MANUAL approval requires at least one --approver-user-id")
	}
	if mode == accessApprovalModeAuto && len(cleanApprovers) > 0 {
		return "", "", nil, fmt.Errorf("--approver-user-id is only valid with MANUAL approval")
	}
	return environment, mode, cleanApprovers, nil
}
