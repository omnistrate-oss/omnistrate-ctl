package login

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGetDeviceCodeURLForMicrosoftEntra(t *testing.T) {
	require.Equal(
		t,
		entraDeviceCodeURL,
		getDeviceCodeURL(identityProviderMicrosoftEntra),
	)
}

func TestGetClientIDForMicrosoftEntra(t *testing.T) {
	clientID := getClientID(identityProviderMicrosoftEntra)
	require.NotEmpty(t, clientID)
	require.Contains(t, []string{entraDevClientID, entraProdClientID}, clientID)
}

func TestGetVerificationURIForMicrosoftEntra(t *testing.T) {
	require.Equal(t, entraVerificationURI, getVerificationURI(identityProviderMicrosoftEntra))
}

func TestGetScopeForMicrosoftEntra(t *testing.T) {
	require.Equal(t, microsoftScope, getScope(identityProviderMicrosoftEntra))
}
