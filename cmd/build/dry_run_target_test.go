package build

// Target contract for `omnistrate-ctl build --dry-run`, per
// dry-run-implementation-specs/02-regression-tests.md §"CLI request contract
// tests" and 04-local-artifacts-and-ctl.md §"CTL execution algorithm" /
// §"Output and logging".
//
// These tests were written before the implementation and were held behind a
// `dryrun_target` build tag while they were red. Spec 04 has landed, so the tag
// is gone and they run by default:
//
//	go test ./cmd/build -run TestDryRunTarget -count=1 -v
//
// Three assumptions in the original file did not survive contact with the
// generated SDK and the local path allowlist; each is called out at its test:
// the artifact cases needed specifications that actually DECLARE the local path
// the server asks for, because a path no local source declares is refused before
// anything is read.
//
// The server here accepts ONLY POST /2022-09-01-00/service/spec/validate.
// Every other request — prepare, either normal build endpoint, artifact upload,
// environment creation, release or promotion — is recorded as a violation and
// answered with an error, and the violation is asserted on the main test
// goroutine (never with require.FailNow inside the handler).

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// startValidationOnlyServer returns a recorder that permits only the new
// validation POST. respond produces the response for each validation request,
// indexed by call number (0-based).
func startValidationOnlyServer(t *testing.T, respond func(call int, rec recordedRequest, w http.ResponseWriter)) *requestRecorder {
	t.Helper()
	return startValidationServerAllowing(t, nil, respond)
}

// targetDryRunOptions writes the artifact-free ServicePlanSpec fixture into a
// fresh working directory and returns the JSON dry-run options for it.
func targetDryRunOptions(t *testing.T) buildOptions {
	t.Helper()
	dir := t.TempDir()
	specPath := writeFixture(t, dir, "spec.yaml", minimalServicePlanSpec)
	t.Chdir(dir)
	return jsonDryRunOptions(specPath, ServicePlanSpecType)
}

// ---------------------------------------------------------------------------
// Request shape
// ---------------------------------------------------------------------------

// TestDryRunTargetNoArtifactIssuesOneValidationRequest — 02: "no artifact (one
// validation request)".
func TestDryRunTargetNoArtifactIssuesOneValidationRequest(t *testing.T) {
	rr := startValidationOnlyServer(t, func(_ int, _ recordedRequest, w http.ResponseWriter) {
		writeJSON(w, validationResult(statusValid, nil))
	})

	opts := targetDryRunOptions(t)
	require.NoError(t, runBuildWithOptions(context.Background(), opts, staticToken()))
	rr.assertNoViolations(t)

	require.Equal(t, []string{"POST " + pathValidateSpec}, rr.sequence())

	body := rr.find(pathValidateSpec)[0].decodeBody(t)
	require.Equal(t, "service-plan", body["specType"])
	require.Equal(t, testServiceName, body["name"])
	require.Equal(t, base64.StdEncoding.EncodeToString([]byte(minimalServicePlanSpec)), body["fileContent"])
	require.NotContains(t, body, "dryrun", "the validation endpoint has no public dryrun flag")
}

// TestDryRunTargetArtifactsIssueAtMostTwoValidationRequests — 02: "artifacts (at
// most two)". First response asks for content, second validates it.
//
// RECONCILED: the original used minimalServicePlanSpec, which declares no local
// artifact source at all, so its only allowlisted path is the documented
// Terraform default root ".". A server asking for "terraform/network" against
// that spec is now refused. The fixture therefore declares the path explicitly.
func TestDryRunTargetArtifactsIssueAtMostTwoValidationRequests(t *testing.T) {
	requirement := []map[string]any{{
		"logicalPath": "terraform/network",
		"uses": []map[string]any{{
			"resourceKey": "network",
			"kind":        "terraform",
			"provider":    "aws",
			"path":        "/services/0/terraformConfigurations/configurationPerCloudProvider/aws",
		}},
	}}

	rr := startValidationOnlyServer(t, func(call int, rec recordedRequest, w http.ResponseWriter) {
		if call == 0 {
			writeJSON(w, validationResult(statusIncomplete, requirement))
			return
		}
		writeJSON(w, validationResultAcknowledging(statusValid, rec, requirement))
	})

	dir := t.TempDir()
	specPath := writeFixture(t, dir, "spec.yaml", servicePlanSpecWithLocalTerraform)
	writeFixture(t, dir, "terraform/network/main.tf", "output \"id\" { value = \"x\" }\n")
	t.Chdir(dir)

	opts := jsonDryRunOptions(specPath, ServicePlanSpecType)
	require.NoError(t, runBuildWithOptions(context.Background(), opts, staticToken()))
	rr.assertNoViolations(t)

	require.Equal(t, 2, rr.countPath(pathValidateSpec), "at most two validation requests")

	firstBody := rr.find(pathValidateSpec)[0].decodeBody(t)
	require.NotContains(t, firstBody, "artifacts", "the discovery request carries no artifacts")

	second := rr.find(pathValidateSpec)[1].decodeBody(t)
	artifacts, ok := second["artifacts"].([]any)
	require.True(t, ok, "second request must carry artifacts: %v", second)
	require.Len(t, artifacts, 1)

	artifact := artifacts[0].(map[string]any)
	require.Equal(t, "terraform/network", artifact["logicalPath"])
	require.Equal(t, "tar+gzip+base64", artifact["encoding"])
	require.NotEmpty(t, artifact["archiveContent"])
	require.Len(t, artifact["sha256"], 64)
	require.NotZero(t, artifact["compressedSizeBytes"])

	// The digest and size describe the exact decoded compressed bytes, and the
	// archive really contains the declared directory's contents, named relative
	// to that directory's own root.
	require.Equal(t, []string{"main.tf"}, tarEntryNames(t, decodeArchive(t, artifact)))

	// RECONCILED: the original compared body["inputDigest"] across the two
	// REQUESTS. inputDigest is a response field, so that comparison was
	// vacuously true (nil == nil). The envelope equality that actually matters
	// is asserted directly here, and the response-digest check the client
	// performs has its own test,
	// TestDryRunValidationRejectsAChangedInputDigest.
	delete(firstBody, "artifacts")
	delete(second, "artifacts")
	require.Equal(t, firstBody, second,
		"both requests must carry the same input envelope apart from the artifacts")
}

// ---------------------------------------------------------------------------
// Output and exit status
// ---------------------------------------------------------------------------

// TestDryRunTargetJSONOutputIsExactlyOneDocument — 02: "JSON/non-TTY ... stdout
// is one JSON document in JSON mode"; 04 §"Output and logging".
func TestDryRunTargetJSONOutputIsExactlyOneDocument(t *testing.T) {
	rr := startValidationOnlyServer(t, func(_ int, _ recordedRequest, w http.ResponseWriter) {
		writeJSON(w, validationResult(statusValid, nil))
	})

	opts := targetDryRunOptions(t)

	var runErr error
	stdout := captureStdout(t, func() {
		runErr = runBuildWithOptions(context.Background(), opts, staticToken())
	})
	require.NoError(t, runErr, "VALID must exit zero")
	rr.assertNoViolations(t)

	requireSingleJSONDocument(t, stdout)

	var result map[string]any
	require.NoError(t, json.Unmarshal([]byte(stdout), &result))
	require.Equal(t, statusValid, result["status"])
	require.NotContains(t, stdout, "service built")
	require.NotContains(t, stdout, "product-tier?serviceId=", "no release URL in dry-run output")
}

// TestDryRunTargetInvalidReturnsNonZero — 02: "INVALID".
func TestDryRunTargetInvalidReturnsNonZero(t *testing.T) {
	rr := startValidationOnlyServer(t, func(_ int, _ recordedRequest, w http.ResponseWriter) {
		writeJSON(w, validationResult(statusInvalid, nil))
	})

	opts := targetDryRunOptions(t)

	var runErr error
	stdout := captureStdout(t, func() {
		runErr = runBuildWithOptions(context.Background(), opts, staticToken())
	})
	require.Error(t, runErr, "INVALID must return a nonzero error")
	rr.assertNoViolations(t)

	requireSingleJSONDocument(t, stdout)
	require.Equal(t, 1, rr.countPath(pathValidateSpec))
}

// TestDryRunTargetIncompleteReturnsNonZero — 02: "INCOMPLETE". An INCOMPLETE
// result with no requiredArtifacts must not trigger a second request.
func TestDryRunTargetIncompleteReturnsNonZero(t *testing.T) {
	rr := startValidationOnlyServer(t, func(_ int, _ recordedRequest, w http.ResponseWriter) {
		writeJSON(w, validationResult(statusIncomplete, nil))
	})

	opts := targetDryRunOptions(t)

	var runErr error
	stdout := captureStdout(t, func() {
		runErr = runBuildWithOptions(context.Background(), opts, staticToken())
	})
	require.Error(t, runErr, "INCOMPLETE must return a nonzero error")
	rr.assertNoViolations(t)

	requireSingleJSONDocument(t, stdout)
	require.Equal(t, 1, rr.countPath(pathValidateSpec))
}

// ---------------------------------------------------------------------------
// Flags and interaction
// ---------------------------------------------------------------------------

// TestDryRunTargetInteractiveDoesNotPromptForPromotion — 02: "--interactive with
// input y (no promotion prompt or mutation)"; 04 step 9.
func TestDryRunTargetInteractiveDoesNotPromptForPromotion(t *testing.T) {
	rr := startValidationOnlyServer(t, func(_ int, _ recordedRequest, w http.ResponseWriter) {
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
	_, err = stdinW.WriteString("y\n")
	require.NoError(t, err)

	opts := targetDryRunOptions(t)
	opts.output = "table"
	opts.interactive = true

	require.NoError(t, runBuildWithOptions(context.Background(), opts, staticToken()))
	rr.assertNoViolations(t)

	require.Equal(t, []string{"POST " + pathValidateSpec}, rr.sequence(),
		"a dry run must not create an environment or promote even when the user answers y")
}

// TestDryRunTargetReleaseFlagsAreCandidateIntentOnly — 02: "release-description/
// name flags"; 04 step 9 ("carried as candidate intent").
func TestDryRunTargetReleaseFlagsAreCandidateIntentOnly(t *testing.T) {
	rr := startValidationOnlyServer(t, func(_ int, _ recordedRequest, w http.ResponseWriter) {
		writeJSON(w, validationResult(statusValid, nil))
	})

	opts := targetDryRunOptions(t)
	opts.release = true
	opts.releaseAsPreferred = true
	opts.releaseDescription = "v1.0.0-alpha"
	opts.releaseName = "legacy-name"
	opts.forceCreateServicePlanVersion = true

	require.NoError(t, runBuildWithOptions(context.Background(), opts, staticToken()))
	rr.assertNoViolations(t)

	require.Equal(t, []string{"POST " + pathValidateSpec}, rr.sequence(),
		"release flags must not produce a release, version or promotion request")

	body := rr.find(pathValidateSpec)[0].decodeBody(t)
	require.Equal(t, true, body["release"])
	require.Equal(t, true, body["releaseAsPreferred"])
	require.Equal(t, "v1.0.0-alpha", body["releaseVersionName"])
	require.Equal(t, true, body["forceCreateNewServicePlanVersion"])
}

// ---------------------------------------------------------------------------
// Server-side failure modes
// ---------------------------------------------------------------------------

// TestDryRunTargetUnsupportedServerNeverFallsBackToLegacy — 02: "unsupported
// server 404/405 ... never fall back to legacy build"; 04 §"Output and logging".
func TestDryRunTargetUnsupportedServerNeverFallsBackToLegacy(t *testing.T) {
	for _, status := range []int{http.StatusNotFound, http.StatusMethodNotAllowed} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			rr := startValidationOnlyServer(t, func(_ int, _ recordedRequest, w http.ResponseWriter) {
				writeAPIError(w, status, "not_supported", "unknown route")
			})

			opts := targetDryRunOptions(t)
			err := runBuildWithOptions(context.Background(), opts, staticToken())
			require.Error(t, err)
			require.Contains(t, strings.ToLower(err.Error()), "upgrade",
				"the error must tell the user the server needs an upgrade")

			rr.assertNoViolations(t)
			require.Equal(t, []string{"POST " + pathValidateSpec}, rr.sequence(),
				"no legacy prepare/build/upload fallback is permitted")
		})
	}
}

// TestDryRunTargetUnauthorizedIsReportedFaithfully — 02: "401/403".
func TestDryRunTargetUnauthorizedIsReportedFaithfully(t *testing.T) {
	for _, status := range []int{http.StatusUnauthorized, http.StatusForbidden} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			rr := startValidationOnlyServer(t, func(_ int, _ recordedRequest, w http.ResponseWriter) {
				writeAPIError(w, status, "unauthorized", "not allowed")
			})

			opts := targetDryRunOptions(t)
			err := runBuildWithOptions(context.Background(), opts, staticToken())
			require.Error(t, err)
			require.NotContains(t, strings.ToLower(err.Error()), "upgrade",
				"auth errors must retain their real meaning")

			rr.assertNoViolations(t)
			require.Equal(t, 1, rr.countPath(pathValidateSpec))
		})
	}
}

// TestDryRunTargetTransportTimeoutFailsWithoutFallback — 02: "transport
// timeout". Retries are disabled, so the request count is exact.
func TestDryRunTargetTransportTimeoutFailsWithoutFallback(t *testing.T) {
	t.Setenv("OMNISTRATE_CLIENT_TIMEOUT_IN_SECONDS", "1")
	rr := startValidationOnlyServer(t, func(_ int, _ recordedRequest, w http.ResponseWriter) {
		time.Sleep(3 * time.Second)
		writeJSON(w, validationResult(statusValid, nil))
	})
	// startRecordingServer sets a 30s timeout; re-apply the short one.
	t.Setenv("OMNISTRATE_CLIENT_TIMEOUT_IN_SECONDS", "1")

	opts := targetDryRunOptions(t)
	err := runBuildWithOptions(context.Background(), opts, staticToken())
	require.Error(t, err)

	rr.assertNoViolations(t)
	require.Equal(t, 1, rr.countPath(pathValidateSpec), "no retry, no legacy fallback")
}

// ---------------------------------------------------------------------------
// Local artifact failures
// ---------------------------------------------------------------------------

// TestDryRunTargetUnreadableLocalArtifactAborts — 02: "unreadable local
// artifact"; 04 step 7 ("abort with a clear error; do not silently substitute").
//
// RECONCILED: the fixture now declares the local path, so the failure is
// genuinely the unreadable directory rather than the allowlist refusing an
// undeclared path.
func TestDryRunTargetUnreadableLocalArtifactAborts(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("running as root: mode 0000 does not make a directory unreadable")
	}

	requirement := []map[string]any{{
		"logicalPath": "terraform/network",
		// RECONCILED: ValidationArtifactUse.path is a required field in the
		// generated SDK, so a use without it fails to decode.
		"uses": []map[string]any{{
			"resourceKey": "network",
			"kind":        "terraform",
			"path":        "/services/0/terraformConfigurations/configurationPerCloudProvider/aws",
		}},
	}}

	rr := startValidationOnlyServer(t, func(_ int, _ recordedRequest, w http.ResponseWriter) {
		writeJSON(w, validationResult(statusIncomplete, requirement))
	})

	dir := t.TempDir()
	specPath := writeFixture(t, dir, "spec.yaml", servicePlanSpecWithLocalTerraform)
	artifactDir := filepath.Join(dir, "terraform", "network")
	require.NoError(t, os.MkdirAll(artifactDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(artifactDir, "main.tf"), []byte("x"), 0o600))
	require.NoError(t, os.Chmod(artifactDir, 0o000))
	t.Cleanup(func() { _ = os.Chmod(artifactDir, 0o755) })
	t.Chdir(dir)

	err := runBuildWithOptions(context.Background(),
		jsonDryRunOptions(specPath, ServicePlanSpecType), staticToken())
	require.Error(t, err, "an unreadable artifact must abort the dry run")

	rr.assertNoViolations(t)
	require.Equal(t, 1, rr.countPath(pathValidateSpec),
		"the second request must not be sent with substituted content")
}

// TestDryRunTargetMalformedArchiveIsRejected — 02: "malformed archive". A
// pre-existing .tar.gz that is not a valid gzip stream must fail explicitly and
// must not be followed by any mutating request.
//
// RECONCILED twice: the fixture now declares the archive as its local Helm
// source, and the client detects the corrupt gzip stream itself, so the second
// request is never sent at all. The original expected the server to reject it.
func TestDryRunTargetMalformedArchiveIsRejected(t *testing.T) {
	requirement := []map[string]any{{
		"logicalPath": "artifacts/bundle.tar.gz",
		"uses": []map[string]any{{
			"resourceKey": "chart",
			"kind":        "helm",
			"path":        "/services/0/helmChartConfiguration",
		}},
	}}

	rr := startValidationOnlyServer(t, func(call int, _ recordedRequest, w http.ResponseWriter) {
		if call == 0 {
			writeJSON(w, validationResult(statusIncomplete, requirement))
			return
		}
		writeAPIError(w, http.StatusBadRequest, "invalid_archive", "archive is not a valid gzip stream")
	})

	dir := t.TempDir()
	specPath := writeFixture(t, dir, "spec.yaml", servicePlanSpecWithLocalArchive)
	writeFixture(t, dir, "artifacts/bundle.tar.gz", "this is not gzip content at all")
	t.Chdir(dir)

	err := runBuildWithOptions(context.Background(),
		jsonDryRunOptions(specPath, ServicePlanSpecType), staticToken())
	require.Error(t, err, "a malformed archive must fail the dry run")
	require.Contains(t, err.Error(), "not a valid gzip archive")

	rr.assertNoViolations(t)
	require.LessOrEqual(t, rr.countPath(pathValidateSpec), 2)
	require.Equal(t, 1, rr.countPath(pathValidateSpec),
		"a corrupt archive is detected locally, so the second request is never sent")
}
