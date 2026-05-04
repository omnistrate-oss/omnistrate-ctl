package revoke

import (
	"fmt"

	"github.com/omnistrate-oss/omnistrate-ctl/internal/config"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/dataaccess"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/utils"
	"github.com/spf13/cobra"
)

// RevokeTokenCmd represents the revoke-token command
var RevokeTokenCmd = &cobra.Command{
	Use:   "revoke-token",
	Short: "Revoke the stored refresh token",
	Long: `The revoke-token command invalidates the stored refresh token on the
server and removes local credentials. After revocation the token can
never be used to obtain a new access token.

This is stronger than "logout" which only removes local credentials:
revoke-token also tells the server to delete the refresh token so it
cannot be replayed from another machine.`,
	Example:      `omnistrate-ctl revoke-token`,
	RunE:         runRevokeToken,
	SilenceUsage: true,
}

func runRevokeToken(cmd *cobra.Command, args []string) error {
	refreshToken, _ := config.GetRefreshToken()

	if refreshToken != "" {
		if err := dataaccess.RevokeToken(cmd.Context(), refreshToken); err != nil {
			// Server-side revocation failed — warn but still clean up locally.
			fmt.Fprintf(cmd.ErrOrStderr(), "Warning: server-side revocation failed: %v\n", err)
		}
	}

	if err := config.RemoveAuthConfig(); err != nil {
		utils.PrintError(err)
		return err
	}

	if refreshToken == "" {
		utils.PrintSuccess("No refresh token found; local credentials removed")
	} else {
		utils.PrintSuccess("Refresh token revoked and local credentials removed")
	}
	return nil
}
