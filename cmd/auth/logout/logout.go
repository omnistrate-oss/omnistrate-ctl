package logout

import (
	"fmt"

	"github.com/omnistrate-oss/omnistrate-ctl/internal/config"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/dataaccess"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/utils"
	"github.com/spf13/cobra"
)

// LogoutCmd represents the logout command
var LogoutCmd = &cobra.Command{
	Use:          "logout",
	Short:        "Logout and revoke refresh token",
	Long:         `The logout command revokes the stored refresh token on the server and removes local credentials.`,
	Example:      `omnistrate-ctl logout`,
	RunE:         runLogout,
	SilenceUsage: true,
}

func runLogout(cmd *cobra.Command, args []string) error {
	refreshToken, err := config.GetRefreshToken()
	if err == nil && refreshToken != "" {
		if rErr := dataaccess.RevokeToken(cmd.Context(), refreshToken); rErr != nil {
			fmt.Fprintf(cmd.ErrOrStderr(), "Warning: server-side revocation failed: %v\n", rErr)
		}
	}

	err = config.RemoveAuthConfig()
	if err != nil && err != config.ErrConfigFileNotFound && err != config.ErrAuthConfigNotFound {
		utils.PrintError(err)
		return err
	}
	utils.PrintSuccess("Credentials removed")

	return nil
}
