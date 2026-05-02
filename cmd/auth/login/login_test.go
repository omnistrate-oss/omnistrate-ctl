package login

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/omnistrate-oss/omnistrate-ctl/internal/config"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/dataaccess"
	"github.com/stretchr/testify/require"
)

func TestLoginCommandHasEntraFlag(t *testing.T) {
	flag := LoginCmd.Flags().Lookup("entra")
	require.NotNil(t, flag)
	require.Equal(t, "bool", flag.Value.Type())
	require.Equal(t, "Login with Microsoft Entra", flag.Usage)
}

func TestLoginCommandIncludesEntraChoice(t *testing.T) {
	require.Contains(t, loginExample, "--entra")
	require.Equal(t, loginMethod("Login with Microsoft Entra"), loginWithEntra)
}

// TestLoginCommandHasAPIKeyFlag asserts the public --api-key flag is
// registered as a string flag with the documented help text. The flag
// is the entry point for the org-bounded api-key signin-exchange path
// (api_key_login.go); a typo or removal would silently break every
// scripted/CI caller.
func TestLoginCommandHasAPIKeyFlag(t *testing.T) {
	flag := LoginCmd.Flags().Lookup("api-key")
	require.NotNil(t, flag)
	require.Equal(t, "string", flag.Value.Type())
	require.Equal(t, "Org-bounded API key plaintext (om_…)", flag.Usage)
}

// TestLoginCommandHasAPIKeyStdinFlag asserts the safer --api-key-stdin
// alternative is wired and documented; this is the CI-recommended
// path (no plaintext on the command line / shell history).
func TestLoginCommandHasAPIKeyStdinFlag(t *testing.T) {
	flag := LoginCmd.Flags().Lookup("api-key-stdin")
	require.NotNil(t, flag)
	require.Equal(t, "bool", flag.Value.Type())
	require.Equal(t, "Reads the API key from stdin", flag.Usage)
}

// TestLoginCommandIncludesAPIKeyChoice asserts the interactive prompt
// (RunLogin → PromptStringWithChoices) includes the api-key option,
// and that the help/example block documents the --api-key flag — both
// surfaces where a regression would silently hide the feature.
func TestLoginCommandIncludesAPIKeyChoice(t *testing.T) {
	require.Equal(t, loginMethod("Login with API key"), loginWithAPIKey)
	require.Contains(t, loginExample, "--api-key")
	require.Contains(t, loginExample, "--api-key-stdin")
}

// TestLoginCommandAPIKeyMutuallyExclusive asserts the api-key flags
// cannot be combined with any other login method or with each other.
// We only check the cobra-level wiring (each annotation pair is
// declared) rather than running the command — a mismatch would let a
// caller silently combine, e.g., --google with --api-key, producing
// undefined behavior.
func TestLoginCommandAPIKeyMutuallyExclusive(t *testing.T) {
	for _, name := range []string{
		"api-key", "api-key-stdin",
	} {
		flag := LoginCmd.Flags().Lookup(name)
		require.NotNilf(t, flag, "flag %s must be registered", name)
		// All exclusivity groups are recorded as annotations on each
		// participating flag; presence of the cobra group key proves the
		// MarkFlagsMutuallyExclusive call was made.
		require.NotEmptyf(t, flag.Annotations["cobra_annotation_mutually_exclusive"],
			"flag %s must participate in at least one mutually-exclusive group", name)
	}
}

// TestLoginCommandOmnistrateAPIKeyEnvConst asserts the environment
// variable name is defined and matches the documented value.
func TestLoginCommandOmnistrateAPIKeyEnvConst(t *testing.T) {
	require.Equal(t, "OMNISTRATE_API_KEY", config.OmnistrateAPIKeyEnv)
}

// TestLoginCommandExampleDocumentsEnvVar ensures the help/example text
// documents the OMNISTRATE_API_KEY env var login flow.
func TestLoginCommandExampleDocumentsEnvVar(t *testing.T) {
	require.Contains(t, loginExample, "OMNISTRATE_API_KEY")
	require.Contains(t, loginExample, "export OMNISTRATE_API_KEY")
}

// TestAPIKeyLoginEmptyKeyErrorMessageBySource validates that the
// empty-key guard in apiKeyLogin produces a source-appropriate error
// message. When the key comes from the interactive prompt the message
// must not instruct the user to supply flags or env vars.
func TestAPIKeyLoginEmptyKeyErrorMessageBySource(t *testing.T) {
	// PrintError calls os.Exit(1) unless dry-run mode is enabled.
	t.Setenv("OMNISTRATE_DRY_RUN", "true")

	tests := []struct {
		name            string
		source          apiKeySource
		expectedMessage string
	}{
		{
			name:            "flag source",
			source:          apiKeyFromFlag,
			expectedMessage: "must provide a non-empty API key via --api-key",
		},
		{
			name:            "stdin source",
			source:          apiKeyFromStdin,
			expectedMessage: "must provide a non-empty API key via --api-key-stdin",
		},
		{
			name:            "env var source",
			source:          apiKeyFromEnv,
			expectedMessage: "must provide a non-empty API key via OMNISTRATE_API_KEY",
		},
		{
			name:            "interactive source",
			source:          apiKeyFromInteractive,
			expectedMessage: "must provide a non-empty API key",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetLogin()
			// apiKey is empty after reset — triggers the empty-key guard.
			err := apiKeyLogin(LoginCmd, tt.source)
			require.Error(t, err)
			require.Contains(t, err.Error(), tt.expectedMessage)
		})
	}

	// Extra: interactive source must NOT suggest flags or the env var.
	t.Run("interactive message is prompt-appropriate", func(t *testing.T) {
		resetLogin()
		err := apiKeyLogin(LoginCmd, apiKeyFromInteractive)
		require.Error(t, err)
		require.NotContains(t, err.Error(), "--api-key")
		require.NotContains(t, err.Error(), "OMNISTRATE_API_KEY")
	})
}

// signinCapture records the fields from the most recent captured signin
// request body.
type signinCapture struct {
	Email    string
	Password string
}

// newFakeSigninServer starts an httptest.Server that:
//   - handles any POST request by reading the JSON body and capturing
//     the "email" and "password" fields,
//   - returns a canned successful signin response so the caller's SDK
//     can parse it without error.
//
// The server is automatically stopped when the test ends.
func newFakeSigninServer(t *testing.T) (*httptest.Server, *signinCapture) {
	t.Helper()
	captured := &signinCapture{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "read error", http.StatusInternalServerError)
			return
		}
		var req struct {
			Email    string `json:"email"`
			Password string `json:"password"`
		}
		if err = json.Unmarshal(body, &req); err != nil {
			http.Error(w, "bad JSON", http.StatusBadRequest)
			t.Errorf("newFakeSigninServer: failed to unmarshal request body: %v", err)
			return
		}
		captured.Email = req.Email
		captured.Password = req.Password

		w.Header().Set("Content-Type", "application/json")
		if _, err = io.WriteString(w, `{"jwtToken":"fake-jwt-for-testing"}`); err != nil {
			t.Errorf("newFakeSigninServer: failed to write response: %v", err)
		}
	}))
	t.Cleanup(srv.Close)
	return srv, captured
}

// pointClientAt redirects all dataaccess HTTP calls to the given
// httptest server for the duration of the current test. It also:
//   - disables retries so tests do not wait on backoff delays,
//   - redirects HOME to a throwaway temp dir to prevent any config
//     writes from reaching the developer's real ~/.omnistrate,
//   - enables dry-run mode so that PrintError does not call os.Exit(1)
//     if an unexpected error occurs during the test.
func pointClientAt(t *testing.T, srv *httptest.Server) {
	t.Helper()
	u, err := url.Parse(srv.URL)
	require.NoError(t, err)
	t.Setenv("OMNISTRATE_HOST", u.Host)
	t.Setenv("OMNISTRATE_HOST_SCHEME", u.Scheme)
	t.Setenv("OMNISTRATE_RETRY_MAX", "0")
	t.Setenv("HOME", t.TempDir())
	t.Setenv("OMNISTRATE_DRY_RUN", "true")
}

// TestRunLoginPicksUpEnvVar validates that RunLogin uses
// OMNISTRATE_API_KEY when no flags are provided. The fake signin server
// captures the request body so we can assert that the env var value
// was forwarded as the password field.
func TestRunLoginPicksUpEnvVar(t *testing.T) {
	srv, captured := newFakeSigninServer(t)
	pointClientAt(t, srv)
	t.Setenv("OMNISTRATE_API_KEY", "om_test_env_key")

	resetLogin()

	err := RunLogin(LoginCmd, nil)
	require.NoError(t, err)
	require.Equal(t, dataaccess.APIKeySigninEmail, captured.Email,
		"env-var login must use the API-key sentinel email")
	require.Equal(t, "om_test_env_key", captured.Password,
		"env var value must be forwarded as the request password")
}

// TestRunLoginFlagTakesPrecedenceOverEnv verifies that the --api-key
// flag value is sent to the signin endpoint even when OMNISTRATE_API_KEY
// is also set, proving the flag takes priority over the env var.
func TestRunLoginFlagTakesPrecedenceOverEnv(t *testing.T) {
	srv, captured := newFakeSigninServer(t)
	pointClientAt(t, srv)
	t.Setenv("OMNISTRATE_API_KEY", "om_env_should_be_ignored")

	resetLogin()
	apiKey = "om_flag_value"

	err := RunLogin(LoginCmd, nil)
	require.NoError(t, err)
	require.Equal(t, "om_flag_value", captured.Password,
		"--api-key flag value must take precedence over the env var")
}

// TestRunLoginEnvVarNotUsedWhenOtherFlagsSet verifies that the env var
// is NOT consulted when email/password flags are present. The fake
// server captures the request so we can assert the correct email and
// password were sent rather than the API-key sentinel.
func TestRunLoginEnvVarNotUsedWhenOtherFlagsSet(t *testing.T) {
	srv, captured := newFakeSigninServer(t)
	pointClientAt(t, srv)
	t.Setenv("OMNISTRATE_API_KEY", "om_should_not_be_used")

	resetLogin()
	email = "test@example.com"
	password = "fake_password"

	err := RunLogin(LoginCmd, nil)
	require.NoError(t, err)
	require.Equal(t, "test@example.com", captured.Email,
		"email/password path must send the user's email, not the API-key sentinel")
	require.Equal(t, "fake_password", captured.Password,
		"email/password path must send the password flag, not the env var API key")
}
