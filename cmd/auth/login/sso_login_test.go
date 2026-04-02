package login

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
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
	require.Equal(t, entraDevClientID, getClientID(identityProviderMicrosoftEntra))
}

func TestGetClientIDForMicrosoftEntraProd(t *testing.T) {
	require.Equal(t, entraProdClientID, getClientID(identityProviderMicrosoftEntra))
}

func TestGetVerificationURIForMicrosoftEntra(t *testing.T) {
	require.Equal(t, "https://microsoft.com/devicelogin", getVerificationURI(identityProviderMicrosoftEntra))
}

func TestGetScopeForMicrosoftEntra(t *testing.T) {
	require.Equal(t, "openid email profile offline_access User.Read", getScope(identityProviderMicrosoftEntra))
}

// redirectClient returns an http.Client that redirects all requests to the given test server.
func redirectClient(serverURL string) *http.Client {
	return &http.Client{
		Transport: redirectTransport(serverURL),
	}
}

type redirectTransport string

func (rt redirectTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	u, _ := url.Parse(string(rt) + req.URL.Path)
	req.URL = u
	return http.DefaultTransport.RoundTrip(req)
}

func TestRequestDeviceCodeEntraFormEncoded(t *testing.T) {
	var capturedContentType string
	var capturedBody string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedContentType = r.Header.Get("Content-Type")
		body, _ := io.ReadAll(r.Body)
		capturedBody = string(body)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"device_code":"dc_entra","user_code":"ENTRA-1234","expires_in":900,"interval":5}`))
	}))
	defer server.Close()

	resp, err := requestDeviceCodeWithHttpClient(context.Background(), redirectClient(server.URL), identityProviderMicrosoftEntra)
	require.NoError(t, err)
	require.Equal(t, "application/x-www-form-urlencoded", capturedContentType)
	require.Contains(t, capturedBody, "client_id=")
	require.Contains(t, capturedBody, "scope=")
	require.Equal(t, "dc_entra", resp.DeviceCode)
	require.Equal(t, "ENTRA-1234", resp.UserCode)
}

func TestRequestDeviceCodeGitHubJSON(t *testing.T) {
	var capturedContentType string
	var capturedBody string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedContentType = r.Header.Get("Content-Type")
		body, _ := io.ReadAll(r.Body)
		capturedBody = string(body)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"device_code":"dc_github","user_code":"GH-5678","expires_in":900,"interval":5}`))
	}))
	defer server.Close()

	resp, err := requestDeviceCodeWithHttpClient(context.Background(), redirectClient(server.URL), identityProviderGitHub)
	require.NoError(t, err)
	require.Equal(t, "application/json", capturedContentType)
	require.Contains(t, capturedBody, `"client_id"`)
	require.Contains(t, capturedBody, `"scope"`)
	require.Equal(t, "dc_github", resp.DeviceCode)
	require.Equal(t, "GH-5678", resp.UserCode)
}

func TestRequestDeviceCodeErrorStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer server.Close()

	_, err := requestDeviceCodeWithHttpClient(context.Background(), redirectClient(server.URL), identityProviderMicrosoftEntra)
	require.Error(t, err)
	require.Contains(t, err.Error(), "unexpected status")
}
