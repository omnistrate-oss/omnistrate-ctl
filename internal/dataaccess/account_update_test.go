package dataaccess

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/omnistrate-oss/omnistrate-ctl/internal/config"
	openapiclient "github.com/omnistrate-oss/omnistrate-sdk-go/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUpdateAccount_ConstructsDirectRESTPayload(t *testing.T) {
	var capturedMethod string
	var capturedPath string
	var capturedAuth string
	var capturedBody map[string]any

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedMethod = r.Method
		capturedPath = r.URL.Path
		capturedAuth = r.Header.Get("Authorization")

		require.NoError(t, json.NewDecoder(r.Body).Decode(&capturedBody))

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		_, _ = w.Write([]byte(`"ac-123"`))
	}))
	defer server.Close()

	serverURL, err := url.Parse(server.URL)
	require.NoError(t, err)
	t.Setenv("OMNISTRATE_HOST", serverURL.Host)
	t.Setenv("OMNISTRATE_HOST_SCHEME", serverURL.Scheme)
	t.Setenv("CLIENT_TIMEOUT_IN_SECONDS", "5")

	accountID, err := UpdateAccount(context.Background(), "Bearer test-token", UpdateAccountParams{
		AccountConfigID: "ac-123",
		Name:            ptr("updated-name"),
		Description:     ptr("updated-description"),
		NebiusBindings: []openapiclient.NebiusAccountBindingInput{
			{
				ProjectID:        "project-1",
				ServiceAccountID: "service-account-1",
				PublicKeyID:      "public-key-1",
				PrivateKeyPEM:    "pem-data",
				Region:           "eu-north1",
			},
		},
	})
	require.NoError(t, err)
	assert.Equal(t, "ac-123", accountID)
	assert.Equal(t, http.MethodPut, capturedMethod)
	assert.Equal(t, "/2022-09-01-00/accountconfig/ac-123", capturedPath)
	assert.Equal(t, "Bearer test-token", capturedAuth)

	assert.Equal(t, "updated-name", capturedBody["name"])
	assert.Equal(t, "updated-description", capturedBody["description"])
	_, hasID := capturedBody["id"]
	assert.False(t, hasID)
	_, hasTenantID := capturedBody["nebiusTenantID"]
	assert.False(t, hasTenantID)

	_jsii, ok := capturedBody["nebiusBindings"].([]any)
	require.True(t, ok)
	require.Len(t, _jsii, 1)

	binding, ok := _jsii[0].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "project-1", binding["projectID"])
	assert.Equal(t, "service-account-1", binding["serviceAccountID"])
	assert.Equal(t, "public-key-1", binding["publicKeyID"])
	assert.Equal(t, "pem-data", binding["privateKeyPEM"])
	_, hasRegion := binding["region"]
	assert.False(t, hasRegion)

	assert.Equal(t, serverURL.Host, config.GetHost())
	assert.Equal(t, serverURL.Scheme, config.GetHostScheme())
}

func TestUpdateAccount_PropagatesAPIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(openapiclient.Error{
			Name:    "bad_request",
			Message: "invalid update payload",
		})
	}))
	defer server.Close()

	serverURL, err := url.Parse(server.URL)
	require.NoError(t, err)
	t.Setenv("OMNISTRATE_HOST", serverURL.Host)
	t.Setenv("OMNISTRATE_HOST_SCHEME", serverURL.Scheme)
	t.Setenv("CLIENT_TIMEOUT_IN_SECONDS", "5")

	_, err = UpdateAccount(context.Background(), "Bearer test-token", UpdateAccountParams{
		AccountConfigID: "ac-123",
		Name:            ptr("updated-name"),
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "bad_request")
	assert.Contains(t, err.Error(), "invalid update payload")
}

func ptr[T any](v T) *T {
	return &v
}
