package dataaccess

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestImportAccountConfigCloudNativeNetworkDeploymentCellUsesFleetRoute(t *testing.T) {
	var requestMethod string
	var requestPath string
	var authorizationHeader string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestMethod = r.Method
		requestPath = r.URL.Path
		authorizationHeader = r.Header.Get("Authorization")

		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"hostClusterId":"hc-123","created":true}`)
	}))
	defer server.Close()

	t.Setenv("OMNISTRATE_HOST_SCHEME", "http")
	t.Setenv("OMNISTRATE_HOST", strings.TrimPrefix(server.URL, "http://"))

	result, err := ImportAccountConfigCloudNativeNetworkDeploymentCell(
		context.Background(),
		"test-token",
		"ac-test",
		"ap-south-1",
		"vpc-123",
		"deployment-cell-name",
	)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "hc-123", result.HostClusterID)
	assert.True(t, result.Created)
	assert.Equal(t, http.MethodPost, requestMethod)
	assert.Equal(t, "/2022-09-01-00/fleet/account-config/ac-test/cloud-native-networks/ap-south-1/vpc-123/host-clusters/deployment-cell-name/import", requestPath)
	assert.Equal(t, "Bearer test-token", authorizationHeader)
}
