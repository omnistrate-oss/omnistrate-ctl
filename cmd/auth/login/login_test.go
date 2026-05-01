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
