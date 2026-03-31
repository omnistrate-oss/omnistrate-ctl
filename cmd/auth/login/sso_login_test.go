package login

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGetDeviceCodeURLForMicrosoftEntra(t *testing.T) {
	require.Equal(
		t,
		"https://login.microsoftonline.com/organizations/oauth2/v2.0/devicecode",
		getDeviceCodeURL(identityProviderMicrosoftEntra),
	)
}

func TestGetClientIDForMicrosoftEntraDev(t *testing.T) {
	t.Setenv("OMNISTRATE_ROOT_DOMAIN", "dev.omnistrate.cloud")
	require.Equal(t, "3a09381f-919b-40d5-ac1e-3ad35297a438", getClientID(identityProviderMicrosoftEntra))
}

func TestGetClientIDForMicrosoftEntraProd(t *testing.T) {
	require.Equal(t, "8ca18dc3-470b-44bd-995b-cb4f6f298514", getClientID(identityProviderMicrosoftEntra))
}

func TestGetVerificationURIForMicrosoftEntra(t *testing.T) {
	require.Equal(t, "https://microsoft.com/devicelogin", getVerificationURI(identityProviderMicrosoftEntra))
}

func TestGetScopeForMicrosoftEntra(t *testing.T) {
	require.Equal(t, "openid email profile offline_access User.Read", getScope(identityProviderMicrosoftEntra))
}
