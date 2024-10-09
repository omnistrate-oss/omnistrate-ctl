package logout

import (
	"github.com/omnistrate/ctl/internal/utils"

	"github.com/omnistrate/ctl/internal/config"
	"github.com/spf13/cobra"
)

// LogoutCmd represents the logout command
var LogoutCmd = &cobra.Command{
	Use:          "logout",
	Short:        "Logout",
	Long:         `The logout command is used to log out from the Omnistrate platform.`,
	Example:      `omctl logout`,
	RunE:         runLogout,
	SilenceUsage: true,
}

func runLogout(cmd *cobra.Command, args []string) error {
	err := config.RemoveAuthConfig()
	if err != nil && err != config.ErrConfigFileNotFound {
		utils.PrintError(err)
		return err
	}
	utils.PrintSuccess("Credentials removed")

	return nil
}
