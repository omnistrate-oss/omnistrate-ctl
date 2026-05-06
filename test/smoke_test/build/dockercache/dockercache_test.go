package dockercache

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/omnistrate-oss/omnistrate-ctl/cmd"
	"github.com/omnistrate-oss/omnistrate-ctl/test/testutils"
	"github.com/stretchr/testify/require"
)

// Test_build_from_repo_with_docker_cache_parsing validates that build-from-repo
// correctly parses cache_from and cache_to from a compose file. Uses
// --skip-docker-build to exercise the compose parsing path without Docker.
func Test_build_from_repo_with_docker_cache_parsing(t *testing.T) {
	testutils.SmokeTest(t)

	ctx := context.TODO()
	require := require.New(t)
	defer testutils.Cleanup()

	// Login
	testEmail, testPassword, err := testutils.GetTestAccount()
	require.NoError(err)
	cmd.RootCmd.SetArgs([]string{"login", fmt.Sprintf("--email=%s", testEmail), fmt.Sprintf("--password=%s", testPassword)})
	err = cmd.RootCmd.ExecuteContext(ctx)
	require.NoError(err)

	// Create a temp directory with a git repo, Dockerfile, and compose file with cache config
	tmpDir, err := os.MkdirTemp("", "omctl-cache-test-*")
	require.NoError(err)
	defer os.RemoveAll(tmpDir)

	origDir, err := os.Getwd()
	require.NoError(err)
	defer func() { _ = os.Chdir(origDir) }()

	initGitRepo(t, tmpDir)
	writeDockerfile(t, tmpDir)
	composeFile := writeComposeWithCache(t, tmpDir)

	require.NoError(os.Chdir(tmpDir))

	// Run build-from-repo with --skip-docker-build to validate compose parsing
	// (including cache_from/cache_to extraction) without needing Docker
	cmd.RootCmd.SetArgs([]string{
		"build-from-repo",
		"--file", composeFile,
		"--skip-docker-build",
		"--skip-service-build",
		"--skip-environment-promotion",
		"--skip-saas-portal-init",
	})
	err = cmd.RootCmd.ExecuteContext(ctx)
	require.NoError(err)
}

// Test_build_from_repo_with_docker_cache_full performs a full build-from-repo
// with --dry-run, including an actual docker buildx build using GHA cache.
// Requires Docker, buildx, GH_TOKEN, and ENABLE_DOCKER_CACHE_TEST=true.
func Test_build_from_repo_with_docker_cache_full(t *testing.T) {
	testutils.SmokeTest(t)
	if os.Getenv("ENABLE_DOCKER_CACHE_TEST") != "true" {
		t.Skip("Skipping full docker cache test (set ENABLE_DOCKER_CACHE_TEST=true to enable)")
	}

	ctx := context.TODO()
	require := require.New(t)
	defer testutils.Cleanup()

	// Verify Docker is available
	err := exec.Command("docker", "version").Run()
	require.NoError(err, "Docker is required for this test")

	// Login
	testEmail, testPassword, err := testutils.GetTestAccount()
	require.NoError(err)
	cmd.RootCmd.SetArgs([]string{"login", fmt.Sprintf("--email=%s", testEmail), fmt.Sprintf("--password=%s", testPassword)})
	err = cmd.RootCmd.ExecuteContext(ctx)
	require.NoError(err)

	// Create a temp directory with git repo, Dockerfile, and compose with cache config
	tmpDir, err := os.MkdirTemp("", "omctl-cache-full-test-*")
	require.NoError(err)
	defer os.RemoveAll(tmpDir)

	origDir, err := os.Getwd()
	require.NoError(err)
	defer func() { _ = os.Chdir(origDir) }()

	initGitRepo(t, tmpDir)
	writeDockerfile(t, tmpDir)
	composeFile := writeComposeWithCache(t, tmpDir)

	require.NoError(os.Chdir(tmpDir))

	// Run build-from-repo --dry-run: builds the Docker image locally with GHA
	// cache flags but skips pushing to GHCR and creating the service.
	// Explicitly set --platforms to avoid CleanupArgsAndFlags StringArray bug
	// where the default "[linux/amd64]" gets re-parsed with brackets.
	cmd.RootCmd.SetArgs([]string{
		"build-from-repo",
		"--file", composeFile,
		"--dry-run",
		"--platforms", "linux/amd64",
	})
	err = cmd.RootCmd.ExecuteContext(ctx)
	require.NoError(err)
}

func initGitRepo(t *testing.T, dir string) {
	t.Helper()
	for _, args := range [][]string{
		{"git", "init", dir},
		{"git", "-C", dir, "config", "user.email", "test@test.com"},
		{"git", "-C", dir, "config", "user.name", "Test"},
		{"git", "-C", dir, "remote", "add", "origin", "https://github.com/test-owner/test-repo.git"},
	} {
		out, err := exec.Command(args[0], args[1:]...).CombinedOutput() //nolint:gosec
		require.NoError(t, err, "git command %v failed: %s", args, string(out))
	}
}

func writeDockerfile(t *testing.T, dir string) {
	t.Helper()
	content := "FROM alpine:3.18\nRUN echo hello\n"
	require.NoError(t, os.WriteFile(filepath.Join(dir, "Dockerfile"), []byte(content), 0600))
}

func writeComposeWithCache(t *testing.T, dir string) string {
	t.Helper()
	composeContent := `services:
  web:
    build:
      context: .
      dockerfile: Dockerfile
      cache_from:
        - type=gha
      cache_to:
        - type=gha,mode=max
    image: alpine:3.18
    ports:
      - "8080:8080"
`
	composeFile := filepath.Join(dir, "omnistrate-compose.yaml")
	require.NoError(t, os.WriteFile(composeFile, []byte(composeContent), 0600))
	return composeFile
}
