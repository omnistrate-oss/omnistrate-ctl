package login

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/omnistrate-oss/omnistrate-ctl/internal/config"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/dataaccess"
	ctlutils "github.com/omnistrate-oss/omnistrate-ctl/internal/utils"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

// apiKeyLogin exchanges an org-bounded API key plaintext for a JWT
// session and persists it to the local auth config in the same shape
// as a password login. Reads the key from --api-key or, when
// --api-key-stdin is set, from stdin.
//
// Refresh tokens are not minted for api-key sessions: the api key
// itself is the long-lived credential and the platform expects clients
// to re-exchange it when the JWT expires. Callers SHOULD re-invoke
// `omnistrate-ctl login --api-key …` (or pipe via --api-key-stdin)
// from automation to obtain a fresh JWT rather than persisting the
// JWT longer than its TTL.
func apiKeyLogin(cmd *cobra.Command) error {
	if len(apiKey) > 0 {
		ctlutils.PrintWarning("Notice: Using the --api-key flag is insecure. Please consider using the --api-key-stdin flag instead. Refer to the help documentation for examples.")

		if apiKeyStdin {
			err := fmt.Errorf("--api-key and --api-key-stdin are mutually exclusive")
			ctlutils.PrintError(err)
			return err
		}
	}

	if apiKeyStdin {
		fromStdin, err := io.ReadAll(os.Stdin)
		if err != nil {
			ctlutils.PrintError(err)
			return err
		}
		apiKey = strings.TrimSpace(string(fromStdin))
	}

	apiKey = strings.TrimSpace(apiKey)
	if len(apiKey) == 0 {
		err := errors.New("must provide a non-empty api key via --api-key or --api-key-stdin")
		ctlutils.PrintError(err)
		return err
	}

	result, err := dataaccess.LoginWithAPIKey(cmd.Context(), apiKey)
	if err != nil {
		ctlutils.PrintError(err)
		return err
	}

	authConfig := config.AuthConfig{
		Token:        result.JWTToken,
		RefreshToken: result.RefreshToken,
	}
	if err = config.CreateOrUpdateAuthConfig(authConfig); err != nil {
		ctlutils.PrintError(err)
		return err
	}

	ctlutils.PrintSuccess("Successfully logged in with API key")
	return nil
}
