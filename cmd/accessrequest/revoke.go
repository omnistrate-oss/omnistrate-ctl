package accessrequest

import (
	"fmt"

	"github.com/omnistrate-oss/omnistrate-ctl/cmd/common"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/config"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/dataaccess"
	"github.com/spf13/cobra"
)

var revokeCmd = &cobra.Command{
	Use:          "revoke [access-request-id]",
	Short:        "Revoke an approved access request",
	Long:         "Revoke and soft-delete an approved deployment-cell access request.",
	Args:         cobra.ExactArgs(1),
	RunE:         runRevoke,
	SilenceUsage: true,
}

func runRevoke(cmd *cobra.Command, args []string) error {
	defer config.CleanupArgsAndFlags(cmd, &args)

	accessRequestID, err := requiredAccessRequestID(args[0])
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
	result, err := dataaccess.RevokeAccessRequest(cmd.Context(), token, accessRequestID)
	if err != nil {
		return fmt.Errorf("failed to revoke access request: %w", err)
	}
	return printAccessRequest(output, result)
}
