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

// apiKeySource indicates how the API key was supplied.
type apiKeySource int

const (
	apiKeyFromFlag        apiKeySource = iota // --api-key (visible in ps/history)
	apiKeyFromStdin                           // --api-key-stdin or piped
	apiKeyFromEnv                             // OMNISTRATE_API_KEY env var
	apiKeyFromInteractive                     // interactive prompt (hidden input)
)

// apiKeyLogin exchanges an org-bounded API key plaintext for a JWT
// session and persists it to the local auth config in the same shape
// as a password login. Reads the key from --api-key, --api-key-stdin,
// OMNISTRATE_API_KEY, or the interactive prompt.
//
// Refresh tokens are not minted for api-key sessions: the api key
// itself is the long-lived credential and the platform expects clients
// to re-exchange it when the JWT expires.
func apiKeyLogin(cmd *cobra.Command, source apiKeySource) error {
	if len(apiKey) > 0 {
		// Warn only when the plaintext appears in the command line
		// (visible to ps, shell history, audit logs).
		if source == apiKeyFromFlag {
			ctlutils.PrintWarning("Notice: Using the --api-key flag is insecure. Please consider using OMNISTRATE_API_KEY or --api-key-stdin instead. Refer to the help documentation for examples.")
		}

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
		err := errors.New("must provide a non-empty api key via --api-key, --api-key-stdin, or OMNISTRATE_API_KEY")
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
