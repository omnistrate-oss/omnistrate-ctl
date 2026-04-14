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
