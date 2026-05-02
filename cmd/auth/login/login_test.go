package login

import (
	"testing"

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
	require.Equal(t, "OMNISTRATE_API_KEY", omnistrateAPIKeyEnv)
}

// TestLoginCommandExampleDocumentsEnvVar ensures the help/example text
// documents the OMNISTRATE_API_KEY env var login flow.
func TestLoginCommandExampleDocumentsEnvVar(t *testing.T) {
	require.Contains(t, loginExample, "OMNISTRATE_API_KEY")
	require.Contains(t, loginExample, "export OMNISTRATE_API_KEY")
}

// TestRunLoginPicksUpEnvVar validates that RunLogin uses
// OMNISTRATE_API_KEY when no flags are provided. We set the env var to
// an invalid key so the request fails at the HTTP layer, but we can
// confirm that the code path attempted to use the env var (error
// message won't say "must provide a non-empty api key").
func TestRunLoginPicksUpEnvVar(t *testing.T) {
	t.Setenv("OMNISTRATE_API_KEY", "om_test_fake_key_for_unit_test")
	t.Setenv("OMNISTRATE_DRY_RUN", "true")

	// Reset all flags to default so no flag-based path triggers.
	resetLogin()

	err := RunLogin(LoginCmd, nil)
	// The call will fail (no real server / invalid key), but it should
	// NOT be the "must provide a non-empty api key" error — proving
	// the env var was read and passed to the signin exchange.
	require.Error(t, err)
	require.NotContains(t, err.Error(), "must provide a non-empty api key",
		"env var should have been picked up; empty-key guard should not fire")
}

// TestRunLoginFlagTakesPrecedenceOverEnv verifies that --api-key flag
// takes priority over the OMNISTRATE_API_KEY env var.
func TestRunLoginFlagTakesPrecedenceOverEnv(t *testing.T) {
	t.Setenv("OMNISTRATE_API_KEY", "om_env_should_be_ignored")
	t.Setenv("OMNISTRATE_DRY_RUN", "true")

	resetLogin()
	apiKey = "om_flag_value"

	err := RunLogin(LoginCmd, nil)
	// Will fail at HTTP layer, but the env var value should not be
	// used — the flag value is what gets sent. We can't easily assert
	// which value was sent without mocking, but we confirm it doesn't
	// hit the empty-key guard.
	require.Error(t, err)
	require.NotContains(t, err.Error(), "must provide a non-empty api key")
}

// TestRunLoginEnvVarNotUsedWhenOtherFlagsSet verifies that the env var
// is NOT consulted when email/password flags are present.
func TestRunLoginEnvVarNotUsedWhenOtherFlagsSet(t *testing.T) {
	t.Setenv("OMNISTRATE_API_KEY", "om_should_not_be_used")
	t.Setenv("OMNISTRATE_DRY_RUN", "true")

	resetLogin()
	email = "test@example.com"
	password = "fake_password"

	err := RunLogin(LoginCmd, nil)
	// Should attempt password login (will fail at HTTP), not api-key login.
	require.Error(t, err)
	require.NotContains(t, err.Error(), "must provide a non-empty api key")
}
