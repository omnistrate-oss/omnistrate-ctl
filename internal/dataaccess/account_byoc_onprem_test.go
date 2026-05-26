package dataaccess

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDownloadByocOnPremInstallKit(t *testing.T) {
	t.Setenv("OMNISTRATE_HOST_SCHEME", "http")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/2022-09-01-00/account-setup/byoc-onprem", r.URL.Path)
		require.Equal(t, "ac-123", r.URL.Query().Get("account_config_id"))
		require.Equal(t, "Bearer token", r.Header.Get("Authorization"))
		_, _ = w.Write([]byte("tar-bytes"))
	}))
	t.Cleanup(server.Close)
	t.Setenv("OMNISTRATE_HOST", server.Listener.Addr().String())

	var data bytes.Buffer
	err := DownloadByocOnPremInstallKit(context.Background(), "token", "ac-123", &data)
	require.NoError(t, err)
	require.Equal(t, "tar-bytes", data.String())
	require.Equal(t, "byoc-onprem-install-kit-ac-123.tar", ByocOnPremInstallKitFileName("ac-123"))
}

func TestDownloadByocOnPremInstallKitRequiresAccountConfigID(t *testing.T) {
	var data bytes.Buffer
	err := DownloadByocOnPremInstallKit(context.Background(), "token", "", &data)
	require.ErrorContains(t, err, "account config ID is required")
}

func TestDownloadByocOnPremInstallKitRequiresWriter(t *testing.T) {
	err := DownloadByocOnPremInstallKit(context.Background(), "token", "ac-123", nil)
	require.ErrorContains(t, err, "install kit writer is required")
}
