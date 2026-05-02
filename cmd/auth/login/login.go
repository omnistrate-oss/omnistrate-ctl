package login

import (
	"os"

	"github.com/omnistrate-oss/omnistrate-ctl/internal/utils"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

const omnistrateAPIKeyEnv = "OMNISTRATE_API_KEY"

type loginMethod string

const (
	loginExample = `# Select login method with a prompt
omnistrate-ctl login

# Login with email and password
omnistrate-ctl login --email email --password password

# Login with environment variables
  export OMNISTRATE_USER_NAME=YOUR_EMAIL
  export OMNISTRATE_PASSWORD=YOUR_PASSWORD
  ./omnistrate-ctl-darwin-arm64 login --email "$OMNISTRATE_USER_NAME" --password "$OMNISTRATE_PASSWORD"

# Login with email and password from stdin. Save the password in a file and use cat to read it
  cat ~/omnistrate_pass.txt | omnistrate-ctl login --email email --password-stdin

# Login with email and password from stdin. Save the password in an environment variable and use echo to read it
  echo $OMNISTRATE_PASSWORD | omnistrate-ctl login --email email --password-stdin

# Login with OMNISTRATE_API_KEY environment variable (recommended for CI/CD)
  export OMNISTRATE_API_KEY=om_…
  omnistrate-ctl login

# Login with an org-bounded API key (insecure; prefer env var or --api-key-stdin)
  omnistrate-ctl login --api-key om_…

# Login with an API key from stdin
  cat ~/omnistrate_apikey.txt | omnistrate-ctl login --api-key-stdin
  echo $OMNISTRATE_API_KEY | omnistrate-ctl login --api-key-stdin

# Login with GitHub SSO
  omnistrate-ctl login --gh

# Login with Google SSO
  omnistrate-ctl login --google

# Login with Microsoft Entra SSO
  omnistrate-ctl login --entra`

	loginWithEmailAndPassword loginMethod = "Login with email and password"
	loginWithAPIKey           loginMethod = "Login with API key" //nolint:gosec // UI label, not a credential
	loginWithGoogle           loginMethod = "Login with Google"
	loginWithGitHub           loginMethod = "Login with GitHub"
	loginWithEntra            loginMethod = "Login with Microsoft Entra"
)

var (
	email         string
	password      string
	passwordStdin bool
	apiKey        string
	apiKeyStdin   bool
	gh            bool
	google        bool
	entra         bool
)

// LoginCmd represents the login command
var LoginCmd = &cobra.Command{
	Use:          `login`,
	Short:        "Log in to the Omnistrate platform",
	Long:         `The login command is used to authenticate and log in to the Omnistrate platform.`,
	Example:      loginExample,
	RunE:         RunLogin,
	SilenceUsage: true,
}

func init() {
	LoginCmd.Flags().StringVarP(&email, "email", "", "", "email")
	LoginCmd.Flags().StringVarP(&password, "password", "", "", "password")
	LoginCmd.Flags().BoolVarP(&passwordStdin, "password-stdin", "", false, "Reads the password from stdin")

	LoginCmd.Flags().StringVarP(&apiKey, "api-key", "", "", "Org-bounded API key plaintext (om_…)")
	LoginCmd.Flags().BoolVarP(&apiKeyStdin, "api-key-stdin", "", false, "Reads the API key from stdin")

	LoginCmd.Flags().BoolVarP(&gh, "gh", "", false, "Login with GitHub")
	LoginCmd.Flags().BoolVarP(&google, "google", "", false, "Login with Google")
	LoginCmd.Flags().BoolVarP(&entra, "entra", "", false, "Login with Microsoft Entra")

	LoginCmd.MarkFlagsMutuallyExclusive("gh", "google", "entra", "email", "api-key", "api-key-stdin")
	LoginCmd.MarkFlagsMutuallyExclusive("gh", "google", "entra", "password", "api-key", "api-key-stdin")
	LoginCmd.MarkFlagsMutuallyExclusive("gh", "google", "entra", "password-stdin", "api-key", "api-key-stdin")
	LoginCmd.MarkFlagsMutuallyExclusive("api-key", "api-key-stdin")

	LoginCmd.Args = cobra.NoArgs
}

func RunLogin(cmd *cobra.Command, args []string) error {
	defer resetLogin()

	// Login with email and password if any of the flags are set
	if len(email) > 0 || len(password) > 0 || passwordStdin {
		return passwordLogin(cmd, false)
	}

	// Login with API key if any of the api-key flags are set
	if len(apiKey) > 0 || apiKeyStdin {
		source := apiKeyFromFlag
		if apiKeyStdin {
			source = apiKeyFromStdin
		}
		return apiKeyLogin(cmd, source)
	}

	// Auto-detect OMNISTRATE_API_KEY from environment when no explicit
	// flags are provided. This enables zero-flag CI/CD login:
	//   export OMNISTRATE_API_KEY=om_…
	//   omnistrate-ctl login
	if envKey := os.Getenv(omnistrateAPIKeyEnv); envKey != "" {
		apiKey = envKey
		return apiKeyLogin(cmd, apiKeyFromEnv)
	}

	if gh {
		return ssoLogin(cmd.Context(), identityProviderGitHub)
	}

	if google {
		return ssoLogin(cmd.Context(), identityProviderGoogle)
	}

	if entra {
		return ssoLogin(cmd.Context(), identityProviderMicrosoftEntra)
	}

	// Login interactively
	choice, err := utils.PromptSelect("How would you like to log in?", []string{
		string(loginWithEmailAndPassword),
		string(loginWithAPIKey),
		string(loginWithGoogle),
		string(loginWithGitHub),
		string(loginWithEntra),
	})
	if err != nil {
		utils.PrintError(err)
		return err
	}

	switch choice {
	case string(loginWithEmailAndPassword):
		email, err = utils.PromptInput("Please enter your email:", "Email", utils.ValidateEmail)
		if err != nil {
			utils.PrintError(err)
			return err
		}

		password, err = utils.PromptPassword("Please enter your password:", "Password")
		if err != nil {
			utils.PrintError(err)
			return err
		}

		return passwordLogin(cmd, true)
	case string(loginWithAPIKey):
		apiKey, err = utils.PromptPassword("Please enter your API key:", "API key")
		if err != nil {
			utils.PrintError(err)
			return err
		}
		return apiKeyLogin(cmd, apiKeyFromInteractive)
	case string(loginWithGoogle):
		return ssoLogin(cmd.Context(), identityProviderGoogle)
	case string(loginWithGitHub):
		return ssoLogin(cmd.Context(), identityProviderGitHub)
	case string(loginWithEntra):
		return ssoLogin(cmd.Context(), identityProviderMicrosoftEntra)

	default:
		err := errors.New("Invalid selection")
		utils.PrintError(err)
		return err
	}
}

func resetLogin() {
	email = ""
	password = ""
	passwordStdin = false
	apiKey = ""
	apiKeyStdin = false
	gh = false
	google = false
	entra = false
}
