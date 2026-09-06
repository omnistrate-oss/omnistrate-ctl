package build

// Regression evidence for specification 02 — "Regression tests for the read-only
// contract" (dry-run-implementation-specs/02-regression-tests.md).
//
// HISTORY. This file was originally written against the unfixed tree and every
// test asserted what `build --dry-run` did TODAY, including its defects. Spec 04
// has now landed, so the tests that pinned a defect would otherwise pin a
// behaviour that no longer exists. Each of those has been converted to assert
// the NEW contract; the comment on each one records the defect it used to
// reproduce, so the historical evidence is not lost. The tests that pinned
// behaviour which is still correct (RenderFile's temp-file handling, the
// "exactly one JSON document" property) are unchanged in substance.
//
// The target contract lives in dry_run_target_test.go, which no longer carries a
// build tag: those cases now run by default.
//
// Rules honoured here:
//   - zero real network calls: every request goes to a local httptest.Server;
//   - no require.FailNow inside an HTTP handler goroutine: handlers record a
//     violation string and answer with an error, and the main test goroutine
//     asserts on the recording;
//   - retries are disabled through OMNISTRATE_RETRY_MAX so request counts are
//     deterministic;
//   - HOME is redirected at a temp dir so no test can read or write the
//     developer's real ~/.omnistrate credentials.

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/mitchellh/go-homedir"
	openapiclient "github.com/omnistrate-oss/omnistrate-sdk-go/v1"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/stretchr/testify/require"
)

const (
	// fakeToken is never a real credential; it only has to survive the
	// Authorization header round trip.
	fakeToken = "dry-run-test-token" //nolint:gosec // G101: fixed fake token used only by the local test server

	testServiceID     = "s-dryrun"
	testEnvironmentID = "se-dryrun"
	testProductTierID = "pt-dryrun"
	testServiceName   = "Dry Run Regression Service"
)

// Wire paths exercised by the build route. Kept as constants so the target
// contract tests can forbid them by name.
const (
	pathPrepareServicePlanSpec = "/2022-09-01-00/service/serviceplanspec/prepare"
	pathBuildServicePlanSpec   = "/2022-09-01-00/service/serviceplanspec"
	pathBuildComposeSpec       = "/2022-09-01-00/service/composespec"
	pathDeploymentArtifact     = "/2022-09-01-00/deployment-artifact"
	pathComposeGenImage        = "/2022-09-01-00/compose-gen/image"

	// pathValidateSpec is the read-only validation endpoint specified in
	// 03-backend-validation.md. It does not exist yet.
	pathValidateSpec = "/2022-09-01-00/service/spec/validate"
)

// recordedRequest is one HTTP request observed by the test server.
type recordedRequest struct {
	Method   string
	Path     string
	RawQuery string
	Auth     string
	Body     []byte
}

func (r recordedRequest) String() string {
	if r.RawQuery == "" {
		return r.Method + " " + r.Path
	}
	return r.Method + " " + r.Path + "?" + r.RawQuery
}

// decodeBody decodes a recorded JSON request body. It returns nil for requests
// without a body.
func (r recordedRequest) decodeBody(t *testing.T) map[string]any {
	t.Helper()
	if len(bytes.TrimSpace(r.Body)) == 0 {
		return nil
	}
	var out map[string]any
	require.NoError(t, json.Unmarshal(r.Body, &out), "request %s body is not a JSON object: %s", r, string(r.Body))
	return out
}

// routeFunc answers a recorded request. It returns false when it does not know
// the route, in which case the recorder reports an unrouted request.
type routeFunc func(rec recordedRequest, w http.ResponseWriter) bool

// requestRecorder is the local test server. It records every request and never
// calls t.Fatal/require.FailNow from the handler goroutine; contract breaches
// are appended to violations and asserted by the main goroutine instead.
type requestRecorder struct {
	server *httptest.Server

	mu         sync.Mutex
	requests   []recordedRequest
	violations []string

	// route answers known endpoints.
	route routeFunc
	// forbid classifies a request as a contract violation. An empty string
	// means the request is allowed.
	forbid func(rec recordedRequest) string
}

func (rr *requestRecorder) handler(w http.ResponseWriter, req *http.Request) {
	body, readErr := io.ReadAll(req.Body)
	rec := recordedRequest{
		Method:   req.Method,
		Path:     req.URL.Path,
		RawQuery: req.URL.RawQuery,
		Auth:     req.Header.Get("Authorization"),
		Body:     body,
	}

	rr.mu.Lock()
	rr.requests = append(rr.requests, rec)
	if readErr != nil {
		rr.violations = append(rr.violations, fmt.Sprintf("failed to read body of %s: %v", rec, readErr))
	}
	forbid := rr.forbid
	route := rr.route
	rr.mu.Unlock()

	if forbid != nil {
		if reason := forbid(rec); reason != "" {
			rr.recordViolation(reason)
			writeAPIError(w, http.StatusInternalServerError, "forbidden_by_test", reason)
			return
		}
	}

	if route != nil && route(rec, w) {
		return
	}

	rr.recordViolation(fmt.Sprintf("unrouted request %s", rec))
	writeAPIError(w, http.StatusNotFound, "not_routed_by_test", "no test route for "+rec.String())
}

func (rr *requestRecorder) recordViolation(reason string) {
	rr.mu.Lock()
	defer rr.mu.Unlock()
	rr.violations = append(rr.violations, reason)
}

// recorded returns the observed requests in order.
func (rr *requestRecorder) recorded() []recordedRequest {
	rr.mu.Lock()
	defer rr.mu.Unlock()
	out := make([]recordedRequest, len(rr.requests))
	copy(out, rr.requests)
	return out
}

// sequence renders the recorded requests as "METHOD /path" strings. This is the
// primary evidence produced by these tests.
func (rr *requestRecorder) sequence() []string {
	recs := rr.recorded()
	out := make([]string, 0, len(recs))
	for _, rec := range recs {
		out = append(out, rec.Method+" "+rec.Path)
	}
	return out
}

// find returns the recorded requests for a path, in order.
func (rr *requestRecorder) find(path string) []recordedRequest {
	var out []recordedRequest
	for _, rec := range rr.recorded() {
		if rec.Path == path {
			out = append(out, rec)
		}
	}
	return out
}

func (rr *requestRecorder) countPath(path string) int {
	n := 0
	for _, rec := range rr.recorded() {
		if rec.Path == path {
			n++
		}
	}
	return n
}

// assertNoViolations runs on the main test goroutine.
func (rr *requestRecorder) assertNoViolations(t *testing.T) {
	t.Helper()
	rr.mu.Lock()
	violations := append([]string(nil), rr.violations...)
	rr.mu.Unlock()
	require.Emptyf(t, violations, "test server recorded contract violations: %s\nrequest sequence: %s",
		strings.Join(violations, "; "), strings.Join(rr.sequence(), " | "))
}

func (rr *requestRecorder) setRoute(route routeFunc) {
	rr.mu.Lock()
	defer rr.mu.Unlock()
	rr.route = route
}

func (rr *requestRecorder) setForbid(forbid func(rec recordedRequest) string) {
	rr.mu.Lock()
	defer rr.mu.Unlock()
	rr.forbid = forbid
}

// startRecordingServer starts the local server and points the CTL SDK clients at
// it. Retries are disabled so that request counts are exact, and HOME is moved
// to a temp directory so that config.GetToken() cannot pick up (or disturb) the
// developer's real credentials.
func startRecordingServer(t *testing.T) *requestRecorder {
	t.Helper()

	rr := &requestRecorder{}
	rr.server = httptest.NewServer(http.HandlerFunc(rr.handler))
	t.Cleanup(rr.server.Close)

	serverURL, err := url.Parse(rr.server.URL)
	require.NoError(t, err)

	t.Setenv("OMNISTRATE_HOST", serverURL.Host)
	t.Setenv("OMNISTRATE_HOST_SCHEME", serverURL.Scheme)
	// Deterministic request counts: no retries, no backoff sleeps.
	t.Setenv("OMNISTRATE_RETRY_MAX", "0")
	t.Setenv("OMNISTRATE_RETRY_WAIT_MIN_IN_SECONDS", "0")
	t.Setenv("OMNISTRATE_RETRY_WAIT_MAX_IN_SECONDS", "0")
	t.Setenv("OMNISTRATE_CLIENT_TIMEOUT_IN_SECONDS", "30")

	// utils.PrintError calls os.Exit(1) unless config.IsDryRun() is true. That
	// env flag is the process-global OMNISTRATE_DRY_RUN setting, which is
	// distinct from the customer's --dry-run flag and, in this codebase, is read
	// by nothing except PrintError. Setting it here only stops the test binary
	// from being killed mid-assertion; it does not change the build route.
	t.Setenv("OMNISTRATE_DRY_RUN", "true")

	// go-homedir caches the resolved home directory; disable the cache so the
	// redirected HOME below is honoured for every lookup in the process.
	homedir.DisableCache = true
	fakeHome := t.TempDir()
	t.Setenv("HOME", fakeHome)
	t.Setenv("USERPROFILE", fakeHome)

	return rr
}

// disablePrintErrorExit stops utils.PrintError from killing the test binary.
// See the note in startRecordingServer.
func disablePrintErrorExit(t *testing.T) {
	t.Helper()
	t.Setenv("OMNISTRATE_DRY_RUN", "true")
}

// writeAPIError emits the SDK's Error shape so handleV1Error surfaces the
// message rather than a decoding failure.
func writeAPIError(w http.ResponseWriter, status int, name, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"id":        "test-error",
		"name":      name,
		"message":   message,
		"fault":     false,
		"temporary": false,
		"timeout":   false,
	})
}

// writeJSON answers 200 with a JSON body.
func writeJSON(w http.ResponseWriter, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(payload)
}

// ---------------------------------------------------------------------------
// Canned validation responses
// ---------------------------------------------------------------------------

// validationStatus values from 03-backend-validation.md.
const (
	statusValid      = "VALID"
	statusInvalid    = "INVALID"
	statusIncomplete = "INCOMPLETE"
)

// validationResult builds a 03-shaped result payload.
func validationResult(status string, requiredArtifacts []map[string]any) map[string]any {
	if requiredArtifacts == nil {
		requiredArtifacts = []map[string]any{}
	}
	diagnostics := []map[string]any{}
	if status == statusInvalid {
		diagnostics = append(diagnostics, map[string]any{
			"code":     "SPEC_SEMANTICS_INVALID",
			"severity": "error",
			"path":     "/services/0",
			"message":  "resource has no deployment configuration",
		})
	}
	if status == statusIncomplete && len(requiredArtifacts) == 0 {
		diagnostics = append(diagnostics, map[string]any{
			"code":     "DEPENDENCY_UNAVAILABLE",
			"severity": "incomplete",
			"path":     "/services/0",
			"message":  "a required dependency could not be read",
		})
	}
	if status == statusIncomplete && len(requiredArtifacts) > 0 {
		diagnostics = append(diagnostics, map[string]any{
			"code":     "ARTIFACT_CONTENT_REQUIRED",
			"severity": "incomplete",
			"path":     "/services/0",
			"message":  "Local artifact content is required for validation.",
		})
	}
	return map[string]any{
		"status":            status,
		"validationVersion": "1",
		"inputDigest":       "0000000000000000000000000000000000000000000000000000000000000000",
		"checks": []map[string]any{
			{"name": "syntax", "status": "PASSED"},
			{"name": "artifact-content", "status": "PASSED"},
		},
		"diagnostics":        diagnostics,
		"requiredArtifacts":  requiredArtifacts,
		"validatedArtifacts": []map[string]any{},
	}
}

// validationResultAcknowledging answers the second request with a result that
// acknowledges every supplied artifact, echoing its digest and size and listing
// the uses that made it required — the shape 04 step 8 tells the client to
// check.
//
// It runs on the HTTP handler goroutine, so it never touches *testing.T: a
// malformed request simply produces an unacknowledged result, which the client
// then rejects on the main goroutine.
func validationResultAcknowledging(status string, rec recordedRequest, requirements []map[string]any) map[string]any {
	result := validationResult(status, nil)

	usesByPath := map[string]any{}
	for _, requirement := range requirements {
		if logicalPath, ok := requirement["logicalPath"].(string); ok {
			usesByPath[logicalPath] = requirement["uses"]
		}
	}

	var request struct {
		Artifacts []struct {
			LogicalPath         string `json:"logicalPath"`
			Sha256              string `json:"sha256"`
			CompressedSizeBytes int64  `json:"compressedSizeBytes"`
		} `json:"artifacts"`
	}
	if err := json.Unmarshal(rec.Body, &request); err != nil {
		return result
	}

	validated := make([]map[string]any, 0, len(request.Artifacts))
	for _, artifact := range request.Artifacts {
		uses := usesByPath[artifact.LogicalPath]
		if uses == nil {
			uses = []map[string]any{}
		}
		validated = append(validated, map[string]any{
			"logicalPath":         artifact.LogicalPath,
			"sha256":              artifact.Sha256,
			"compressedSizeBytes": artifact.CompressedSizeBytes,
			"uses":                uses,
		})
	}
	result["validatedArtifacts"] = validated
	return result
}

// firstArtifact returns the single artifact object from a validation request
// body.
func firstArtifact(t *testing.T, body map[string]any) map[string]any {
	t.Helper()
	artifacts, ok := body["artifacts"].([]any)
	require.True(t, ok, "request body carries no artifacts: %v", body)
	require.Len(t, artifacts, 1)
	artifact, ok := artifacts[0].(map[string]any)
	require.True(t, ok)
	return artifact
}

// decodeArchive base64-decodes an artifact's archiveContent and checks the
// declared digest and size against the decoded bytes, exactly as the server
// would.
func decodeArchive(t *testing.T, artifact map[string]any) []byte {
	t.Helper()
	encoded, ok := artifact["archiveContent"].(string)
	require.True(t, ok, "artifact has no archiveContent: %v", artifact)
	raw, err := base64.StdEncoding.DecodeString(encoded)
	require.NoError(t, err)

	sum := sha256.Sum256(raw)
	require.Equal(t, hex.EncodeToString(sum[:]), artifact["sha256"],
		"sha256 must hash the decoded compressed bytes")
	require.Equal(t, float64(len(raw)), artifact["compressedSizeBytes"],
		"compressedSizeBytes must count the decoded compressed bytes")
	require.Equal(t, "tar+gzip+base64", artifact["encoding"])
	return raw
}

// tarEntryNames lists the member names of a gzip-compressed tar archive.
func tarEntryNames(t *testing.T, archive []byte) []string {
	t.Helper()
	gzReader, err := gzip.NewReader(bytes.NewReader(archive))
	require.NoError(t, err)
	defer gzReader.Close()

	var names []string
	tarReader := tar.NewReader(gzReader)
	for {
		header, readErr := tarReader.Next()
		if readErr == io.EOF {
			break
		}
		require.NoError(t, readErr)
		names = append(names, header.Name)
	}
	return names
}

// ---------------------------------------------------------------------------
// Canned legacy responses
// ---------------------------------------------------------------------------

func prepareResult(tasks []map[string]string) map[string]any {
	if tasks == nil {
		tasks = []map[string]string{}
	}
	return map[string]any{
		"serviceID":               testServiceID,
		"serviceEnvironmentID":    testEnvironmentID,
		"productTierID":           testProductTierID,
		"isNewProductTierCreated": false,
		"artifactUploadingTasks":  tasks,
	}
}

func buildResult() map[string]any {
	return map[string]any{
		"serviceID":                      testServiceID,
		"serviceEnvironmentID":           testEnvironmentID,
		"productTierID":                  testProductTierID,
		"isNewServicePlanVersionCreated": false,
		"undefinedResources":             map[string]string{},
	}
}

func productTierResult() map[string]any {
	return map[string]any{
		"id":                       testProductTierID,
		"key":                      "dry-run-plan",
		"name":                     "Dry Run Plan",
		"description":              "plan description",
		"planDescription":          "plan description",
		"documentation":            "docs",
		"enableDeletionProtection": false,
		"isDisabled":               false,
		"pricing":                  map[string]any{},
		"serviceId":                testServiceID,
		"serviceModelId":           "sm-dryrun",
		"support":                  "support",
		"tierType":                 "OMNISTRATE_HOSTED",
	}
}

func environmentResult(id, envType string, saasPortalReady bool) map[string]any {
	out := map[string]any{
		"id":                 id,
		"key":                strings.ToLower(envType),
		"name":               envType,
		"description":        envType + " environment",
		"deploymentConfigId": "dc-default",
		"serviceId":          testServiceID,
		"type":               envType,
		"visibility":         "PUBLIC",
	}
	if saasPortalReady {
		out["saasPortalUrl"] = "saas.example.invalid"
		out["saasPortalStatus"] = "RUNNING"
	}
	return out
}

// legacyRoutes answers every endpoint the current build route can reach.
// Handlers must not fail the test directly; they only serve canned data.
func legacyRoutes(prepareTasks []map[string]string) routeFunc {
	return func(rec recordedRequest, w http.ResponseWriter) bool {
		switch {
		case rec.Method == http.MethodPut && rec.Path == pathPrepareServicePlanSpec:
			writeJSON(w, prepareResult(prepareTasks))
		case rec.Method == http.MethodPut && rec.Path == pathBuildServicePlanSpec:
			writeJSON(w, buildResult())
		case rec.Method == http.MethodPut && rec.Path == pathBuildComposeSpec:
			writeJSON(w, buildResult())
		case rec.Method == http.MethodPost && rec.Path == pathDeploymentArtifact:
			writeJSON(w, "artifact-1")
		case rec.Method == http.MethodGet && strings.HasPrefix(rec.Path, pathDeploymentArtifact+"/"):
			writeJSON(w, map[string]any{"id": "artifact-1", "status": "READY"})
		case rec.Method == http.MethodGet && rec.Path == "/2022-09-01-00/service/"+testServiceID+"/product-tier/"+testProductTierID:
			writeJSON(w, productTierResult())
		case rec.Method == http.MethodGet && rec.Path == "/2022-09-01-00/accountconfig":
			writeJSON(w, map[string]any{"accountConfigs": []any{}})
		case rec.Method == http.MethodGet && rec.Path == "/2022-09-01-00/service/"+testServiceID+"/environment":
			writeJSON(w, map[string]any{"ids": []string{testEnvironmentID}})
		case rec.Method == http.MethodGet && rec.Path == "/2022-09-01-00/service/"+testServiceID+"/environment/"+testEnvironmentID:
			writeJSON(w, environmentResult(testEnvironmentID, "DEV", true))
		case rec.Method == http.MethodGet && strings.HasPrefix(rec.Path, "/2022-09-01-00/service/"+testServiceID+"/environment/"):
			id := strings.TrimPrefix(rec.Path, "/2022-09-01-00/service/"+testServiceID+"/environment/")
			writeJSON(w, environmentResult(id, "PROD", true))
		case rec.Method == http.MethodPost && rec.Path == "/2022-09-01-00/service/"+testServiceID+"/environment":
			writeJSON(w, "se-prod-created")
		case rec.Method == http.MethodPost && strings.HasSuffix(rec.Path, "/promote"):
			writeJSON(w, map[string]any{})
		case rec.Method == http.MethodGet && rec.Path == "/2022-09-01-00/deployment-config/default":
			writeJSON(w, map[string]any{
				"id":                     "dc-default",
				"name":                   "default",
				"description":            "default",
				"infraRollConfiguration": map[string]any{},
				"rolloutPriorityList":    []any{},
			})
		case rec.Method == http.MethodGet && rec.Path == pathComposeGenImage:
			writeJSON(w, map[string]any{"imageAccessible": true})
		case rec.Method == http.MethodPost && rec.Path == pathComposeGenImage:
			writeJSON(w, map[string]any{
				"fileContent": base64.StdEncoding.EncodeToString([]byte(minimalComposeSpec)),
			})
		default:
			return false
		}
		return true
	}
}

// ---------------------------------------------------------------------------
// Fixtures
// ---------------------------------------------------------------------------

const minimalComposeSpec = `services:
  api:
    image: docker.io/library/nginx:1.27
`

const minimalServicePlanSpec = `version: "1.0"
name: Dry Run Plan
deployment:
  hostedDeployment:
    awsAccountId: "000000000000"
services:
  - name: network
    terraformConfigurations:
      configurationPerCloudProvider:
        aws:
          terraformPath: /terraform/network
`

// servicePlanSpecWithLocalTerraform declares an explicit local Terraform source.
// The path it names is the only path a validation response is allowed to ask
// for; see collectDeclaredArtifactPaths.
const servicePlanSpecWithLocalTerraform = `version: "1.0"
name: Dry Run Plan
deployment:
  hostedDeployment:
    awsAccountId: "000000000000"
services:
  - name: network
    terraformConfigurations:
      configurationPerCloudProvider:
        aws:
          terraformPath: /terraform/network
          artifactsLocalPath: ./terraform/network
`

// servicePlanSpecWithLocalArchive declares a pre-built archive as its local
// Helm source, exercising the "existing .tgz is not rewrapped" branch.
const servicePlanSpecWithLocalArchive = `version: "1.0"
name: Dry Run Plan
deployment:
  hostedDeployment:
    awsAccountId: "000000000000"
services:
  - name: chart
    helmChartConfiguration:
      chartName: demo
      chartVersion: "1.0.0"
      artifactsLocalPath: ./artifacts/bundle.tar.gz
`

// writeFixture writes content into dir/name, creating parent directories.
func writeFixture(t *testing.T, dir, name, content string) string {
	t.Helper()
	full := filepath.Join(dir, name)
	require.NoError(t, os.MkdirAll(filepath.Dir(full), 0o755))
	require.NoError(t, os.WriteFile(full, []byte(content), 0o600))
	return full
}

// treeSnapshot maps every regular file under root to its sha256. It is used to
// prove that preprocessing does not modify or add source files.
func treeSnapshot(t *testing.T, root string) map[string]string {
	t.Helper()
	out := map[string]string{}
	require.NoError(t, filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() || !info.Mode().IsRegular() {
			return nil
		}
		data, readErr := os.ReadFile(path) //nolint:gosec // test-owned temp tree
		if readErr != nil {
			return readErr
		}
		rel, relErr := filepath.Rel(root, path)
		if relErr != nil {
			return relErr
		}
		sum := sha256.Sum256(data)
		out[rel] = hex.EncodeToString(sum[:])
		return nil
	}))
	return out
}

// execShim installs stub executables on PATH so the test can prove that the
// dry-run route runs no Docker build/push and no Git command. Each invocation is
// appended to a log file. names maps a binary name to the shell body it runs
// after logging; an empty body just exits 0.
func execShim(t *testing.T, names map[string]string) (logPath string) {
	t.Helper()
	shimDir := t.TempDir()
	logPath = filepath.Join(shimDir, "invocations.log")
	require.NoError(t, os.WriteFile(logPath, nil, 0o600))

	for name, body := range names {
		// The shim runs with PATH restricted to shimDir, so give the script
		// itself a usable PATH for the helpers it needs.
		script := "#!/bin/sh\n" +
			"PATH=/usr/bin:/bin:/usr/sbin:/sbin\n" +
			"export PATH\n" +
			"printf '%s %s\\n' " + name + " \"$*\" >> " + logPath + "\n" +
			body + "\n"
		require.NoError(t, os.WriteFile(filepath.Join(shimDir, name), []byte(script), 0o700)) //nolint:gosec // test shim must be executable
	}

	// PATH contains only the shim directory: any unexpected subprocess fails
	// loudly instead of silently reaching a real binary.
	t.Setenv("PATH", shimDir)
	return logPath
}

func shimInvocations(t *testing.T, logPath string) []string {
	t.Helper()
	data, err := os.ReadFile(logPath) //nolint:gosec // test-owned temp file
	require.NoError(t, err)
	var out []string
	for _, line := range strings.Split(string(data), "\n") {
		if strings.TrimSpace(line) != "" {
			out = append(out, line)
		}
	}
	return out
}

// captureStdout redirects os.Stdout for the duration of fn.
func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	r, w, err := os.Pipe()
	require.NoError(t, err)

	orig := os.Stdout
	os.Stdout = w

	done := make(chan string, 1)
	go func() {
		var buf bytes.Buffer
		_, _ = io.Copy(&buf, r)
		done <- buf.String()
	}()

	fn()

	os.Stdout = orig
	require.NoError(t, w.Close())
	out := <-done
	require.NoError(t, r.Close())
	return out
}

// jsonDryRunOptions returns the options for `build --dry-run -o json`.
func jsonDryRunOptions(file, specType string) buildOptions {
	return buildOptions{
		file:            file,
		specType:        specType,
		name:            testServiceName,
		environment:     "Dev",
		environmentType: "dev",
		output:          "json",
		dryRun:          true,
	}
}

func staticToken() tokenProvider {
	return func() (string, error) { return fakeToken, nil }
}

// ---------------------------------------------------------------------------
// Shared validation-endpoint server
// ---------------------------------------------------------------------------

// startValidationServerAllowing starts a recorder that treats every request as a
// contract violation except POST /service/spec/validate and whatever extraAllow
// permits. It is the single implementation behind both this file's converted
// reproductions and dry_run_target_test.go's startValidationOnlyServer.
func startValidationServerAllowing(
	t *testing.T,
	extraAllow func(rec recordedRequest) bool,
	respond func(call int, rec recordedRequest, w http.ResponseWriter),
) *requestRecorder {
	t.Helper()

	rr := startRecordingServer(t)
	rr.setForbid(func(rec recordedRequest) string {
		if rec.Method == http.MethodPost && rec.Path == pathValidateSpec {
			return ""
		}
		if extraAllow != nil && extraAllow(rec) {
			return ""
		}
		return "read-only validation must not issue " + rec.String()
	})

	base := legacyRoutes(nil)
	calls := 0
	rr.setRoute(func(rec recordedRequest, w http.ResponseWriter) bool {
		if rec.Method == http.MethodPost && rec.Path == pathValidateSpec {
			n := calls
			calls++
			respond(n, rec, w)
			return true
		}
		if extraAllow != nil && extraAllow(rec) {
			return base(rec, w)
		}
		return false
	})
	return rr
}

// allowComposeGenReads permits the two read-only compose-generation calls the
// --image branch makes before there is any specification to validate.
//
// Both are reads: ows-orchestration's ComposeGen.GenerateComposeSpecFromContainerImage
// checks access, reads container image metadata and marshals an in-memory
// compose project. It performs no database write, starts no workflow and pushes
// no image.
func allowComposeGenReads(rec recordedRequest) bool {
	return rec.Path == pathComposeGenImage
}

// ---------------------------------------------------------------------------
// ServicePlanSpec dry run
// ---------------------------------------------------------------------------

// TestBuildDryRunServicePlanSpecNeverIssuesPrepare records the request sequence
// for `build --spec-type ServicePlanSpec --dry-run`.
//
// CONVERTED. This test used to be named
// TestBuildDryRunServicePlanSpecIssuesPrepareBeforeValidation and asserted the
// defect from the investigation ("Preparation can create real objects before
// dry-run validation"): FindOrCreateServiceHierarchy issued the mutating
// PUT .../serviceplanspec/prepare before any validation happened, and that
// prepare request carried no dry-run field, so the backend could not know the
// caller only wanted validation. The dry-run branch now returns before
// FindOrCreateServiceHierarchy, so the test asserts the absence of that request
// instead of its presence.
func TestBuildDryRunServicePlanSpecNeverIssuesPrepare(t *testing.T) {
	rr := startValidationServerAllowing(t, nil, func(_ int, _ recordedRequest, w http.ResponseWriter) {
		writeJSON(w, validationResult(statusValid, nil))
	})

	dir := t.TempDir()
	specPath := writeFixture(t, dir, "spec.yaml", minimalServicePlanSpec)
	t.Chdir(dir)

	err := runBuildWithOptions(context.Background(),
		jsonDryRunOptions(specPath, ServicePlanSpecType), staticToken())
	require.NoError(t, err)
	rr.assertNoViolations(t)

	require.Equal(t, []string{
		"POST " + pathValidateSpec,
	}, rr.sequence(), "recorded request sequence for build --dry-run (ServicePlanSpec)")
	t.Logf("ServicePlanSpec dry-run request sequence: %s", strings.Join(rr.sequence(), " | "))

	require.Zero(t, rr.countPath(pathPrepareServicePlanSpec), "no prepare request")
	require.Zero(t, rr.countPath(pathBuildServicePlanSpec), "no build request")
	require.Zero(t, rr.countPath(pathDeploymentArtifact), "no artifact publication")

	validations := rr.find(pathValidateSpec)
	require.Len(t, validations, 1)
	require.Equal(t, "Bearer "+fakeToken, validations[0].Auth)

	body := validations[0].decodeBody(t)
	require.Equal(t, base64.StdEncoding.EncodeToString([]byte(minimalServicePlanSpec)), body["fileContent"])
	for key := range body {
		require.NotContains(t, strings.ToLower(key), "dry",
			"the validation envelope has no dry-run flag: %v", body)
	}
}

// TestBuildDryRunServicePlanSpecValidatesLocalArtifactsWithoutUpload is the
// converted form of TestBuildDryRunServicePlanSpecSkipsUploadButStillPrepares,
// which recorded the asymmetry the investigation found: artifact upload was
// guarded by dryRun, but the prepare call that produced the upload tasks was
// not, so local artifact content was never validated at all. Local content is
// now validated through the request-scoped transport, with no upload.
func TestBuildDryRunServicePlanSpecValidatesLocalArtifactsWithoutUpload(t *testing.T) {
	requirement := []map[string]any{{
		"logicalPath": "terraform/network",
		"uses": []map[string]any{{
			"resourceKey": "network",
			"kind":        "terraform",
			"provider":    "aws",
			"path":        "/services/0/terraformConfigurations/configurationPerCloudProvider/aws",
		}},
	}}

	rr := startValidationServerAllowing(t, nil, func(call int, rec recordedRequest, w http.ResponseWriter) {
		if call == 0 {
			writeJSON(w, validationResult(statusIncomplete, requirement))
			return
		}
		writeJSON(w, validationResultAcknowledging(statusValid, rec, requirement))
	})

	dir := t.TempDir()
	specPath := writeFixture(t, dir, "spec.yaml", servicePlanSpecWithLocalTerraform)
	writeFixture(t, dir, "terraform/network/main.tf", "resource \"null_resource\" \"n\" {}\n")
	t.Chdir(dir)

	err := runBuildWithOptions(context.Background(),
		jsonDryRunOptions(specPath, ServicePlanSpecType), staticToken())
	require.NoError(t, err)
	rr.assertNoViolations(t)

	require.Equal(t, []string{
		"POST " + pathValidateSpec,
		"POST " + pathValidateSpec,
	}, rr.sequence(), "recorded request sequence for build --dry-run with local artifacts")
	t.Logf("ServicePlanSpec dry-run with artifacts request sequence: %s", strings.Join(rr.sequence(), " | "))

	require.Zero(t, rr.countPath(pathPrepareServicePlanSpec), "no prepare request")
	require.Zero(t, rr.countPath(pathDeploymentArtifact), "no artifact upload")

	// The archive really carries the local file.
	artifact := firstArtifact(t, rr.find(pathValidateSpec)[1].decodeBody(t))
	names := tarEntryNames(t, decodeArchive(t, artifact))
	require.Contains(t, names, "main.tf")
}

// ---------------------------------------------------------------------------
// Compose dry run
// ---------------------------------------------------------------------------

// TestBuildDryRunComposeSpecUsesValidationEndpoint is the converted form of
// TestBuildDryRunComposeSpecUsesNormalBuildEndpoint, which recorded that the
// Compose dry run went to the normal build endpoint with dryrun=true, i.e. into
// the mutating import workflow.
func TestBuildDryRunComposeSpecUsesValidationEndpoint(t *testing.T) {
	rr := startValidationServerAllowing(t, nil, func(_ int, _ recordedRequest, w http.ResponseWriter) {
		writeJSON(w, validationResult(statusValid, nil))
	})

	dir := t.TempDir()
	specPath := writeFixture(t, dir, "omnistrate-compose.yaml", minimalComposeSpec)
	t.Chdir(dir)

	err := runBuildWithOptions(context.Background(),
		jsonDryRunOptions(specPath, DockerComposeSpecType), staticToken())
	require.NoError(t, err)
	rr.assertNoViolations(t)

	require.Equal(t, []string{
		"POST " + pathValidateSpec,
	}, rr.sequence(), "recorded request sequence for build --dry-run (Compose)")
	t.Logf("Compose dry-run request sequence: %s", strings.Join(rr.sequence(), " | "))

	require.Zero(t, rr.countPath(pathBuildComposeSpec), "no build request")
	require.Zero(t, rr.countPath(pathPrepareServicePlanSpec))

	body := rr.find(pathValidateSpec)[0].decodeBody(t)
	require.Equal(t, "compose", body["specType"])
}

// ---------------------------------------------------------------------------
// Interactive mode
// ---------------------------------------------------------------------------

// TestBuildDryRunInteractiveNeverPromotes drives the full route with
// --dry-run --interactive and makes "y" available on stdin.
//
// CONVERTED. This test used to be named TestBuildDryRunInteractivePromotionHasNoGuard
// and asserted the defect from the investigation ("Interactive dry-run can
// proceed to production actions"): the post-build prompts were not guarded by
// dryRun, so answering "y" created a real production environment and promoted
// into it. It now asserts that no prompt is reached and no environment or
// promotion request is issued, per 04 §"CTL execution algorithm" step 9.
func TestBuildDryRunInteractiveNeverPromotes(t *testing.T) {
	rr := startValidationServerAllowing(t, nil, func(_ int, _ recordedRequest, w http.ResponseWriter) {
		writeJSON(w, validationResult(statusValid, nil))
	})

	stdinR, stdinW, err := os.Pipe()
	require.NoError(t, err)
	origStdin := os.Stdin
	os.Stdin = stdinR
	t.Cleanup(func() {
		os.Stdin = origStdin
		_ = stdinW.Close()
		_ = stdinR.Close()
	})
	// Make "y" available up front: if any prompt were still reached it would
	// consume this and proceed, so the assertions below would fail loudly rather
	// than deadlock.
	_, err = stdinW.WriteString("y\ny\ny\n")
	require.NoError(t, err)

	dir := t.TempDir()
	specPath := writeFixture(t, dir, "omnistrate-compose.yaml", minimalComposeSpec)
	t.Chdir(dir)

	opts := jsonDryRunOptions(specPath, DockerComposeSpecType)
	opts.output = "table" // --interactive is rejected with json output
	opts.interactive = true

	runErr := runBuildWithOptions(context.Background(), opts, staticToken())
	require.NoError(t, runErr)
	rr.assertNoViolations(t)

	sequence := strings.Join(rr.sequence(), " | ")
	require.Equal(t, []string{"POST " + pathValidateSpec}, rr.sequence(),
		"interactive dry run must issue only the validation request; sequence: %s", sequence)
	require.NotContains(t, sequence, "/environment")
	require.NotContains(t, sequence, "/promote")
	t.Logf("interactive dry-run request sequence: %s", sequence)
}

// ---------------------------------------------------------------------------
// RenderFile writes <basename>.tmp into the input root; the dry run does not
// ---------------------------------------------------------------------------

// TestBuildDryRunRenderFileOverwritesSpecTmpSentinel proves the preprocessing
// defect described in 04 §"CTL execution algorithm" step 2: RenderFile writes
// "<basename>.tmp" next to the user's spec for `docker compose config`.
//
// A sentinel file is placed at that path first. Both sub-cases show it being
// destroyed, and the failure sub-case additionally shows the temporary file
// being left behind in the user's source tree.
//
// UNCHANGED. RenderFile still behaves this way and is still what the real build
// uses; the dry-run route no longer calls it. See
// TestDryRunValidationLeavesTheSpecTmpSentinelAlone for the replacement path.
func TestBuildDryRunRenderFileOverwritesSpecTmpSentinel(t *testing.T) {
	const sentinel = "SENTINEL-DO-NOT-TOUCH\n"
	const envFileSpec = `services:
  api:
    image: docker.io/library/nginx:1.27
    env_file:
      - ./api.env
`

	t.Run("DockerConfigSucceedsAndDeletesTheSentinelPath", func(t *testing.T) {
		disablePrintErrorExit(t)
		dir := t.TempDir()
		specPath := writeFixture(t, dir, "omnistrate-compose.yaml", envFileSpec)
		writeFixture(t, dir, "api.env", "KEY=value\n")
		tmpPath := filepath.Join(dir, "omnistrate-compose.yaml.tmp")
		require.NoError(t, os.WriteFile(tmpPath, []byte(sentinel), 0o600))

		// Controlled `docker` that echoes the file it was handed; no real Docker.
		execShim(t, map[string]string{"docker": `cat "$3"`})

		rendered, err := RenderFile([]byte(envFileSpec), dir, specPath, nil, nil)
		require.NoError(t, err)
		require.NotEmpty(t, rendered)

		_, statErr := os.Stat(tmpPath)
		require.True(t, os.IsNotExist(statErr),
			"RenderFile deleted the pre-existing %s written by the user", tmpPath)
	})

	t.Run("DockerConfigFailsAndLeavesTheRenderedTempFileBehind", func(t *testing.T) {
		disablePrintErrorExit(t)
		dir := t.TempDir()
		specPath := writeFixture(t, dir, "omnistrate-compose.yaml", envFileSpec)
		writeFixture(t, dir, "api.env", "KEY=value\n")
		tmpPath := filepath.Join(dir, "omnistrate-compose.yaml.tmp")
		require.NoError(t, os.WriteFile(tmpPath, []byte(sentinel), 0o600))

		execShim(t, map[string]string{"docker": "exit 1"})

		_, err := RenderFile([]byte(envFileSpec), dir, specPath, nil, nil)
		require.Error(t, err, "docker compose config failure must surface")

		leftover, readErr := os.ReadFile(tmpPath) //nolint:gosec // test-owned temp tree
		require.NoError(t, readErr, "the temporary file is left in the user's input root on error")
		require.NotEqual(t, sentinel, string(leftover),
			"the user's file at %s was overwritten by RenderFile", tmpPath)
		require.Contains(t, string(leftover), "nginx")
	})
}

// ---------------------------------------------------------------------------
// Preprocessing safety checks
// ---------------------------------------------------------------------------

// TestBuildDryRunPreprocessingPreservesSourceTreeAndPathSemantics asserts the
// safety properties 02 requires from preprocessing: `$file` references and
// Compose config/secret files keep their path semantics, no source file is
// modified, no file is added to the input tree, and no Docker or Git subprocess
// runs.
//
// CONVERTED. The assertions on the envelope moved from the legacy build endpoint
// to the validation endpoint; the preprocessing properties themselves are
// unchanged, which is the point — the same bytes a real build would send are
// what gets validated.
func TestBuildDryRunPreprocessingPreservesSourceTreeAndPathSemantics(t *testing.T) {
	rr := startValidationServerAllowing(t, nil, func(_ int, _ recordedRequest, w http.ResponseWriter) {
		writeJSON(w, validationResult(statusValid, nil))
	})

	const composeWithFileRefs = `services:
  x-embed-$: {{ $file:./fragments/api.yaml }}
configs:
  app_config:
    file: ./conf/app.conf
secrets:
  app_secret:
    file: ./conf/app.secret
`
	const apiFragment = `api:
  image: docker.io/library/nginx:1.27
  labels:
    embedded: from-file-reference
`
	dir := t.TempDir()
	specPath := writeFixture(t, dir, "omnistrate-compose.yaml", composeWithFileRefs)
	writeFixture(t, dir, "fragments/api.yaml", apiFragment)
	writeFixture(t, dir, "conf/app.conf", "config-file-content\n")
	writeFixture(t, dir, "conf/app.secret", "secret-file-content\n")

	logPath := execShim(t, map[string]string{"docker": "exit 7", "git": "exit 7"})
	before := treeSnapshot(t, dir)
	t.Chdir(dir)

	err := runBuildWithOptions(context.Background(),
		jsonDryRunOptions(specPath, DockerComposeSpecType), staticToken())
	require.NoError(t, err)
	rr.assertNoViolations(t)

	// No source file modified, none added, none removed.
	require.Equal(t, before, treeSnapshot(t, dir), "dry run modified the input tree")

	// No Docker image build/push and no Git command.
	require.Empty(t, shimInvocations(t, logPath), "dry run invoked a subprocess")

	// No account onboarding or artifact publication.
	require.Zero(t, rr.countPath(pathDeploymentArtifact))
	for _, rec := range rr.recorded() {
		require.False(t, strings.HasPrefix(rec.Path, "/2022-09-01-00/accountconfig") && rec.Method != http.MethodGet,
			"dry run issued a mutating account request: %s", rec)
	}

	validations := rr.find(pathValidateSpec)
	require.Len(t, validations, 1)
	body := validations[0].decodeBody(t)

	// $file reference was resolved relative to the spec file's directory.
	sent, decErr := base64.StdEncoding.DecodeString(body["fileContent"].(string))
	require.NoError(t, decErr)
	require.Contains(t, string(sent), "embedded: from-file-reference")

	// Compose config/secret files keep path semantics: their bytes are read from
	// the declared relative paths and base64-encoded.
	configs, ok := body["configs"].(map[string]any)
	require.True(t, ok, "configs missing from request body: %v", body)
	require.Equal(t, base64.StdEncoding.EncodeToString([]byte("config-file-content\n")), configs["app_config"])

	secrets, ok := body["secrets"].(map[string]any)
	require.True(t, ok, "secrets missing from request body: %v", body)
	require.Equal(t, base64.StdEncoding.EncodeToString([]byte("secret-file-content\n")), secrets["app_secret"])
}

// TestBuildDryRunImageGeneratedSpecUsesControlledImageReader covers the
// image-generated spec branch with a controlled image metadata reader (the local
// server answers the compose-gen endpoints).
//
// CONVERTED. It used to record that this branch reached the normal build
// endpoint with dryrun=true. The generated spec is now validated instead.
//
// The two compose-gen calls remain, and are the only non-validation requests any
// dry run makes. Both are reads: ows-orchestration's
// ComposeGen.GenerateComposeSpecFromContainerImage checks access, reads image
// metadata and marshals an in-memory compose project — no database write, no
// workflow, no image push — so this branch can produce a candidate without
// mutation and does not need the "unsupported validation" refusal that 04 §"CTL
// execution algorithm" reserves for branches that cannot.
func TestBuildDryRunImageGeneratedSpecUsesControlledImageReader(t *testing.T) {
	rr := startValidationServerAllowing(t, allowComposeGenReads,
		func(_ int, _ recordedRequest, w http.ResponseWriter) {
			writeJSON(w, validationResult(statusValid, nil))
		})

	dir := t.TempDir()
	logPath := execShim(t, map[string]string{"docker": "exit 7", "git": "exit 7"})
	t.Chdir(dir)

	opts := jsonDryRunOptions("", DockerComposeSpecType)
	opts.imageUrl = "docker.io/library/nginx:1.27"

	err := runBuildWithOptions(context.Background(), opts, staticToken())
	require.NoError(t, err)
	rr.assertNoViolations(t)

	require.Equal(t, []string{
		"GET " + pathComposeGenImage,
		"POST " + pathComposeGenImage,
		"POST " + pathValidateSpec,
	}, rr.sequence(), "recorded request sequence for build --image --dry-run")
	t.Logf("image-generated dry-run request sequence: %s", strings.Join(rr.sequence(), " | "))

	require.Zero(t, rr.countPath(pathBuildComposeSpec), "no build request")
	require.Empty(t, shimInvocations(t, logPath), "image dry run invoked a subprocess")
	require.Empty(t, treeSnapshot(t, dir), "image dry run wrote files into the working directory")
}

// TestBuildDryRunJSONOutputIsSingleJSONDocument asserts the stdout contract from
// 04 §"Output and logging": exactly one JSON document, carrying the validation
// result and none of the build vocabulary.
//
// CONVERTED. The "exactly one JSON document" property is the part that was true
// before and after; the payload assertions are new.
func TestBuildDryRunJSONOutputIsSingleJSONDocument(t *testing.T) {
	rr := startValidationServerAllowing(t, nil, func(_ int, _ recordedRequest, w http.ResponseWriter) {
		writeJSON(w, validationResult(statusValid, nil))
	})

	dir := t.TempDir()
	specPath := writeFixture(t, dir, "omnistrate-compose.yaml", minimalComposeSpec)
	t.Chdir(dir)

	var runErr error
	stdout := captureStdout(t, func() {
		runErr = runBuildWithOptions(context.Background(),
			jsonDryRunOptions(specPath, DockerComposeSpecType), staticToken())
	})
	require.NoError(t, runErr)
	rr.assertNoViolations(t)

	requireSingleJSONDocument(t, stdout)

	var result map[string]any
	require.NoError(t, json.Unmarshal([]byte(stdout), &result))
	require.Equal(t, statusValid, result["status"])

	// Required arrays are always present, never null.
	for _, key := range []string{"checks", "diagnostics", "requiredArtifacts", "validatedArtifacts"} {
		require.NotNil(t, result[key], "%s must be an array, not null: %s", key, stdout)
	}

	// None of the build vocabulary 04 forbids.
	require.NotContains(t, stdout, "service built")
	require.NotContains(t, stdout, "Successfully built")
	require.NotContains(t, stdout, "product-tier?serviceId=")
	require.NotContains(t, stdout, "\"version\":")
	require.NotContains(t, stdout, "Next steps")
	t.Logf("dry-run JSON stdout: %s", strings.TrimSpace(stdout))
}

// requireSingleJSONDocument asserts stdout holds exactly one JSON value.
func requireSingleJSONDocument(t *testing.T, stdout string) {
	t.Helper()
	dec := json.NewDecoder(strings.NewReader(stdout))
	var first any
	require.NoError(t, dec.Decode(&first), "stdout is not valid JSON: %q", stdout)

	var extra any
	err := dec.Decode(&extra)
	require.ErrorIs(t, err, io.EOF, "stdout contained more than one JSON document: %q", stdout)
}

// ---------------------------------------------------------------------------
// Preprocessing: the read-only route never writes into the source tree
// ---------------------------------------------------------------------------

const envFileComposeSpec = `services:
  api:
    image: docker.io/library/nginx:1.27
    env_file:
      - ./api.env
`

// TestDryRunValidationLeavesTheSpecTmpSentinelAlone is the counterpart of
// TestBuildDryRunRenderFileOverwritesSpecTmpSentinel: a sentinel is placed at
// the path RenderFile would clobber, and the dry-run route must leave it
// untouched in both the success and the failure case, while still resolving the
// env file against the original project directory.
func TestDryRunValidationLeavesTheSpecTmpSentinelAlone(t *testing.T) {
	const sentinel = "SENTINEL-DO-NOT-TOUCH\n"

	t.Run("DockerConfigSucceeds", func(t *testing.T) {
		rr := startValidationServerAllowing(t, nil, func(_ int, _ recordedRequest, w http.ResponseWriter) {
			writeJSON(w, validationResult(statusValid, nil))
		})

		dir := t.TempDir()
		specPath := writeFixture(t, dir, "omnistrate-compose.yaml", envFileComposeSpec)
		writeFixture(t, dir, "api.env", "KEY=value\n")
		tmpPath := filepath.Join(dir, "omnistrate-compose.yaml.tmp")
		require.NoError(t, os.WriteFile(tmpPath, []byte(sentinel), 0o600))

		// A controlled `docker` that records its arguments and echoes the file
		// it was handed. "$5" is the -f argument once --project-directory <dir>
		// precedes it.
		logPath := execShim(t, map[string]string{"docker": `cat "$5"`})
		before := treeSnapshot(t, dir)
		t.Chdir(dir)

		require.NoError(t, runBuildWithOptions(context.Background(),
			jsonDryRunOptions(specPath, DockerComposeSpecType), staticToken()))
		rr.assertNoViolations(t)

		require.Equal(t, before, treeSnapshot(t, dir), "the dry run modified the input tree")
		leftover, readErr := os.ReadFile(tmpPath) //nolint:gosec // test-owned temp tree
		require.NoError(t, readErr)
		require.Equal(t, sentinel, string(leftover), "the user's %s was overwritten", tmpPath)

		// The project directory is preserved so relative env_file/config/secret
		// references still resolve against the user's project.
		invocations := shimInvocations(t, logPath)
		require.Len(t, invocations, 1)
		require.Contains(t, invocations[0], "--project-directory "+dir)
		require.NotContains(t, invocations[0], dir+"/omnistrate-compose.yaml.tmp",
			"the temporary file must live outside the source tree")
	})

	t.Run("DockerConfigFails", func(t *testing.T) {
		rr := startValidationServerAllowing(t, nil, func(_ int, _ recordedRequest, w http.ResponseWriter) {
			writeJSON(w, validationResult(statusValid, nil))
		})

		dir := t.TempDir()
		specPath := writeFixture(t, dir, "omnistrate-compose.yaml", envFileComposeSpec)
		writeFixture(t, dir, "api.env", "KEY=value\n")
		tmpPath := filepath.Join(dir, "omnistrate-compose.yaml.tmp")
		require.NoError(t, os.WriteFile(tmpPath, []byte(sentinel), 0o600))

		execShim(t, map[string]string{"docker": "echo 'boom' >&2; exit 1"})
		before := treeSnapshot(t, dir)
		t.Chdir(dir)

		err := runBuildWithOptions(context.Background(),
			jsonDryRunOptions(specPath, DockerComposeSpecType), staticToken())
		require.Error(t, err, "a rendering failure must surface as an error")

		require.Equal(t, before, treeSnapshot(t, dir),
			"a failed dry run must leave nothing behind in the input tree")
		leftover, readErr := os.ReadFile(tmpPath) //nolint:gosec // test-owned temp tree
		require.NoError(t, readErr)
		require.Equal(t, sentinel, string(leftover))
		require.Zero(t, rr.countPath(pathValidateSpec), "no request is sent when rendering fails")
	})
}

// TestDryRunValidationRenderingMatchesRenderFile pins renderFileForValidation to
// the same output as RenderFile so the read-only preprocessing cannot drift from
// what a real build sends.
func TestDryRunValidationRenderingMatchesRenderFile(t *testing.T) {
	disablePrintErrorExit(t)

	specDir := t.TempDir()
	specPath := writeFixture(t, specDir, "omnistrate-compose.yaml", envFileComposeSpec)
	writeFixture(t, specDir, "api.env", "KEY=value\n")

	// Both renderers get the same controlled `docker`, which echoes the compose
	// file it was handed. RenderFile puts it at $3; the validation renderer puts
	// it at $5 because --project-directory precedes -f.
	execShim(t, map[string]string{"docker": `if [ "$2" = "--project-directory" ]; then cat "$5"; else cat "$3"; fi`})

	viaRenderFile, err := RenderFile([]byte(envFileComposeSpec), specDir, specPath, nil, nil)
	require.NoError(t, err)

	viaValidation, err := renderFileForValidation([]byte(envFileComposeSpec), specDir, specPath)
	require.NoError(t, err)

	require.Equal(t, string(viaRenderFile), string(viaValidation),
		"the read-only renderer must produce the same bytes as RenderFile")
}

// TestDryRunComposeEnvelopeMatchesRealBuild proves the dry run validates exactly
// the bytes, configs and secrets a real build would send. prepareComposeValidationContent
// reproduces BuildService's compose preprocessing; this pins the two together.
func TestDryRunComposeEnvelopeMatchesRealBuild(t *testing.T) {
	const composeWithConfigs = `services:
  api:
    image: docker.io/library/nginx:1.27
configs:
  app_config:
    file: ./conf/app.conf
secrets:
  app_secret:
    file: ./conf/app.secret
`
	dir := t.TempDir()
	specPath := writeFixture(t, dir, "omnistrate-compose.yaml", composeWithConfigs)
	writeFixture(t, dir, "conf/app.conf", "config-file-content\n")
	writeFixture(t, dir, "conf/app.secret", "secret-file-content\n")

	// Capture what the legacy build endpoint receives.
	legacy := startRecordingServer(t)
	legacy.setRoute(legacyRoutes(nil))
	t.Chdir(dir)

	opts := jsonDryRunOptions(specPath, DockerComposeSpecType)
	opts.dryRun = false
	require.NoError(t, runBuildWithOptions(context.Background(), opts, staticToken()))
	legacy.assertNoViolations(t)

	buildBody := legacy.find(pathBuildComposeSpec)[0].decodeBody(t)

	// Now capture what the validation endpoint receives.
	validation := startValidationServerAllowing(t, nil, func(_ int, _ recordedRequest, w http.ResponseWriter) {
		writeJSON(w, validationResult(statusValid, nil))
	})
	require.NoError(t, runBuildWithOptions(context.Background(),
		jsonDryRunOptions(specPath, DockerComposeSpecType), staticToken()))
	validation.assertNoViolations(t)

	validateBody := validation.find(pathValidateSpec)[0].decodeBody(t)

	require.Equal(t, buildBody["fileContent"], validateBody["fileContent"],
		"the validated bytes must be the bytes a real build would send")
	require.Equal(t, buildBody["configs"], validateBody["configs"])
	require.Equal(t, buildBody["secrets"], validateBody["secrets"])
	require.NotEmpty(t, validateBody["configs"])
	require.NotEmpty(t, validateBody["secrets"])
}

// ---------------------------------------------------------------------------
// The backend is not trusted to choose which local files are read
// ---------------------------------------------------------------------------

// TestDryRunValidationRefusesUndeclaredServerPaths is the hostile-server case
// from 04 §"CTL execution algorithm" step 5. Each sub-case has a server that
// answers the discovery request by demanding local content, and each demand must
// be refused before anything is opened.
func TestDryRunValidationRefusesUndeclaredServerPaths(t *testing.T) {
	cases := []struct {
		name        string
		spec        string
		logicalPath string
		wantErr     string
	}{{
		// The attack 04 names explicitly: an unrelated in-workspace credentials
		// directory that happens to be inside cwd.
		name:        "UnrelatedDirectoryInsideTheWorkingDirectory",
		spec:        servicePlanSpecWithLocalTerraform,
		logicalPath: "credentials",
		wantErr:     "does not declare as a local artifact source",
	}, {
		name:        "AbsolutePath",
		spec:        servicePlanSpecWithLocalTerraform,
		logicalPath: "/etc",
		wantErr:     "is absolute",
	}, {
		name:        "ParentEscape",
		spec:        servicePlanSpecWithLocalTerraform,
		logicalPath: "../elsewhere",
		wantErr:     "escapes the working directory",
	}, {
		name:        "EscapeHiddenInsideAnAllowedPrefix",
		spec:        servicePlanSpecWithLocalTerraform,
		logicalPath: "terraform/network/../../../elsewhere",
		wantErr:     "escapes the working directory",
	}, {
		name:        "BackslashPath",
		spec:        servicePlanSpecWithLocalTerraform,
		logicalPath: `terraform\network`,
		wantErr:     "contains a backslash",
	}, {
		// A compose specification declares no local artifact source at all, so
		// every requirement against one is refused.
		name:        "ComposeSpecificationHasNoLocalSources",
		spec:        "",
		logicalPath: "terraform/network",
		wantErr:     "declared: none",
	}}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			requirement := []map[string]any{{
				"logicalPath": tc.logicalPath,
				"uses": []map[string]any{{
					"resourceKey": "network",
					"kind":        "terraform",
					"path":        "/services/0",
				}},
			}}

			rr := startValidationServerAllowing(t, nil, func(_ int, _ recordedRequest, w http.ResponseWriter) {
				writeJSON(w, validationResult(statusIncomplete, requirement))
			})

			dir := t.TempDir()
			// A credentials directory that a hostile server might try to exfiltrate.
			writeFixture(t, dir, "credentials/aws.json", "{\"secret\":\"NEVER-SEND-THIS\"}\n")
			writeFixture(t, dir, "terraform/network/main.tf", "output \"id\" { value = \"x\" }\n")

			var opts buildOptions
			if tc.spec == "" {
				specPath := writeFixture(t, dir, "omnistrate-compose.yaml", minimalComposeSpec)
				t.Chdir(dir)
				opts = jsonDryRunOptions(specPath, DockerComposeSpecType)
			} else {
				specPath := writeFixture(t, dir, "spec.yaml", tc.spec)
				t.Chdir(dir)
				opts = jsonDryRunOptions(specPath, ServicePlanSpecType)
			}

			err := runBuildWithOptions(context.Background(), opts, staticToken())
			require.Error(t, err)
			require.Contains(t, err.Error(), tc.wantErr)

			rr.assertNoViolations(t)
			require.Equal(t, 1, rr.countPath(pathValidateSpec),
				"no second request is sent, so nothing was read or transmitted")

			// Belt and braces: the credential bytes never appear on the wire.
			for _, rec := range rr.recorded() {
				require.NotContains(t, string(rec.Body), "NEVER-SEND-THIS")
			}
		})
	}
}

// TestDryRunValidationRefusesSymlinkEscapes covers the case where the path IS
// declared but resolves outside the working directory through a symlink.
func TestDryRunValidationRefusesSymlinkEscapes(t *testing.T) {
	const specWithLinkedSource = `version: "1.0"
name: Dry Run Plan
services:
  - name: network
    terraformConfigurations:
      configurationPerCloudProvider:
        aws:
          terraformPath: /terraform
          artifactsLocalPath: ./linked
`
	requirement := []map[string]any{{
		"logicalPath": "linked",
		"uses": []map[string]any{{
			"resourceKey": "network",
			"kind":        "terraform",
			"path":        "/services/0",
		}},
	}}

	rr := startValidationServerAllowing(t, nil, func(_ int, _ recordedRequest, w http.ResponseWriter) {
		writeJSON(w, validationResult(statusIncomplete, requirement))
	})

	outside := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(outside, "secret.txt"), []byte("OUTSIDE-SECRET"), 0o600))

	dir := t.TempDir()
	specPath := writeFixture(t, dir, "spec.yaml", specWithLinkedSource)
	require.NoError(t, os.Symlink(outside, filepath.Join(dir, "linked")))
	t.Chdir(dir)

	err := runBuildWithOptions(context.Background(),
		jsonDryRunOptions(specPath, ServicePlanSpecType), staticToken())
	require.Error(t, err)
	require.Contains(t, err.Error(), "escapes base directory")

	rr.assertNoViolations(t)
	require.Equal(t, 1, rr.countPath(pathValidateSpec))
	for _, rec := range rr.recorded() {
		require.NotContains(t, string(rec.Body), "OUTSIDE-SECRET")
	}
}

// TestDryRunValidationPreservesTheExplicitRootWithoutBroadening asserts the
// canonical "." root behaviour from 04 step 5: a specification with no explicit
// Terraform source defaults to "./", which is canonical "." and means the
// working directory — never the user's home directory.
func TestDryRunValidationPreservesTheExplicitRootWithoutBroadening(t *testing.T) {
	requirement := []map[string]any{{
		"logicalPath": ".",
		"uses": []map[string]any{{
			"resourceKey": "network",
			"kind":        "terraform",
			"path":        "/services/0/terraformConfigurations/configurationPerCloudProvider/aws",
		}},
	}}

	rr := startValidationServerAllowing(t, nil, func(call int, rec recordedRequest, w http.ResponseWriter) {
		if call == 0 {
			writeJSON(w, validationResult(statusIncomplete, requirement))
			return
		}
		writeJSON(w, validationResultAcknowledging(statusValid, rec, requirement))
	})

	dir := t.TempDir()
	specPath := writeFixture(t, dir, "spec.yaml", minimalServicePlanSpec)
	writeFixture(t, dir, "main.tf", "output \"id\" { value = \"x\" }\n")
	t.Chdir(dir)

	require.NoError(t, runBuildWithOptions(context.Background(),
		jsonDryRunOptions(specPath, ServicePlanSpecType), staticToken()))
	rr.assertNoViolations(t)

	artifact := firstArtifact(t, rr.find(pathValidateSpec)[1].decodeBody(t))
	require.Equal(t, ".", artifact["logicalPath"], "the default root is canonical \".\"")

	names := tarEntryNames(t, decodeArchive(t, artifact))
	require.Contains(t, names, "main.tf")
	require.Contains(t, names, "spec.yaml")
}

// ---------------------------------------------------------------------------
// Second-request contract
// ---------------------------------------------------------------------------

func localTerraformFixture(t *testing.T) buildOptions {
	t.Helper()
	dir := t.TempDir()
	specPath := writeFixture(t, dir, "spec.yaml", servicePlanSpecWithLocalTerraform)
	writeFixture(t, dir, "terraform/network/main.tf", "output \"id\" { value = \"x\" }\n")
	t.Chdir(dir)
	return jsonDryRunOptions(specPath, ServicePlanSpecType)
}

func terraformRequirement() []map[string]any {
	return []map[string]any{{
		"logicalPath": "terraform/network",
		"uses": []map[string]any{{
			"resourceKey": "network",
			"kind":        "terraform",
			"provider":    "aws",
			"path":        "/services/0/terraformConfigurations/configurationPerCloudProvider/aws",
		}},
	}}
}

// TestDryRunValidationRejectsAChangedInputDigest — 04 step 8.
func TestDryRunValidationRejectsAChangedInputDigest(t *testing.T) {
	requirement := terraformRequirement()
	rr := startValidationServerAllowing(t, nil, func(call int, rec recordedRequest, w http.ResponseWriter) {
		if call == 0 {
			writeJSON(w, validationResult(statusIncomplete, requirement))
			return
		}
		result := validationResultAcknowledging(statusValid, rec, requirement)
		result["inputDigest"] = "1111111111111111111111111111111111111111111111111111111111111111"
		writeJSON(w, result)
	})

	err := runBuildWithOptions(context.Background(), localTerraformFixture(t), staticToken())
	require.Error(t, err)
	require.Contains(t, err.Error(), "different input for the second request")

	rr.assertNoViolations(t)
	require.Equal(t, 2, rr.countPath(pathValidateSpec), "exactly one retry, no loop")
}

// TestDryRunValidationRejectsAThirdContentRequest — 04 step 8: "INCOMPLETE or
// another request for content after request two returns nonzero; no unbounded
// retry loop and no mutation fallback."
func TestDryRunValidationRejectsAThirdContentRequest(t *testing.T) {
	requirement := terraformRequirement()
	rr := startValidationServerAllowing(t, nil, func(_ int, _ recordedRequest, w http.ResponseWriter) {
		// The server keeps asking for the same content, for ever.
		writeJSON(w, validationResult(statusIncomplete, requirement))
	})

	var runErr error
	stdout := captureStdout(t, func() {
		runErr = runBuildWithOptions(context.Background(), localTerraformFixture(t), staticToken())
	})
	require.Error(t, runErr)
	require.Contains(t, runErr.Error(), "asked for more local content after the final request")

	rr.assertNoViolations(t)
	require.Equal(t, 2, rr.countPath(pathValidateSpec), "no unbounded retry loop")
	require.Zero(t, rr.countPath(pathBuildServicePlanSpec), "no mutation fallback")
	require.Zero(t, rr.countPath(pathPrepareServicePlanSpec))
	requireSingleJSONDocument(t, stdout)
}

// TestDryRunValidationRejectsUnacknowledgedArtifacts — 04 step 8: a VALID result
// must acknowledge the supplied digest and every required use.
func TestDryRunValidationRejectsUnacknowledgedArtifacts(t *testing.T) {
	requirement := terraformRequirement()

	t.Run("DigestNotAcknowledged", func(t *testing.T) {
		rr := startValidationServerAllowing(t, nil, func(call int, _ recordedRequest, w http.ResponseWriter) {
			if call == 0 {
				writeJSON(w, validationResult(statusIncomplete, requirement))
				return
			}
			// VALID, but the supplied content is never mentioned.
			writeJSON(w, validationResult(statusValid, nil))
		})

		err := runBuildWithOptions(context.Background(), localTerraformFixture(t), staticToken())
		require.Error(t, err)
		require.Contains(t, err.Error(), "never acknowledged the content supplied")
		rr.assertNoViolations(t)
	})

	t.Run("RequiredUseNotChecked", func(t *testing.T) {
		rr := startValidationServerAllowing(t, nil, func(call int, rec recordedRequest, w http.ResponseWriter) {
			if call == 0 {
				writeJSON(w, validationResult(statusIncomplete, requirement))
				return
			}
			// The digest is acknowledged but the use that required it is not.
			writeJSON(w, validationResultAcknowledging(statusValid, rec, nil))
		})

		err := runBuildWithOptions(context.Background(), localTerraformFixture(t), staticToken())
		require.Error(t, err)
		require.Contains(t, err.Error(), "did not check")
		rr.assertNoViolations(t)
	})
}

// TestDryRunValidationIncompleteAfterContentIsNonZero — the second response can
// still be INCOMPLETE for reasons unrelated to artifacts; that is a nonzero exit
// with one JSON document, never a fallback to the build endpoint.
func TestDryRunValidationIncompleteAfterContentIsNonZero(t *testing.T) {
	requirement := terraformRequirement()
	rr := startValidationServerAllowing(t, nil, func(call int, rec recordedRequest, w http.ResponseWriter) {
		if call == 0 {
			writeJSON(w, validationResult(statusIncomplete, requirement))
			return
		}
		result := validationResultAcknowledging(statusIncomplete, rec, requirement)
		writeJSON(w, result)
	})

	var runErr error
	stdout := captureStdout(t, func() {
		runErr = runBuildWithOptions(context.Background(), localTerraformFixture(t), staticToken())
	})
	require.Error(t, runErr)
	require.Contains(t, runErr.Error(), "validation incomplete")

	rr.assertNoViolations(t)
	require.Equal(t, 2, rr.countPath(pathValidateSpec))
	requireSingleJSONDocument(t, stdout)

	var result map[string]any
	require.NoError(t, json.Unmarshal([]byte(stdout), &result))
	require.Equal(t, statusIncomplete, result["status"])
}

// ---------------------------------------------------------------------------
// Archive semantics
// ---------------------------------------------------------------------------

// TestDryRunValidationDoesNotRewrapExistingArchives — 04 step 6.
func TestDryRunValidationDoesNotRewrapExistingArchives(t *testing.T) {
	requirement := []map[string]any{{
		"logicalPath": "artifacts/bundle.tar.gz",
		"uses": []map[string]any{{
			"resourceKey": "chart",
			"kind":        "helm",
			"path":        "/services/0/helmChartConfiguration",
		}},
	}}

	rr := startValidationServerAllowing(t, nil, func(call int, rec recordedRequest, w http.ResponseWriter) {
		if call == 0 {
			writeJSON(w, validationResult(statusIncomplete, requirement))
			return
		}
		writeJSON(w, validationResultAcknowledging(statusValid, rec, requirement))
	})

	// A real, minimal tar.gz written by the same writer the normal build uses.
	var raw bytes.Buffer
	gzw := gzip.NewWriter(&raw)
	tw := tar.NewWriter(gzw)
	payload := []byte("chart contents\n")
	require.NoError(t, tw.WriteHeader(&tar.Header{Name: "Chart.yaml", Mode: 0o600, Size: int64(len(payload))}))
	_, err := tw.Write(payload)
	require.NoError(t, err)
	require.NoError(t, tw.Close())
	require.NoError(t, gzw.Close())

	dir := t.TempDir()
	specPath := writeFixture(t, dir, "spec.yaml", servicePlanSpecWithLocalArchive)
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "artifacts"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "artifacts", "bundle.tar.gz"), raw.Bytes(), 0o600))
	t.Chdir(dir)

	require.NoError(t, runBuildWithOptions(context.Background(),
		jsonDryRunOptions(specPath, ServicePlanSpecType), staticToken()))
	rr.assertNoViolations(t)

	artifact := firstArtifact(t, rr.find(pathValidateSpec)[1].decodeBody(t))
	require.Equal(t, raw.Bytes(), decodeArchive(t, artifact),
		"an existing .tar.gz must be transmitted byte-for-byte, not rewrapped")
}

// TestDryRunValidationRejectsSymlinksInsideAnArtifact — 04 step 6: unsupported
// members are detected explicitly rather than silently omitted.
func TestDryRunValidationRejectsSymlinksInsideAnArtifact(t *testing.T) {
	requirement := terraformRequirement()
	rr := startValidationServerAllowing(t, nil, func(_ int, _ recordedRequest, w http.ResponseWriter) {
		writeJSON(w, validationResult(statusIncomplete, requirement))
	})

	dir := t.TempDir()
	specPath := writeFixture(t, dir, "spec.yaml", servicePlanSpecWithLocalTerraform)
	writeFixture(t, dir, "terraform/network/main.tf", "output \"id\" { value = \"x\" }\n")
	require.NoError(t, os.Symlink("main.tf", filepath.Join(dir, "terraform", "network", "alias.tf")))
	t.Chdir(dir)

	err := runBuildWithOptions(context.Background(),
		jsonDryRunOptions(specPath, ServicePlanSpecType), staticToken())
	require.Error(t, err)
	require.Contains(t, err.Error(), "cannot transport symlinks")
	require.Contains(t, err.Error(), "alias.tf")

	rr.assertNoViolations(t)
	require.Equal(t, 1, rr.countPath(pathValidateSpec),
		"an archive with an unsupported member is never sent")
}

// TestBoundedArchiveEnforcesLimitsWhileWriting drives createBoundedTarGz
// directly with tiny budgets, so the limits are proven to fail during the write
// rather than after an unbounded allocation.
func TestBoundedArchiveEnforcesLimitsWhileWriting(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"a.tf", "b.tf", "c.tf"} {
		require.NoError(t, os.WriteFile(filepath.Join(dir, name), bytes.Repeat([]byte("x"), 4096), 0o600))
	}

	t.Run("EntryBudget", func(t *testing.T) {
		budget := &archiveBudget{
			remainingCompressed: 1 << 20,
			remainingExtracted:  1 << 20,
			remainingEntries:    2,
			maxMemberPathBytes:  1024,
		}
		_, err := createBoundedTarGz(dir, "terraform", budget)
		require.ErrorIs(t, err, errArtifactLimitExceeded)
		require.Contains(t, err.Error(), "too many archive entries")
	})

	t.Run("ExtractedByteBudget", func(t *testing.T) {
		budget := &archiveBudget{
			remainingCompressed: 1 << 20,
			remainingExtracted:  1000, // less than one file
			remainingEntries:    100,
			maxMemberPathBytes:  1024,
		}
		_, err := createBoundedTarGz(dir, "terraform", budget)
		require.ErrorIs(t, err, errArtifactLimitExceeded)
		require.Contains(t, err.Error(), "uncompressed artifact size")
	})

	t.Run("CompressedByteBudget", func(t *testing.T) {
		budget := &archiveBudget{
			remainingCompressed: 8, // smaller than a gzip header
			remainingExtracted:  1 << 20,
			remainingEntries:    100,
			maxMemberPathBytes:  1024,
		}
		_, err := createBoundedTarGz(dir, "terraform", budget)
		require.ErrorIs(t, err, errArtifactLimitExceeded)
		require.Contains(t, err.Error(), "compressed archive size")
	})

	t.Run("MemberPathBudget", func(t *testing.T) {
		budget := &archiveBudget{
			remainingCompressed: 1 << 20,
			remainingExtracted:  1 << 20,
			remainingEntries:    100,
			maxMemberPathBytes:  2,
		}
		_, err := createBoundedTarGz(dir, "terraform", budget)
		require.ErrorIs(t, err, errArtifactLimitExceeded)
		require.Contains(t, err.Error(), "archive member path")
	})

	t.Run("WithinBudget", func(t *testing.T) {
		budget := newArchiveBudget(defaultValidationLimits())
		archive, err := createBoundedTarGz(dir, "terraform", budget)
		require.NoError(t, err)
		require.Equal(t, []string{"a.tf", "b.tf", "c.tf"}, tarEntryNames(t, archive))
	})
}

// ---------------------------------------------------------------------------
// Pure helpers
// ---------------------------------------------------------------------------

func TestCanonicalValidationPath(t *testing.T) {
	cases := []struct {
		raw     string
		want    string
		wantErr string
	}{
		{raw: "terraform/network", want: "terraform/network"},
		{raw: "./terraform/network", want: "terraform/network"},
		{raw: "  ./terraform/network  ", want: "terraform/network"},
		{raw: "./", want: "."},
		{raw: ".", want: "."},
		{raw: "terraform//network", want: "terraform/network"},
		{raw: "terraform/./network", want: "terraform/network"},
		{raw: "", wantErr: "is empty"},
		{raw: "   ", wantErr: "is empty"},
		{raw: "/terraform", wantErr: "is absolute"},
		{raw: "//terraform", wantErr: "is absolute"},
		{raw: "..", wantErr: "escapes the working directory"},
		{raw: "../x", wantErr: "escapes the working directory"},
		{raw: "a/../../b", wantErr: "escapes the working directory"},
		// A sibling-prefix path is legitimate and must NOT be rejected.
		{raw: "terraform-other/network", want: "terraform-other/network"},
		{raw: "a/b\x00c", wantErr: "NUL"},
		{raw: `a\b`, wantErr: "backslash"},
	}

	for _, tc := range cases {
		t.Run(fmt.Sprintf("%q", tc.raw), func(t *testing.T) {
			got, err := canonicalValidationPath(tc.raw, 1024)
			if tc.wantErr != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.wantErr)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.want, got)
		})
	}

	t.Run("PathLengthLimit", func(t *testing.T) {
		_, err := canonicalValidationPath(strings.Repeat("a", 40), 16)
		require.ErrorIs(t, err, errArtifactLimitExceeded)
	})
}

func TestCollectDeclaredArtifactPaths(t *testing.T) {
	t.Run("DefaultTerraformRootIsCanonicalDot", func(t *testing.T) {
		allowed := collectDeclaredArtifactPaths(ServicePlanSpecType, []byte(minimalServicePlanSpec), 1024)
		require.Equal(t, []string{"."}, sortedPaths(allowed))
	})

	t.Run("ExplicitLocalPaths", func(t *testing.T) {
		const spec = `services:
  - name: a
    terraformConfigurations:
      configurationPerCloudProvider:
        aws:
          artifactsLocalPath: ./tf/aws
        gcp:
          artifactRelativePath: tf/gcp
      configurationPerOnPremPlatform:
        GenericPlatform:
          artifactsLocalPath: ./tf/onprem
  - name: b
    helmChartConfiguration:
      artifactsLocalPath: ./charts/demo
  - name: c
    kustomizeConfiguration:
      artifactsLocalPath: ./kustomize
  - name: d
    operatorCRDConfiguration:
      artifactsLocalPath: ./operator
`
		allowed := collectDeclaredArtifactPaths(ServicePlanSpecType, []byte(spec), 1024)
		require.Equal(t, []string{
			"charts/demo", "kustomize", "operator", "tf/aws", "tf/gcp", "tf/onprem",
		}, sortedPaths(allowed))
	})

	t.Run("GitBackedSourcesDeclareNoLocalPath", func(t *testing.T) {
		const spec = `services:
  - name: a
    terraformConfigurations:
      configurationPerCloudProvider:
        aws:
          gitConfiguration:
            repositoryUrl: https://example.invalid/repo.git
  - name: b
    kustomizeConfiguration:
      gitConfiguration:
        repositoryUrl: https://example.invalid/repo.git
      artifactsLocalPath: ./ignored
`
		allowed := collectDeclaredArtifactPaths(ServicePlanSpecType, []byte(spec), 1024)
		require.Empty(t, sortedPaths(allowed))
	})

	t.Run("ContradictoryExclusiveFieldsDeclareNothing", func(t *testing.T) {
		const spec = `services:
  - name: a
    terraformConfigurations:
      configurationPerCloudProvider:
        aws:
          artifactsLocalPath: ./one
          artifactRelativePath: two
`
		allowed := collectDeclaredArtifactPaths(ServicePlanSpecType, []byte(spec), 1024)
		require.Empty(t, sortedPaths(allowed))
	})

	t.Run("ComposeDeclaresNothing", func(t *testing.T) {
		allowed := collectDeclaredArtifactPaths(DockerComposeSpecType, []byte(minimalComposeSpec), 1024)
		require.Empty(t, sortedPaths(allowed))
	})

	t.Run("EscapingDeclarationsAreNotAllowlisted", func(t *testing.T) {
		const spec = `services:
  - name: a
    terraformConfigurations:
      configurationPerCloudProvider:
        aws:
          artifactsLocalPath: ../../etc
`
		allowed := collectDeclaredArtifactPaths(ServicePlanSpecType, []byte(spec), 1024)
		require.Empty(t, sortedPaths(allowed))
	})
}

func TestValidationLimitsRespectTheSmallerServerValue(t *testing.T) {
	limits := defaultValidationLimits().withServerLimits(&openapiclient.ValidationLimits{
		MaxTotalCompressedArtifactBytes: 1024,
		MaxArtifacts:                    1024, // larger than the client default
		MaxArchiveEntries:               0,    // unset, must be ignored
	})
	require.Equal(t, int64(1024), limits.MaxTotalCompressedArtifactBytes)
	require.Equal(t, defaultMaxArtifacts, limits.MaxArtifacts)
	require.Equal(t, defaultMaxArchiveEntries, limits.MaxArchiveEntries)
}

// ---------------------------------------------------------------------------
// Debug logging must not leak the payload
// ---------------------------------------------------------------------------

// TestDryRunValidationDebugLogsOmitSecretSentinels runs the whole dry-run route
// with debug logging on and unique sentinels in the specification, in a compose
// secret and inside the local artifact, then asserts that none of them — nor the
// token, nor the base64 archive — reaches the captured log, while the request
// metadata still does. 04 §"Output and logging".
func TestDryRunValidationDebugLogsOmitSecretSentinels(t *testing.T) {
	const (
		specSentinel     = "SENTINEL-SPEC-4f1a9c2b"
		artifactSentinel = "SENTINEL-ARTIFACT-77c31d0e"
	)

	spec := fmt.Sprintf(`version: "1.0"
name: Dry Run Plan
description: %s
services:
  - name: network
    terraformConfigurations:
      configurationPerCloudProvider:
        aws:
          terraformPath: /terraform/network
          artifactsLocalPath: ./terraform/network
`, specSentinel)

	requirement := terraformRequirement()
	rr := startValidationServerAllowing(t, nil, func(call int, rec recordedRequest, w http.ResponseWriter) {
		if call == 0 {
			writeJSON(w, validationResult(statusIncomplete, requirement))
			return
		}
		writeJSON(w, validationResultAcknowledging(statusValid, rec, requirement))
	})

	dir := t.TempDir()
	specPath := writeFixture(t, dir, "spec.yaml", spec)
	writeFixture(t, dir, "terraform/network/main.tf",
		fmt.Sprintf("variable \"token\" { default = \"%s\" }\n", artifactSentinel))
	t.Chdir(dir)

	logs := captureDebugLogs(t, func() {
		require.NoError(t, runBuildWithOptions(context.Background(),
			jsonDryRunOptions(specPath, ServicePlanSpecType), staticToken()))
	})
	rr.assertNoViolations(t)

	require.NotEmpty(t, logs, "debug logging produced no output, so this test would prove nothing")
	require.Contains(t, logs, pathValidateSpec, "request metadata must still be logged")
	require.Contains(t, logs, "[REDACTED]", "the Authorization header must be redacted")
	require.Contains(t, logs, "body omitted")

	forbidden := map[string]string{
		"the specification sentinel":     specSentinel,
		"the base64 specification":       base64.StdEncoding.EncodeToString([]byte(spec)),
		"the artifact sentinel":          artifactSentinel,
		"the API token":                  fakeToken,
		"the Authorization header value": "Bearer " + fakeToken,
	}
	for what, sentinel := range forbidden {
		require.NotContains(t, logs, sentinel, "%s appeared in the debug log", what)
	}

	// The base64 archive itself must not appear either.
	artifact := firstArtifact(t, rr.find(pathValidateSpec)[1].decodeBody(t))
	archiveContent, ok := artifact["archiveContent"].(string)
	require.True(t, ok)
	require.NotContains(t, logs, archiveContent, "the base64 archive appeared in the debug log")
}

// captureDebugLogs redirects the global zerolog logger into a buffer and turns
// on the debug level that gates the request/response dumps.
func captureDebugLogs(t *testing.T, fn func()) string {
	t.Helper()
	t.Setenv("OMNISTRATE_LOG_LEVEL", "debug")

	var buf bytes.Buffer
	original := log.Logger
	log.Logger = zerolog.New(&buf).Level(zerolog.DebugLevel)
	t.Cleanup(func() { log.Logger = original })

	fn()
	return buf.String()
}
