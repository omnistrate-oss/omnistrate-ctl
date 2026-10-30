package accessrequest

import (
	"fmt"

	"github.com/omnistrate-oss/omnistrate-ctl/cmd/common"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/config"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/dataaccess"
	"github.com/spf13/cobra"
)

var approveCmd = &cobra.Command{
	Use:   "approve [access-request-id]",
	Short: "Approve a pending access request",
	Long:  "Approve a pending access request with an optional fixed duration from 1 to 720 minutes.",
	Example: `# Approve using the backend's default duration
omnistrate-ctl access-request approve ar-12345678

# Approve for two hours
omnistrate-ctl access-request approve ar-12345678 --duration-minutes 120`,
	Args:         cobra.ExactArgs(1),
	RunE:         runApprove,
	SilenceUsage: true,
}

func init() {
	approveCmd.Flags().Int("duration-minutes", 0, "Access duration in minutes (1-720; omitted uses the backend default)")
}

func runApprove(cmd *cobra.Command, args []string) error {
	defer config.CleanupArgsAndFlags(cmd, &args)

	accessRequestID, err := requiredAccessRequestID(args[0])
	if err != nil {
		return err
	}
	duration, err := cmd.Flags().GetInt("duration-minutes")
	if err != nil {
		return err
	}
	if duration != 0 && (duration < 1 || duration > 720) {
		return fmt.Errorf("duration minutes must be between 1 and 720")
	}
	output, err := outputFormat(cmd)
	if err != nil {
		return err
	}

	var durationPointer *int
	if duration != 0 {
		durationPointer = &duration
	}
	token, err := common.GetTokenWithLogin()
	if err != nil {
		return fmt.Errorf("failed to get user token: %w", err)
	}
	result, err := dataaccess.ApproveAccessRequest(cmd.Context(), token, accessRequestID, dataaccess.ApproveAccessRequestRequest{
		DurationMinutes: durationPointer,
	})
	if err != nil {
		return fmt.Errorf("failed to approve access request: %w", err)
	}
	return printAccessRequest(output, result)
}
