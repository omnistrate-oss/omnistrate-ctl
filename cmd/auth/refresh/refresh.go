package refresh

import (
	"fmt"

	"github.com/omnistrate-oss/omnistrate-ctl/internal/config"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/dataaccess"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/utils"
	"github.com/spf13/cobra"
)

// RefreshCmd represents the refresh command
var RefreshCmd = &cobra.Command{
	Use:   "refresh",
	Short: "Refresh the access token using the stored refresh token",
	Long: `The refresh command exchanges the stored refresh token for a new
JWT access token without requiring the user to re-enter credentials.

This is useful for testing the token refresh flow end-to-end and for
scripting scenarios where a fresh token is needed.`,
	Example:      `omnistrate-ctl refresh`,
	RunE:         runRefresh,
	SilenceUsage: true,
}

func runRefresh(cmd *cobra.Command, args []string) error {
	refreshToken, err := config.GetRefreshToken()
	if err != nil {
		return fmt.Errorf("no refresh token found — please login first: %w", err)
	}

	result, err := dataaccess.RefreshToken(cmd.Context(), refreshToken)
	if err != nil {
		utils.PrintError(err)
		return err
	}

	// Preserve existing refresh token if the server didn't return a new one
	persistedRefreshToken := result.RefreshToken
	if persistedRefreshToken == "" {
		persistedRefreshToken = refreshToken
	}

	authConfig := config.AuthConfig{
		Token:        result.JWTToken,
		RefreshToken: persistedRefreshToken,
	}
	if err := config.CreateOrUpdateAuthConfig(authConfig); err != nil {
		utils.PrintError(err)
		return err
	}

	utils.PrintSuccess("Access token refreshed successfully")
	return nil
}
