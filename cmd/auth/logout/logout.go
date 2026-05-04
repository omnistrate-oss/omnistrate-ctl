package logout

import (
	"fmt"

	"github.com/omnistrate-oss/omnistrate-ctl/internal/config"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/dataaccess"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/utils"
	"github.com/spf13/cobra"
)

var skipRevoke bool

// LogoutCmd represents the logout command
var LogoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Logout and revoke refresh token",
	Long: `The logout command revokes the stored refresh token on the server
and removes local credentials. This ensures the token cannot be replayed
from another machine.

Use --skip-revoke to only remove local credentials without server-side
revocation (legacy behavior).`,
	Example:      `omnistrate-ctl logout`,
	RunE:         runLogout,
	SilenceUsage: true,
}

func init() {
	LogoutCmd.Flags().BoolVar(&skipRevoke, "skip-revoke", false,
		"Skip server-side token revocation; only remove local credentials")
}

func runLogout(cmd *cobra.Command, args []string) error {
	if !skipRevoke {
		refreshToken, err := config.GetRefreshToken()
		if err == nil && refreshToken != "" {
			if rErr := dataaccess.RevokeToken(cmd.Context(), refreshToken); rErr != nil {
				fmt.Fprintf(cmd.ErrOrStderr(), "Warning: server-side revocation failed: %v\n", rErr)
			}
		}
	}

	err := config.RemoveAuthConfig()
	if err != nil && err != config.ErrConfigFileNotFound && err != config.ErrAuthConfigNotFound {
		utils.PrintError(err)
		return err
	}
	utils.PrintSuccess("Credentials removed")

	return nil
}
