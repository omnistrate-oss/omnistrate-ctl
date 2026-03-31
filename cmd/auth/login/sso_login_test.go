package login

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGetDeviceCodeURLForMicrosoftEntraDefaultTenant(t *testing.T) {
	t.Setenv(entraTenantEnv, "")
	require.Equal(
		t,
		"https://login.microsoftonline.com/common/oauth2/v2.0/devicecode",
		getDeviceCodeURL(identityProviderMicrosoftEntra),
	)
}

func TestGetDeviceCodeURLForMicrosoftEntraCustomTenant(t *testing.T) {
	t.Setenv(entraTenantEnv, "my-tenant-id")
	require.Equal(
		t,
		"https://login.microsoftonline.com/my-tenant-id/oauth2/v2.0/devicecode",
		getDeviceCodeURL(identityProviderMicrosoftEntra),
	)
}

func TestGetClientIDForMicrosoftEntraFromEnv(t *testing.T) {
	t.Setenv(entraClientIDEnv, "entra-client-id")
	require.Equal(t, "entra-client-id", getClientID(identityProviderMicrosoftEntra))
}

func TestGetVerificationURIForMicrosoftEntra(t *testing.T) {
	require.Equal(t, "https://microsoft.com/devicelogin", getVerificationURI(identityProviderMicrosoftEntra))
}

func TestGetScopeForMicrosoftEntra(t *testing.T) {
	require.Equal(t, "openid profile email offline_access", getScope(identityProviderMicrosoftEntra))
}

func TestGetClientIDForMicrosoftEntraUnset(t *testing.T) {
	_ = os.Unsetenv(entraClientIDEnv)
	require.Equal(t, "214069e3-8166-4283-8d89-a8378fe914c8", getClientID(identityProviderMicrosoftEntra))
}
