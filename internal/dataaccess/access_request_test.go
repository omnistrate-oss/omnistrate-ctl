package dataaccess

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/omnistrate-oss/omnistrate-ctl/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateAndListAccessRequests(t *testing.T) {
	var createRequest CreateAccessRequestRequest
	var listQuery url.Values
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "Bearer test-token", r.Header.Get("Authorization"))
		assert.NotEmpty(t, r.Header.Get("User-Agent"))
		w.Header().Set("Content-Type", "application/json")

		switch {
		case r.Method == http.MethodPost && r.URL.Path == accessRequestAPIPath:
			require.NoError(t, json.NewDecoder(r.Body).Decode(&createRequest))
			require.NoError(t, json.NewEncoder(w).Encode(model.AccessRequest{
				ID:               "ar-created",
				DeploymentCellID: createRequest.DeploymentCellID,
				Role:             createRequest.Role,
				Reason:           valueOrEmpty(createRequest.Reason),
				Status:           "PENDING",
			}))
		case r.Method == http.MethodGet && r.URL.Path == accessRequestAPIPath:
			listQuery = r.URL.Query()
			require.NoError(t, json.NewEncoder(w).Encode(model.AccessRequestList{Requests: []model.AccessRequest{{ID: "ar-created", Status: "PENDING"}}}))
		default:
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(server.Close)
	setAccessRequestTestHost(t, server.URL)

	reason := "Investigate deployment health"
	created, err := CreateAccessRequest(context.Background(), "test-token", CreateAccessRequestRequest{
		DeploymentCellID: "hc-12345678",
		Role:             "cluster-reader",
		Reason:           &reason,
	})
	require.NoError(t, err)
	assert.Equal(t, "ar-created", created.ID)
	assert.Equal(t, "hc-12345678", createRequest.DeploymentCellID)
	assert.Equal(t, "cluster-reader", createRequest.Role)
	require.NotNil(t, createRequest.Reason)
	assert.Equal(t, reason, *createRequest.Reason)

	listed, err := ListAccessRequests(context.Background(), "test-token", ListAccessRequestsOptions{
		Statuses:       []string{"PENDING", "APPROVED"},
		IncludeDeleted: true,
	})
	require.NoError(t, err)
	require.Len(t, listed.Requests, 1)
	assert.Equal(t, []string{"PENDING", "APPROVED"}, listQuery["statuses"])
	assert.Equal(t, "true", listQuery.Get("includeDeleted"))
}

func TestAccessRequestLifecycleActions(t *testing.T) {
	var approveRequest ApproveAccessRequestRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "Bearer test-token", r.Header.Get("Authorization"))
		w.Header().Set("Content-Type", "application/json")

		switch r.URL.Path {
		case accessRequestAPIPath + "/ar-described":
			assert.Equal(t, http.MethodGet, r.Method)
			require.NoError(t, json.NewEncoder(w).Encode(model.AccessRequest{ID: "ar-described", Status: "APPROVED"}))
		case accessRequestAPIPath + "/ar-approved/approve":
			assert.Equal(t, http.MethodPost, r.Method)
			require.NoError(t, json.NewDecoder(r.Body).Decode(&approveRequest))
			require.NoError(t, json.NewEncoder(w).Encode(model.AccessRequest{ID: "ar-approved", Status: "APPROVED"}))
		case accessRequestAPIPath + "/ar-denied/deny":
			assert.Equal(t, http.MethodPost, r.Method)
			assert.Equal(t, int64(0), r.ContentLength)
			require.NoError(t, json.NewEncoder(w).Encode(model.AccessRequest{ID: "ar-denied", Status: "DENIED"}))
		case accessRequestAPIPath + "/ar-revoked/revoke":
			assert.Equal(t, http.MethodPost, r.Method)
			assert.Equal(t, int64(0), r.ContentLength)
			require.NoError(t, json.NewEncoder(w).Encode(model.AccessRequest{ID: "ar-revoked", Status: "REVOKED"}))
		default:
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(server.Close)
	setAccessRequestTestHost(t, server.URL)

	described, err := DescribeAccessRequest(t.Context(), "test-token", "ar-described")
	require.NoError(t, err)
	assert.Equal(t, "APPROVED", described.Status)

	duration := 120
	approved, err := ApproveAccessRequest(t.Context(), "test-token", "ar-approved", ApproveAccessRequestRequest{DurationMinutes: &duration})
	require.NoError(t, err)
	assert.Equal(t, "APPROVED", approved.Status)
	require.NotNil(t, approveRequest.DurationMinutes)
	assert.Equal(t, 120, *approveRequest.DurationMinutes)

	denied, err := DenyAccessRequest(t.Context(), "test-token", "ar-denied")
	require.NoError(t, err)
	assert.Equal(t, "DENIED", denied.Status)

	revoked, err := RevokeAccessRequest(t.Context(), "test-token", "ar-revoked")
	require.NoError(t, err)
	assert.Equal(t, "REVOKED", revoked.Status)
}

func TestApproveAccessRequestOmitsDefaultDuration(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var requestBody map[string]any
		require.NoError(t, json.NewDecoder(r.Body).Decode(&requestBody))
		assert.NotContains(t, requestBody, "durationMinutes")
		w.Header().Set("Content-Type", "application/json")
		require.NoError(t, json.NewEncoder(w).Encode(model.AccessRequest{ID: "ar-approved", Status: "APPROVED"}))
	}))
	t.Cleanup(server.Close)
	setAccessRequestTestHost(t, server.URL)

	_, err := ApproveAccessRequest(t.Context(), "test-token", "ar-approved", ApproveAccessRequestRequest{})
	require.NoError(t, err)
}

func TestAccessRequestAPIErrors(t *testing.T) {
	for _, testCase := range []struct {
		name       string
		statusCode int
		body       string
		want       string
	}{
		{name: "structured error", statusCode: http.StatusConflict, body: `{"name":"invalid_state","message":"request is already approved"}`, want: "invalid_state\nDetail: request is already approved"},
		{name: "unstructured error", statusCode: http.StatusBadGateway, body: "gateway unavailable", want: "access request API returned 502: gateway unavailable"},
		{name: "empty success", statusCode: http.StatusOK, want: "without a JSON result"},
		{name: "invalid success", statusCode: http.StatusOK, body: "not-json", want: "failed to decode access request API response"},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(testCase.statusCode)
				_, _ = w.Write([]byte(testCase.body))
			}))
			t.Cleanup(server.Close)
			setAccessRequestTestHost(t, server.URL)

			result, err := DescribeAccessRequest(t.Context(), "test-token", "ar-failed")
			require.ErrorContains(t, err, testCase.want)
			assert.Nil(t, result)
		})
	}
}

func TestAccessRequestErrorClassification(t *testing.T) {
	assert.True(t, IsKubeConfigAccessRequestError(fmt.Errorf("forbidden\nDetail: access request is not approved")))
	assert.False(t, IsManualAccessReasonRequiredError(assert.AnError))

	manualReasonError := fmt.Errorf("bad_request\nDetail: manual access requires a nonblank reason")
	assert.False(t, IsKubeConfigAccessRequestError(manualReasonError))
	assert.True(t, IsManualAccessReasonRequiredError(manualReasonError))

	assert.False(t, IsKubeConfigAccessRequestError(fmt.Errorf("forbidden\nDetail: kubeconfig access is disabled")))
	assert.False(t, IsKubeConfigAccessRequestError(nil))

	assert.True(t, IsKubeConfigAccessRequestPendingError(
		fmt.Errorf("forbidden\nDetail: unauthorized: access request ar-test is not approved")))
	assert.False(t, IsKubeConfigAccessRequestPendingError(
		fmt.Errorf("forbidden\nDetail: access request role conflicts with the active request")))
	assert.False(t, IsKubeConfigAccessRequestPendingError(nil))
}

func valueOrEmpty(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func setAccessRequestTestHost(t *testing.T, rawURL string) {
	t.Helper()

	serverURL, err := url.Parse(rawURL)
	require.NoError(t, err)
	t.Setenv("OMNISTRATE_HOST", serverURL.Host)
	t.Setenv("OMNISTRATE_HOST_SCHEME", serverURL.Scheme)
	t.Setenv("OMNISTRATE_CLIENT_TIMEOUT_IN_SECONDS", "5")
	t.Setenv("OMNISTRATE_RETRY_MAX", "0")
}
