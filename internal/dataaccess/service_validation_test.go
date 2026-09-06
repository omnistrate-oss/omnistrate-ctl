package dataaccess

// SDK wire-contract evidence for specification 02 —
// dry-run-implementation-specs/02-regression-tests.md.
//
// These tests pin the request serialisation of the two calls that
// `omnistrate-ctl build --dry-run` makes today against a local httptest.Server.
// They are deliberately untagged and green: they document the current contract
// so that spec 04 cannot change it by accident, and they supply the
// deterministic-request-count and no-real-network guarantees that 02 requires of
// the CLI suite.
//
// The read-only ValidateServiceSpec wrapper specified in 03/04 does not exist
// yet. Its CLI-level contract is asserted from cmd/build/dry_run_target_test.go
// behind the `dryrun_target` build tag; there is nothing here to call until the
// generated SDK ships the new operation.

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync"
	"testing"

	openapiclient "github.com/omnistrate-oss/omnistrate-sdk-go/v1"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/stretchr/testify/require"
)

const validationTestToken = "dry-run-wire-token" //nolint:gosec // G101: fixed fake token used only by the local test server

type capturedRequest struct {
	Method string
	Path   string
	Auth   string
	Body   map[string]any
	Raw    []byte
}

type validationTestServer struct {
	mu       sync.Mutex
	requests []capturedRequest
	// violations is written from the handler goroutine and asserted on the main
	// goroutine; the handler never calls require.FailNow.
	violations []string
	respond    func(n int, w http.ResponseWriter)
}

func (s *validationTestServer) record(w http.ResponseWriter, r *http.Request) {
	raw, err := io.ReadAll(r.Body)

	s.mu.Lock()
	if err != nil {
		s.violations = append(s.violations, "failed to read request body: "+err.Error())
	}
	captured := capturedRequest{
		Method: r.Method,
		Path:   r.URL.Path,
		Auth:   r.Header.Get("Authorization"),
		Raw:    raw,
	}
	if len(raw) > 0 {
		if decodeErr := json.Unmarshal(raw, &captured.Body); decodeErr != nil {
			s.violations = append(s.violations, "request body is not JSON: "+decodeErr.Error())
		}
	}
	s.requests = append(s.requests, captured)
	n := len(s.requests) - 1
	respond := s.respond
	s.mu.Unlock()

	respond(n, w)
}

func (s *validationTestServer) snapshot() []capturedRequest {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]capturedRequest, len(s.requests))
	copy(out, s.requests)
	return out
}

func (s *validationTestServer) assertNoViolations(t *testing.T) {
	t.Helper()
	s.mu.Lock()
	violations := append([]string(nil), s.violations...)
	s.mu.Unlock()
	require.Empty(t, violations)
}

// startValidationWireServer points the SDK clients at a local server and turns
// retries off so request counts are exact.
func startValidationWireServer(t *testing.T, respond func(n int, w http.ResponseWriter)) *validationTestServer {
	t.Helper()

	srv := &validationTestServer{respond: respond}
	httpServer := httptest.NewServer(http.HandlerFunc(srv.record))
	t.Cleanup(httpServer.Close)

	parsed, err := url.Parse(httpServer.URL)
	require.NoError(t, err)

	t.Setenv("OMNISTRATE_HOST", parsed.Host)
	t.Setenv("OMNISTRATE_HOST_SCHEME", parsed.Scheme)
	t.Setenv("OMNISTRATE_RETRY_MAX", "0")
	t.Setenv("OMNISTRATE_RETRY_WAIT_MIN_IN_SECONDS", "0")
	t.Setenv("OMNISTRATE_RETRY_WAIT_MAX_IN_SECONDS", "0")
	t.Setenv("OMNISTRATE_CLIENT_TIMEOUT_IN_SECONDS", "20")

	return srv
}

func okJSON(payload any) func(int, http.ResponseWriter) {
	return func(_ int, w http.ResponseWriter) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(payload)
	}
}

// TestPrepareServicePlanSpecDryRunValidationHasNoDryRunField documents the root
// cause recorded in ctl-build-dry-run-investigation.md: the prepare request that
// CTL sends before a dry run cannot express that it is only validating, so the
// backend legitimately treats it as a real preparation.
func TestPrepareServicePlanSpecDryRunValidationHasNoDryRunField(t *testing.T) {
	srv := startValidationWireServer(t, okJSON(map[string]any{
		"serviceID":               "s-1",
		"serviceEnvironmentID":    "se-1",
		"productTierID":           "pt-1",
		"isNewProductTierCreated": false,
	}))

	spec := []byte("name: plan\n")
	_, err := PrepareServiceFromServicePlanSpec(context.Background(), validationTestToken,
		openapiclient.PrepareServiceFromServicePlanSpecRequest2{
			Name:            "Wire Contract Service",
			Environment:     "Dev",
			EnvironmentType: "DEV",
			FileContent:     base64.StdEncoding.EncodeToString(spec),
		})
	require.NoError(t, err)
	srv.assertNoViolations(t)

	requests := srv.snapshot()
	require.Len(t, requests, 1, "retries are disabled, so exactly one request is expected")
	require.Equal(t, http.MethodPut, requests[0].Method)
	require.Equal(t, "/2022-09-01-00/service/serviceplanspec/prepare", requests[0].Path)
	require.Equal(t, "Bearer "+validationTestToken, requests[0].Auth)

	for key := range requests[0].Body {
		require.NotContains(t, key, "dryrun")
		require.NotContains(t, key, "dryRun")
	}
	require.Equal(t, base64.StdEncoding.EncodeToString(spec), requests[0].Body["fileContent"])
}

// TestBuildServicePlanSpecDryRunValidationForwardsDryrunFlag pins the wire name
// and value of the flag the SDK sends for a ServicePlanSpec dry run.
func TestBuildServicePlanSpecDryRunValidationForwardsDryrunFlag(t *testing.T) {
	srv := startValidationWireServer(t, okJSON(map[string]any{
		"serviceID":            "s-1",
		"serviceEnvironmentID": "se-1",
		"productTierID":        "pt-1",
	}))

	spec := []byte("name: plan\n")
	_, err := BuildServiceFromServicePlanSpec(context.Background(), validationTestToken,
		openapiclient.BuildServiceFromServicePlanSpecRequest2{
			Name:               "Wire Contract Service",
			FileContent:        base64.StdEncoding.EncodeToString(spec),
			Dryrun:             ptr(true),
			Release:            ptr(false),
			ReleaseAsPreferred: ptr(false),
		})
	require.NoError(t, err)
	srv.assertNoViolations(t)

	requests := srv.snapshot()
	require.Len(t, requests, 1)
	require.Equal(t, http.MethodPut, requests[0].Method)
	require.Equal(t, "/2022-09-01-00/service/serviceplanspec", requests[0].Path)
	require.Equal(t, true, requests[0].Body["dryrun"])
	require.Equal(t, false, requests[0].Body["release"], "false flags must be sent explicitly")
	require.Equal(t, false, requests[0].Body["releaseAsPreferred"])
}

// TestBuildComposeSpecDryRunValidationForwardsDryrunFlag is the Compose
// counterpart, including the base64 configs/secrets convention that 03 reuses.
func TestBuildComposeSpecDryRunValidationForwardsDryrunFlag(t *testing.T) {
	srv := startValidationWireServer(t, okJSON(map[string]any{
		"serviceID":            "s-1",
		"serviceEnvironmentID": "se-1",
		"productTierID":        "pt-1",
	}))

	configs := map[string]string{"app": base64.StdEncoding.EncodeToString([]byte("conf"))}
	secrets := map[string]string{"app": base64.StdEncoding.EncodeToString([]byte("secret"))}

	_, err := BuildServiceFromComposeSpec(context.Background(), validationTestToken,
		openapiclient.BuildServiceFromComposeSpecRequest2{
			Name:        "Wire Contract Service",
			FileContent: base64.StdEncoding.EncodeToString([]byte("services: {}\n")),
			Dryrun:      ptr(true),
			Configs:     &configs,
			Secrets:     &secrets,
		})
	require.NoError(t, err)
	srv.assertNoViolations(t)

	requests := srv.snapshot()
	require.Len(t, requests, 1)
	require.Equal(t, http.MethodPut, requests[0].Method)
	require.Equal(t, "/2022-09-01-00/service/composespec", requests[0].Path)
	require.Equal(t, true, requests[0].Body["dryrun"])

	sentConfigs, ok := requests[0].Body["configs"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, configs["app"], sentConfigs["app"])

	sentSecrets, ok := requests[0].Body["secrets"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, secrets["app"], sentSecrets["app"])
}

// TestDryRunValidationRetriesAreDisabledForDeterministicCounts proves the test
// configuration used by the whole 02 CLI suite really does stop retryablehttp
// from repeating a request. Without this, "exactly one validation request"
// assertions would be meaningless.
func TestDryRunValidationRetriesAreDisabledForDeterministicCounts(t *testing.T) {
	srv := startValidationWireServer(t, func(_ int, w http.ResponseWriter) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusServiceUnavailable)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id":        "test-error",
			"name":      "unavailable",
			"message":   "try later",
			"fault":     false,
			"temporary": true,
			"timeout":   false,
		})
	})

	_, err := BuildServiceFromServicePlanSpec(context.Background(), validationTestToken,
		openapiclient.BuildServiceFromServicePlanSpecRequest2{
			Name:        "Wire Contract Service",
			FileContent: base64.StdEncoding.EncodeToString([]byte("name: plan\n")),
			Dryrun:      ptr(true),
		})
	require.Error(t, err)
	srv.assertNoViolations(t)

	require.Len(t, srv.snapshot(), 1, "OMNISTRATE_RETRY_MAX=0 must suppress all retries")
}

// ---------------------------------------------------------------------------
// ValidateServiceSpec — the read-only route added by spec 04
// ---------------------------------------------------------------------------

func validationOKResult() map[string]any {
	return map[string]any{
		"status":             "VALID",
		"validationVersion":  "1",
		"inputDigest":        "0000000000000000000000000000000000000000000000000000000000000000",
		"checks":             []map[string]any{{"name": "syntax", "status": "PASSED"}},
		"diagnostics":        []map[string]any{},
		"requiredArtifacts":  []map[string]any{},
		"validatedArtifacts": []map[string]any{},
	}
}

// TestValidateServiceSpecWireContract pins the request serialisation of the new
// operation: method, path, auth header, wire field names, explicit false flags
// and the artifact envelope from 04 §"Wire types".
func TestValidateServiceSpecWireContract(t *testing.T) {
	srv := startValidationWireServer(t, okJSON(validationOKResult()))

	spec := []byte("name: plan\n")
	result, err := ValidateServiceSpec(context.Background(), validationTestToken,
		openapiclient.ValidateServiceSpecRequest2{
			Name:                             "Wire Contract Service",
			SpecType:                         ValidationSpecTypeServicePlan,
			FileContent:                      base64.StdEncoding.EncodeToString(spec),
			Environment:                      ptr("Dev"),
			EnvironmentType:                  ptr("DEV"),
			Release:                          ptr(false),
			ReleaseAsPreferred:               ptr(false),
			ForceCreateNewServicePlanVersion: ptr(false),
			Artifacts: []openapiclient.ValidationArtifactInput{{
				LogicalPath:         "terraform/network",
				Encoding:            ValidationArtifactEncoding,
				ArchiveContent:      "H4sIAAAAAAAA",
				Sha256:              "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
				CompressedSizeBytes: 1234,
			}},
		})
	require.NoError(t, err)
	srv.assertNoViolations(t)
	require.Equal(t, "VALID", result.Status)

	requests := srv.snapshot()
	require.Len(t, requests, 1, "retries are disabled, so exactly one request is expected")
	require.Equal(t, http.MethodPost, requests[0].Method)
	require.Equal(t, ValidateServiceSpecPath, requests[0].Path)
	require.Equal(t, "Bearer "+validationTestToken, requests[0].Auth)

	body := requests[0].Body
	require.Equal(t, "service-plan", body["specType"])
	require.Equal(t, base64.StdEncoding.EncodeToString(spec), body["fileContent"])
	require.Equal(t, false, body["release"], "false flags must be sent explicitly")
	require.Equal(t, false, body["releaseAsPreferred"])
	require.Equal(t, false, body["forceCreateNewServicePlanVersion"])

	// The validation envelope has no public dry-run flag.
	for key := range body {
		require.NotContains(t, key, "dryrun")
		require.NotContains(t, key, "dryRun")
	}

	artifacts, ok := body["artifacts"].([]any)
	require.True(t, ok, "artifacts missing from request body: %v", body)
	require.Len(t, artifacts, 1)
	artifact := artifacts[0].(map[string]any)
	require.Equal(t, "terraform/network", artifact["logicalPath"])
	require.Equal(t, "tar+gzip+base64", artifact["encoding"])
	require.Equal(t, "H4sIAAAAAAAA", artifact["archiveContent"])
	require.Len(t, artifact["sha256"], 64)
	require.Equal(t, float64(1234), artifact["compressedSizeBytes"])
}

// TestValidateServiceSpecPreservesTheHTTPStatus proves the wrapper keeps the
// status handleV1Error would otherwise discard, so the CLI can tell "this server
// has no validation endpoint" from a genuine authentication failure without ever
// falling back to a legacy build request.
func TestValidateServiceSpecPreservesTheHTTPStatus(t *testing.T) {
	cases := []struct {
		status      int
		wantUpgrade bool
	}{
		{status: http.StatusNotFound, wantUpgrade: true},
		{status: http.StatusMethodNotAllowed, wantUpgrade: true},
		{status: http.StatusUnauthorized},
		{status: http.StatusForbidden},
		{status: http.StatusBadRequest},
	}

	for _, tc := range cases {
		t.Run(http.StatusText(tc.status), func(t *testing.T) {
			status := tc.status
			srv := startValidationWireServer(t, func(_ int, w http.ResponseWriter) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(status)
				_ = json.NewEncoder(w).Encode(map[string]any{
					"id": "e", "name": "failed", "message": "no",
					"fault": false, "temporary": false, "timeout": false,
				})
			})

			_, err := ValidateServiceSpec(context.Background(), validationTestToken,
				openapiclient.ValidateServiceSpecRequest2{
					Name:        "Wire Contract Service",
					SpecType:    ValidationSpecTypeCompose,
					FileContent: base64.StdEncoding.EncodeToString([]byte("services: {}\n")),
				})
			require.Error(t, err)
			srv.assertNoViolations(t)

			var validationErr *ValidateServiceSpecError
			require.ErrorAs(t, err, &validationErr)
			require.Equal(t, status, validationErr.StatusCode)
			require.Equal(t, tc.wantUpgrade, validationErr.ServerLacksValidationEndpoint())
			require.Len(t, srv.snapshot(), 1, "no retry, no fallback")
		})
	}
}

// TestValidateServiceSpecPostRetryPolicy checks the bounded-retry behaviour of
// the existing retryPolicy against this POST route: only the transient gateway
// and rate-limit statuses are repeated, and the request is immutable, so a
// repeat cannot have a business effect.
func TestValidateServiceSpecPostRetryPolicy(t *testing.T) {
	cases := []struct {
		status       int
		wantRequests int
	}{
		{status: http.StatusServiceUnavailable, wantRequests: 3}, // retried
		{status: http.StatusTooManyRequests, wantRequests: 3},    // retried
		{status: http.StatusInternalServerError, wantRequests: 1},
		{status: http.StatusBadRequest, wantRequests: 1},
	}

	for _, tc := range cases {
		t.Run(http.StatusText(tc.status), func(t *testing.T) {
			status := tc.status
			srv := startValidationWireServer(t, func(_ int, w http.ResponseWriter) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(status)
				_ = json.NewEncoder(w).Encode(map[string]any{
					"id": "e", "name": "failed", "message": "no",
					"fault": false, "temporary": true, "timeout": false,
				})
			})
			t.Setenv("OMNISTRATE_RETRY_MAX", "2")

			_, err := ValidateServiceSpec(context.Background(), validationTestToken,
				openapiclient.ValidateServiceSpecRequest2{
					Name:        "Wire Contract Service",
					SpecType:    ValidationSpecTypeCompose,
					FileContent: base64.StdEncoding.EncodeToString([]byte("services: {}\n")),
				})
			require.Error(t, err)
			srv.assertNoViolations(t)
			require.Len(t, srv.snapshot(), tc.wantRequests)
		})
	}
}

// TestValidateServiceSpecDebugLogsOmitTheBody is the secret-sentinel test from
// 04 §"Output and logging". It is a red/green pair: the same sentinel is sent
// through the legacy build route, where DumpRequestOut still writes the whole
// body, and through the validation route, where the narrow policy must keep it
// out. Without the control the assertion could pass because nothing was logged
// at all.
func TestValidateServiceSpecDebugLogsOmitTheBody(t *testing.T) {
	const sentinel = "SENTINEL-PAYLOAD-9d4e1f7a"
	payload := base64.StdEncoding.EncodeToString([]byte(sentinel))

	t.Run("ControlLegacyBuildRouteStillDumpsTheBody", func(t *testing.T) {
		srv := startValidationWireServer(t, okJSON(map[string]any{
			"serviceID": "s-1", "serviceEnvironmentID": "se-1", "productTierID": "pt-1",
		}))

		logs := captureDataAccessDebugLogs(t, func() {
			_, err := BuildServiceFromServicePlanSpec(context.Background(), validationTestToken,
				openapiclient.BuildServiceFromServicePlanSpecRequest2{
					Name:        "Wire Contract Service",
					FileContent: payload,
				})
			require.NoError(t, err)
		})
		srv.assertNoViolations(t)

		require.Contains(t, logs, payload,
			"the control must show that debug dumps really do include request bodies")
	})

	t.Run("ValidationRouteLogsMetadataOnly", func(t *testing.T) {
		srv := startValidationWireServer(t, okJSON(validationOKResult()))

		logs := captureDataAccessDebugLogs(t, func() {
			_, err := ValidateServiceSpec(context.Background(), validationTestToken,
				openapiclient.ValidateServiceSpecRequest2{
					Name:        "Wire Contract Service",
					SpecType:    ValidationSpecTypeServicePlan,
					FileContent: payload,
					Secrets:     &map[string]string{"app": payload},
					Artifacts: []openapiclient.ValidationArtifactInput{{
						LogicalPath:         "terraform/network",
						Encoding:            ValidationArtifactEncoding,
						ArchiveContent:      payload,
						Sha256:              "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
						CompressedSizeBytes: 12,
					}},
				})
			require.NoError(t, err)
		})
		srv.assertNoViolations(t)

		require.NotEmpty(t, logs)
		require.Contains(t, logs, ValidateServiceSpecPath, "request metadata must still be logged")
		require.Contains(t, logs, "body omitted")
		require.Contains(t, logs, "[REDACTED]")

		require.NotContains(t, logs, sentinel, "the raw sentinel reached the debug log")
		require.NotContains(t, logs, payload, "the encoded payload reached the debug log")
		require.NotContains(t, logs, validationTestToken, "the token reached the debug log")
		require.NotContains(t, logs, "Bearer ", "the Authorization value reached the debug log")
	})
}

// captureDataAccessDebugLogs redirects the global zerolog logger into a buffer
// and turns on the debug level that gates the request/response dumps.
func captureDataAccessDebugLogs(t *testing.T, fn func()) string {
	t.Helper()
	t.Setenv("OMNISTRATE_LOG_LEVEL", "debug")

	var buf bytes.Buffer
	original := log.Logger
	log.Logger = zerolog.New(&buf).Level(zerolog.DebugLevel)
	t.Cleanup(func() { log.Logger = original })

	fn()
	return buf.String()
}
