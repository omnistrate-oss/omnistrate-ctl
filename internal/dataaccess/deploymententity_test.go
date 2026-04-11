package dataaccess

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLocalDeploymentAgentURLEscapesPathSegments(t *testing.T) {
	url := localDeploymentAgentURL("terraform", "instance/123", "deployment name")

	require.Equal(t, "http://localhost:80/2022-09-01-00/terraform/instance%2F123/deployment%20name", url)
}

func TestDoLocalDeploymentAgentRequestRejectsNonLocalHost(t *testing.T) {
	request, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "http://example.com/path", nil)
	require.NoError(t, err)

	response, err := doLocalDeploymentAgentRequest(http.DefaultClient, request)
	if response != nil {
		defer response.Body.Close()
	}
	require.Error(t, err)
	require.Contains(t, err.Error(), "non-local deployment agent URL")
}
