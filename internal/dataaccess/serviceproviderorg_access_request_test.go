package dataaccess

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	openapiclient "github.com/omnistrate-oss/omnistrate-sdk-go/v1"
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

func TestResolveAccessRequestEnvironment(t *testing.T) {
	perEnvironment := true
	global := false

	for _, testCase := range []struct {
		name         string
		organization *openapiclient.DescribeServiceProviderOrganizationResult
		environment  string
		want         string
		wantErr      string
	}{
		{name: "explicit global", environment: " global ", want: "GLOBAL"},
		{name: "explicit prod", environment: "PROD", want: "PROD"},
		{name: "defaults per environment to prod", organization: &openapiclient.DescribeServiceProviderOrganizationResult{IsPerEnvClusterEnabled: &perEnvironment}, want: "PROD"},
		{name: "defaults shared cluster to global", organization: &openapiclient.DescribeServiceProviderOrganizationResult{IsPerEnvClusterEnabled: &global}, want: "GLOBAL"},
		{name: "rejects unsupported environment", environment: "DEV", wantErr: "GLOBAL or PROD"},
		{name: "requires explicit environment when mode missing", organization: &openapiclient.DescribeServiceProviderOrganizationResult{}, wantErr: "specify --environment-type"},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			actual, err := ResolveAccessRequestEnvironment(testCase.organization, testCase.environment)
			if testCase.wantErr != "" {
				require.ErrorContains(t, err, testCase.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, testCase.want, actual)
		})
	}
}

func TestAccessRequestApprovalConfigurationTypedManual(t *testing.T) {
	manual := "MANUAL"
	userID := "user-1"
	staleUserID := "user-stale"
	emptyUserID := ""
	configurations := map[string]openapiclient.JustInTimeAccessConfiguration{
		"GLOBAL": {
			ApprovalMode: &manual,
			ApproverUsers: []openapiclient.JustInTimeAccessApproverUser{
				{UserId: &userID},
				{},
				{UserId: &emptyUserID},
				{UserId: &staleUserID},
			},
		},
	}
	organization := &openapiclient.DescribeServiceProviderOrganizationResult{
		JustInTimeAccessConfigurations: &configurations,
	}
	users := []openapiclient.OrgUsers{{UserId: "user-1", Email: "approver@example.com", Name: "Approver"}}

	configuration, err := AccessRequestApprovalConfiguration(organization, users, "GLOBAL")
	require.NoError(t, err)
	assert.Equal(t, "GLOBAL", configuration.EnvironmentType)
	assert.Equal(t, "MANUAL", configuration.ApprovalMode)
	assert.False(t, configuration.Defaulted)
	require.Len(t, configuration.Approvers, 2)
	assert.Equal(t, "approver@example.com", configuration.Approvers[0].Email)
	assert.Equal(t, "user-stale", configuration.Approvers[1].UserID)
	assert.Empty(t, configuration.Approvers[1].Email)
}

func TestAccessRequestApprovalConfigurationTypedAuto(t *testing.T) {
	automatic := "AUTO"
	configurations := map[string]openapiclient.JustInTimeAccessConfiguration{
		"PROD": {
			ApprovalMode: &automatic,
			AdditionalProperties: map[string]any{
				"defaulted": true,
			},
		},
	}
	organization := &openapiclient.DescribeServiceProviderOrganizationResult{
		JustInTimeAccessConfigurations: &configurations,
	}

	configuration, err := AccessRequestApprovalConfiguration(organization, nil, "PROD")
	require.NoError(t, err)
	assert.Equal(t, "AUTO", configuration.ApprovalMode)
	assert.True(t, configuration.Defaulted)
	assert.Empty(t, configuration.Approvers)
}

func TestAccessRequestApprovalConfigurationLegacyFallback(t *testing.T) {
	organization := &openapiclient.DescribeServiceProviderOrganizationResult{
		AdditionalProperties: map[string]any{
			"justInTimeAccessConfigurations": map[string]any{
				"GLOBAL": map[string]any{
					"approvalMode": "MANUAL",
					"defaulted":    false,
					"approverUsers": []any{
						map[string]any{"userId": "user-1"},
						map[string]any{"userId": "user-stale"},
					},
				},
			},
		},
	}
	users := []openapiclient.OrgUsers{{UserId: "user-1", Email: "approver@example.com", Name: "Approver"}}

	configuration, err := AccessRequestApprovalConfiguration(organization, users, "GLOBAL")
	require.NoError(t, err)
	assert.Equal(t, "GLOBAL", configuration.EnvironmentType)
	assert.Equal(t, "MANUAL", configuration.ApprovalMode)
	assert.False(t, configuration.Defaulted)
	require.Len(t, configuration.Approvers, 2)
	assert.Equal(t, "approver@example.com", configuration.Approvers[0].Email)
	assert.Equal(t, "user-stale", configuration.Approvers[1].UserID)
	assert.Empty(t, configuration.Approvers[1].Email)

	defaulted, err := AccessRequestApprovalConfiguration(organization, users, "PROD")
	require.NoError(t, err)
	assert.Equal(t, "AUTO", defaulted.ApprovalMode)
	assert.True(t, defaulted.Defaulted)
	assert.Empty(t, defaulted.Approvers)
}

func TestAccessRequestApprovalConfigurationTypedFieldTakesPrecedence(t *testing.T) {
	automatic := "AUTO"
	configurations := map[string]openapiclient.JustInTimeAccessConfiguration{
		"GLOBAL": {ApprovalMode: &automatic},
	}
	organization := &openapiclient.DescribeServiceProviderOrganizationResult{
		JustInTimeAccessConfigurations: &configurations,
		AdditionalProperties: map[string]any{
			"justInTimeAccessConfigurations": map[string]any{
				"GLOBAL": map[string]any{
					"approvalMode":  "MANUAL",
					"approverUsers": []any{map[string]any{"userId": "legacy-user"}},
				},
			},
		},
	}

	configuration, err := AccessRequestApprovalConfiguration(organization, nil, "GLOBAL")
	require.NoError(t, err)
	assert.Equal(t, "AUTO", configuration.ApprovalMode)
	assert.False(t, configuration.Defaulted)
	assert.Empty(t, configuration.Approvers)
}

func TestResolveAccessRequestApproverEmails(t *testing.T) {
	users := []openapiclient.OrgUsers{
		{UserId: "user-1", Email: "first@example.com"},
		{UserId: "user-2", Email: "second@example.com"},
	}

	resolved, err := ResolveAccessRequestApproverEmails(users, []string{
		" First@Example.com ", "first@example.com", "second@example.com",
	})
	require.NoError(t, err)
	assert.Equal(t, []string{"user-1", "user-2"}, resolved)

	_, err = ResolveAccessRequestApproverEmails(users, []string{"missing@example.com"})
	require.ErrorContains(t, err, "no organization user found")

	users = append(users, openapiclient.OrgUsers{UserId: "user-3", Email: "FIRST@example.com"})
	_, err = ResolveAccessRequestApproverEmails(users, []string{"first@example.com"})
	require.ErrorContains(t, err, "multiple organization users")
}
