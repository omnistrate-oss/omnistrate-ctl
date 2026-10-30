package dataaccess

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfigureAccessRequestApproval(t *testing.T) {
	for _, testCase := range []struct {
		name              string
		environment       string
		mode              string
		approverUserIDs   []string
		expectedApprovers []any
	}{
		{
			name:              "manual",
			environment:       "GLOBAL",
			mode:              "MANUAL",
			approverUserIDs:   []string{"user-1", "user-2"},
			expectedApprovers: []any{map[string]any{"userId": "user-1"}, map[string]any{"userId": "user-2"}},
		},
		{
			name:              "automatic clears approvers",
			environment:       "PROD",
			mode:              "AUTO",
			expectedApprovers: []any{},
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, http.MethodPatch, r.Method)
				assert.Equal(t, "/2022-09-01-00/sp-organization", r.URL.Path)
				assert.Equal(t, "Bearer test-token", r.Header.Get("Authorization"))

				var requestBody map[string]any
				require.NoError(t, json.NewDecoder(r.Body).Decode(&requestBody))
				configurations, ok := requestBody["justInTimeAccessConfigurations"].(map[string]any)
				require.True(t, ok)
				configuration, ok := configurations[testCase.environment].(map[string]any)
				require.True(t, ok)
				assert.Equal(t, testCase.mode, configuration["approvalMode"])
				assert.Equal(t, testCase.expectedApprovers, configuration["approverUsers"])
				w.WriteHeader(http.StatusNoContent)
			}))
			t.Cleanup(server.Close)
			setAccessRequestTestHost(t, server.URL)

			err := ConfigureAccessRequestApproval(
				t.Context(), "test-token", testCase.environment, testCase.mode, testCase.approverUserIDs)
			require.NoError(t, err)
		})
	}
}
