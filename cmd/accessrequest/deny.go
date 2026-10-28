package accessrequest

import (
	"fmt"

	"github.com/omnistrate-oss/omnistrate-ctl/cmd/common"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/config"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/dataaccess"
	"github.com/spf13/cobra"
)

var denyCmd = &cobra.Command{
	Use:          "deny [access-request-id]",
	Short:        "Deny a pending access request",
	Long:         "Deny and soft-delete a pending deployment-cell access request.",
	Args:         cobra.ExactArgs(1),
	RunE:         runDeny,
	SilenceUsage: true,
}

func runDeny(cmd *cobra.Command, args []string) error {
	defer config.CleanupArgsAndFlags(cmd, &args)

	accessRequestID, err := requiredValue("access request ID", args[0])
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
	result, err := dataaccess.DenyAccessRequest(cmd.Context(), token, accessRequestID)
	if err != nil {
		return fmt.Errorf("failed to deny access request: %w", err)
	}
	return printAccessRequest(output, result)
}
