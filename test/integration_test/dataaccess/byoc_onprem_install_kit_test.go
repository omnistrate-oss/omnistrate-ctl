package dataaccess

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/omnistrate-oss/omnistrate-ctl/internal/dataaccess"
	"github.com/omnistrate-oss/omnistrate-ctl/test/testutils"
	"github.com/stretchr/testify/require"
)

func TestDownloadByocOnPremInstallKitIntegration(t *testing.T) {
	testutils.IntegrationTest(t)

	t.Setenv("OMNISTRATE_HOST_SCHEME", "http")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/2022-09-01-00/account-setup/byoc-onprem", r.URL.Path)
		require.Equal(t, "ac-integration", r.URL.Query().Get("account_config_id"))
		require.Equal(t, "Bearer token", r.Header.Get("Authorization"))
		_, _ = w.Write([]byte("integration-kit"))
	}))
	t.Cleanup(server.Close)
	t.Setenv("OMNISTRATE_HOST", server.Listener.Addr().String())

	var data bytes.Buffer
	err := dataaccess.DownloadByocOnPremInstallKit(context.Background(), "token", "ac-integration", &data)
	require.NoError(t, err)
	require.Equal(t, "integration-kit", data.String())
}
