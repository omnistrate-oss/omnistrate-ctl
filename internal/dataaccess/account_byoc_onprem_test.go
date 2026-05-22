package dataaccess

import (
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

	data, fileName, err := DownloadByocOnPremInstallKit(context.Background(), "token", "ac-123")
	require.NoError(t, err)
	require.Equal(t, []byte("tar-bytes"), data)
	require.Equal(t, "byoc-onprem-install-kit-ac-123.tar", fileName)
}

func TestDownloadByocOnPremInstallKitRequiresAccountConfigID(t *testing.T) {
	_, _, err := DownloadByocOnPremInstallKit(context.Background(), "token", "")
	require.ErrorContains(t, err, "account config ID is required")
}
