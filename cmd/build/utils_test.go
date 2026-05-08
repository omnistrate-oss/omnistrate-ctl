package build

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/compose-spec/compose-go/loader"
	"github.com/compose-spec/compose-go/types"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDetectSpecType_TerraformSpec(t *testing.T) {
	yamlContent := map[string]interface{}{
		"name": "Terraform",
		"services": []interface{}{
			map[string]interface{}{
				"name": "terraformResource",
				"type": "terraform",
				"terraformConfigurations": map[string]interface{}{
					"configurationPerCloudProvider": map[string]interface{}{
						"aws": map[string]interface{}{
							"terraformPath": "/",
						},
					},
				},
			},
		},
	}

	specType := DetectSpecType(yamlContent)
	assert.Equal(t, ServicePlanSpecType, specType)
}

func TestDetectSpecType_HelmSpec(t *testing.T) {
	yamlContent := map[string]interface{}{
		"name": "Redis",
		"services": []interface{}{
			map[string]interface{}{
				"name": "redis",
				"helmChartConfiguration": map[string]interface{}{
					"chartName": "redis",
				},
			},
		},
	}

	specType := DetectSpecType(yamlContent)
	assert.Equal(t, ServicePlanSpecType, specType)
}

func TestDetectSpecType_DockerCompose(t *testing.T) {
	yamlContent := map[string]interface{}{
		"services": map[string]interface{}{
			"web": map[string]interface{}{
				"image": "nginx",
			},
		},
	}

	specType := DetectSpecType(yamlContent)
	assert.Equal(t, DockerComposeSpecType, specType)
}

func TestContainsOmnistrateKey(t *testing.T) {
	tests := []struct {
		name     string
		content  map[string]interface{}
		expected bool
	}{
		{
			name: "has x-omnistrate-service-plan",
			content: map[string]interface{}{
				"x-omnistrate-service-plan": map[string]interface{}{
					"name": "test",
				},
			},
			expected: true,
		},
		{
			name: "has x-omnistrate-hosted",
			content: map[string]interface{}{
				"x-omnistrate-hosted": map[string]interface{}{
					"AwsAccountId": "123",
				},
			},
			expected: true,
		},
		{
			name: "no omnistrate keys",
			content: map[string]interface{}{
				"services": map[string]interface{}{
					"web": map[string]interface{}{
						"image": "nginx",
					},
				},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, ContainsOmnistrateKey(tt.content))
		})
	}
}

func TestUniqueArtifactPathsFromTasks_Empty(t *testing.T) {
	paths := UniqueArtifactPathsFromTasks(nil)
	assert.Nil(t, paths)

	paths = UniqueArtifactPathsFromTasks([]ArtifactUploadingTask{})
	assert.Nil(t, paths)
}

func TestUniqueArtifactPathsFromTasks_Deduplication(t *testing.T) {
	tasks := []ArtifactUploadingTask{
		{ArtifactPath: "/path/a", AccountConfigID: "acc-1"},
		{ArtifactPath: "/path/b", AccountConfigID: "acc-2"},
		{ArtifactPath: "/path/a", AccountConfigID: "acc-3"},
	}
	paths := UniqueArtifactPathsFromTasks(tasks)
	assert.Len(t, paths, 2)
	assert.Contains(t, paths, "/path/a")
	assert.Contains(t, paths, "/path/b")
}

func TestUniqueArtifactPathsFromTasks_SinglePath(t *testing.T) {
	tasks := []ArtifactUploadingTask{
		{ArtifactPath: "/only/path", AccountConfigID: "acc-1", ServiceName: "svc", ProductTierName: "pt", EnvironmentType: "DEV"},
	}
	paths := UniqueArtifactPathsFromTasks(tasks)
	assert.Len(t, paths, 1)
	assert.Equal(t, "/only/path", paths[0])
}

func TestArchiveArtifactPaths_CreatesBase64Archive(t *testing.T) {
	// Create a temporary source directory with some files
	sourceDir, err := os.MkdirTemp("", "test-source-*")
	require.NoError(t, err)
	defer os.RemoveAll(sourceDir)

	// Create a subdirectory to archive
	artifactDir := filepath.Join(sourceDir, "artifacts")
	err = os.MkdirAll(artifactDir, 0755)
	require.NoError(t, err)

	// Create some test files
	err = os.WriteFile(filepath.Join(artifactDir, "test.txt"), []byte("test content"), 0600)
	require.NoError(t, err)

	// Archive the directory
	result, err := ArchiveArtifactPaths(sourceDir, []string{"artifacts"})
	require.NoError(t, err)

	assert.Len(t, result, 1)
	assert.Contains(t, result, "artifacts")

	// Verify the base64 content is valid
	base64Content := result["artifacts"]
	assert.NotEmpty(t, base64Content)

	// Verify it can be decoded
	decoded, err := base64.StdEncoding.DecodeString(base64Content)
	require.NoError(t, err)
	assert.NotEmpty(t, decoded)
}

func TestArchiveArtifactPaths_MultipleDirectories(t *testing.T) {
	// Create a temporary source directory
	sourceDir, err := os.MkdirTemp("", "test-source-*")
	require.NoError(t, err)
	defer os.RemoveAll(sourceDir)

	// Create subdirectories
	err = os.MkdirAll(filepath.Join(sourceDir, "dir1"), 0755)
	require.NoError(t, err)
	err = os.WriteFile(filepath.Join(sourceDir, "dir1", "file1.txt"), []byte("content1"), 0600)
	require.NoError(t, err)

	err = os.MkdirAll(filepath.Join(sourceDir, "dir2"), 0755)
	require.NoError(t, err)
	err = os.WriteFile(filepath.Join(sourceDir, "dir2", "file2.txt"), []byte("content2"), 0600)
	require.NoError(t, err)

	// Archive multiple directories
	result, err := ArchiveArtifactPaths(sourceDir, []string{"dir1", "dir2"})
	require.NoError(t, err)

	assert.Len(t, result, 2)
	assert.Contains(t, result, "dir1")
	assert.Contains(t, result, "dir2")
}

func TestArchiveArtifactPaths_EmptyPaths(t *testing.T) {
	result, err := ArchiveArtifactPaths("/tmp", []string{})
	require.NoError(t, err)
	assert.Nil(t, result)
}

func TestArchiveArtifactPaths_NilPaths(t *testing.T) {
	result, err := ArchiveArtifactPaths("/tmp", nil)
	require.NoError(t, err)
	assert.Nil(t, result)
}

func TestArchiveArtifactPaths_NonExistentPath(t *testing.T) {
	_, err := ArchiveArtifactPaths("/tmp", []string{"/non/existent/path/that/does/not/exist"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "does not exist")
}

func TestArchiveArtifactPaths_RejectsPathsOutsideBaseDir(t *testing.T) {
	rootDir, err := os.MkdirTemp("", "test-archive-root-*")
	require.NoError(t, err)
	defer os.RemoveAll(rootDir)

	sourceDir := filepath.Join(rootDir, "source")
	outsideDir := filepath.Join(rootDir, "outside")
	require.NoError(t, os.MkdirAll(sourceDir, 0755))
	require.NoError(t, os.MkdirAll(outsideDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(outsideDir, "secret.txt"), []byte("secret"), 0600))

	tests := []struct {
		name string
		path string
	}{
		{
			name: "relative parent traversal",
			path: filepath.Join("..", "outside"),
		},
		{
			name: "absolute path outside base dir",
			path: outsideDir,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ArchiveArtifactPaths(sourceDir, []string{tt.path})
			require.Error(t, err)
			assert.Contains(t, err.Error(), "escapes base directory")
		})
	}
}

func TestArchiveArtifactPaths_FileNotDirectory(t *testing.T) {
	// Create a temporary file (not a directory and not tar.gz)
	tmpFile, err := os.CreateTemp("", "test-file-*.txt")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	_, err = ArchiveArtifactPaths(filepath.Dir(tmpFile.Name()), []string{filepath.Base(tmpFile.Name())})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "is not a directory or a .tar.gz file")
}

func TestArchiveArtifactPaths_TarGzFilePassthrough(t *testing.T) {
	// Create a temporary directory with a real tar.gz file inside
	sourceDir, err := os.MkdirTemp("", "test-source-*")
	require.NoError(t, err)
	defer os.RemoveAll(sourceDir)

	// Create a subdirectory, archive it, then use the resulting tar.gz as input
	artifactDir := filepath.Join(sourceDir, "myartifacts")
	err = os.MkdirAll(artifactDir, 0755)
	require.NoError(t, err)
	err = os.WriteFile(filepath.Join(artifactDir, "hello.txt"), []byte("hello world"), 0600)
	require.NoError(t, err)

	// First, archive the directory to get valid tar.gz content
	base64Content, err := createTarGzBase64(artifactDir)
	require.NoError(t, err)
	rawTarGz, err := base64.StdEncoding.DecodeString(base64Content)
	require.NoError(t, err)

	// Write the tar.gz content to a .tar.gz file
	tarGzPath := filepath.Join(sourceDir, "myartifacts.tar.gz")
	err = os.WriteFile(tarGzPath, rawTarGz, 0600)
	require.NoError(t, err)

	// Archive should detect the file is already tar.gz and just base64 encode it
	result, err := ArchiveArtifactPaths(sourceDir, []string{"myartifacts.tar.gz"})
	require.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Contains(t, result, "myartifacts.tar.gz")

	// The base64 output should decode to exactly the same bytes we wrote
	decoded, err := base64.StdEncoding.DecodeString(result["myartifacts.tar.gz"])
	require.NoError(t, err)
	assert.Equal(t, rawTarGz, decoded)
}

func TestArchiveArtifactPaths_TgzFilePassthrough(t *testing.T) {
	// Create a temporary directory with a .tgz file inside
	sourceDir, err := os.MkdirTemp("", "test-source-*")
	require.NoError(t, err)
	defer os.RemoveAll(sourceDir)

	artifactDir := filepath.Join(sourceDir, "myartifacts")
	err = os.MkdirAll(artifactDir, 0755)
	require.NoError(t, err)
	err = os.WriteFile(filepath.Join(artifactDir, "data.txt"), []byte("some data"), 0600)
	require.NoError(t, err)

	base64Content, err := createTarGzBase64(artifactDir)
	require.NoError(t, err)
	rawTarGz, err := base64.StdEncoding.DecodeString(base64Content)
	require.NoError(t, err)

	tgzPath := filepath.Join(sourceDir, "myartifacts.tgz")
	err = os.WriteFile(tgzPath, rawTarGz, 0600)
	require.NoError(t, err)

	result, err := ArchiveArtifactPaths(sourceDir, []string{"myartifacts.tgz"})
	require.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Contains(t, result, "myartifacts.tgz")

	decoded, err := base64.StdEncoding.DecodeString(result["myartifacts.tgz"])
	require.NoError(t, err)
	assert.Equal(t, rawTarGz, decoded)
}

func TestIsGzipTarFile(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		content  []byte
		expected bool
	}{
		{
			name:     "tar.gz extension with gzip magic",
			filename: "archive.tar.gz",
			content:  []byte{0x1f, 0x8b, 0x08, 0x00},
			expected: true,
		},
		{
			name:     "tgz extension with gzip magic",
			filename: "archive.tgz",
			content:  []byte{0x1f, 0x8b, 0x08, 0x00},
			expected: true,
		},
		{
			name:     "tar.gz extension without gzip magic",
			filename: "fake.tar.gz",
			content:  []byte("not gzip"),
			expected: true, // extension match is enough
		},
		{
			name:     "no extension but has gzip magic bytes",
			filename: "archive.bin",
			content:  []byte{0x1f, 0x8b, 0x08, 0x00},
			expected: true, // magic bytes match is enough
		},
		{
			name:     "plain text file",
			filename: "readme.txt",
			content:  []byte("hello world"),
			expected: false,
		},
		{
			name:     "empty file with tar.gz extension",
			filename: "empty.tar.gz",
			content:  []byte{},
			expected: true, // extension match is enough
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir, err := os.MkdirTemp("", "test-isgzip-*")
			require.NoError(t, err)
			defer os.RemoveAll(tmpDir)

			filePath := filepath.Join(tmpDir, tt.filename)
			err = os.WriteFile(filePath, tt.content, 0600)
			require.NoError(t, err)

			assert.Equal(t, tt.expected, isGzipTarFile(filePath))
		})
	}
}

func TestArchiveArtifactPaths_NestedDirectories(t *testing.T) {
	// Create a temporary source directory with nested structure
	sourceDir, err := os.MkdirTemp("", "test-source-*")
	require.NoError(t, err)
	defer os.RemoveAll(sourceDir)

	// Create a nested directory structure
	nestedDir := filepath.Join(sourceDir, "artifacts", "subdir1", "subdir2")
	err = os.MkdirAll(nestedDir, 0755)
	require.NoError(t, err)

	// Create files at different levels
	err = os.WriteFile(filepath.Join(sourceDir, "artifacts", "root.txt"), []byte("root"), 0600)
	require.NoError(t, err)
	err = os.WriteFile(filepath.Join(sourceDir, "artifacts", "subdir1", "level1.txt"), []byte("level1"), 0600)
	require.NoError(t, err)
	err = os.WriteFile(filepath.Join(nestedDir, "level2.txt"), []byte("level2"), 0600)
	require.NoError(t, err)

	// Archive the directory
	result, err := ArchiveArtifactPaths(sourceDir, []string{"artifacts"})
	require.NoError(t, err)

	assert.Len(t, result, 1)
	assert.Contains(t, result, "artifacts")

	// Verify the content is valid base64
	decoded, err := base64.StdEncoding.DecodeString(result["artifacts"])
	require.NoError(t, err)
	assert.NotEmpty(t, decoded)
}

func TestExpandOmctlEnvVars(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		envVars       map[string]string
		expected      string
		expectError   bool
		errorContains string
	}{
		{
			name:  "expands OMCTL_ prefixed vars",
			input: `AwsAccountId: "${OMCTL_AWS_ACCOUNT_ID}"`,
			envVars: map[string]string{
				"OMCTL_AWS_ACCOUNT_ID": "123456789012",
			},
			expected: `AwsAccountId: "123456789012"`,
		},
		{
			name:     "leaves non-OMCTL vars unchanged",
			input:    `password: $var.password`,
			expected: `password: $var.password`,
		},
		{
			name:     "leaves $sys references unchanged",
			input:    `host: $sys.network.externalClusterEndpoint`,
			expected: `host: $sys.network.externalClusterEndpoint`,
		},
		{
			name:  "mixed OMCTL and non-OMCTL vars",
			input: "AwsAccountId: \"${OMCTL_AWS_ACCOUNT_ID}\"\npassword: $var.password\nGcpProjectId: \"${OMCTL_GCP_PROJECT_ID}\"",
			envVars: map[string]string{
				"OMCTL_AWS_ACCOUNT_ID": "111222333444",
				"OMCTL_GCP_PROJECT_ID": "my-project",
			},
			expected: "AwsAccountId: \"111222333444\"\npassword: $var.password\nGcpProjectId: \"my-project\"",
		},
		{
			name:          "unset OMCTL var returns error",
			input:         `AwsAccountId: "${OMCTL_AWS_ACCOUNT_ID}"`,
			expected:      `AwsAccountId: "${OMCTL_AWS_ACCOUNT_ID}"`,
			expectError:   true,
			errorContains: "OMCTL_AWS_ACCOUNT_ID",
		},
		{
			name:  "expands in ARN patterns",
			input: `AwsBootstrapRoleAccountArn: "arn:aws:iam::${OMCTL_AWS_ACCOUNT_ID}:role/omnistrate-bootstrap-role"`,
			envVars: map[string]string{
				"OMCTL_AWS_ACCOUNT_ID": "339713121445",
			},
			expected: `AwsBootstrapRoleAccountArn: "arn:aws:iam::339713121445:role/omnistrate-bootstrap-role"`,
		},
		{
			name:          "empty env var treated as unresolved",
			input:         `AwsAccountId: "${OMCTL_AWS_ACCOUNT_ID}"`,
			envVars:       map[string]string{"OMCTL_AWS_ACCOUNT_ID": ""},
			expected:      `AwsAccountId: "${OMCTL_AWS_ACCOUNT_ID}"`,
			expectError:   true,
			errorContains: "OMCTL_AWS_ACCOUNT_ID",
		},
		{
			name:          "whitespace-only env var treated as unresolved",
			input:         `AwsAccountId: "${OMCTL_AWS_ACCOUNT_ID}"`,
			envVars:       map[string]string{"OMCTL_AWS_ACCOUNT_ID": "  "},
			expected:      `AwsAccountId: "${OMCTL_AWS_ACCOUNT_ID}"`,
			expectError:   true,
			errorContains: "OMCTL_AWS_ACCOUNT_ID",
		},
		{
			name:          "multiple unset vars lists all in error",
			input:         `AwsAccountId: "${OMCTL_AWS_ACCOUNT_ID}" GcpProjectId: "${OMCTL_GCP_PROJECT_ID}"`,
			expected:      `AwsAccountId: "${OMCTL_AWS_ACCOUNT_ID}" GcpProjectId: "${OMCTL_GCP_PROJECT_ID}"`,
			expectError:   true,
			errorContains: "OMCTL_AWS_ACCOUNT_ID, OMCTL_GCP_PROJECT_ID",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set env vars
			for k, v := range tt.envVars {
				t.Setenv(k, v)
			}

			result, err := ExpandOmctlEnvVars([]byte(tt.input))
			assert.Equal(t, tt.expected, string(result))
			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorContains)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestBuildDockerBuildArgs(t *testing.T) {
	tests := []struct {
		name      string
		platforms string
		dockerfile string
		imageURL  string
		cacheFrom []string
		cacheTo   []string
		expected  []string
	}{
		{
			name:       "no cache flags",
			platforms:  "linux/amd64",
			dockerfile: "Dockerfile",
			imageURL:   "ghcr.io/owner/repo",
			cacheFrom:  nil,
			cacheTo:    nil,
			expected:   []string{"buildx", "build", "--pull", "--platform", "linux/amd64", ".", "-f", "Dockerfile", "-t", "ghcr.io/owner/repo", "--load"},
		},
		{
			name:       "with gha cache",
			platforms:  "linux/amd64",
			dockerfile: "Dockerfile",
			imageURL:   "ghcr.io/owner/repo",
			cacheFrom:  []string{"type=gha"},
			cacheTo:    []string{"type=gha,mode=max"},
			expected:   []string{"buildx", "build", "--pull", "--platform", "linux/amd64", ".", "-f", "Dockerfile", "-t", "ghcr.io/owner/repo", "--cache-from", "type=gha", "--cache-to", "type=gha,mode=max", "--load"},
		},
		{
			name:       "multiple cache sources",
			platforms:  "linux/amd64,linux/arm64",
			dockerfile: "docker/Dockerfile.prod",
			imageURL:   "ghcr.io/owner/repo",
			cacheFrom:  []string{"type=gha", "type=registry,ref=ghcr.io/owner/repo:cache"},
			cacheTo:    []string{"type=gha,mode=max"},
			expected:   []string{"buildx", "build", "--pull", "--platform", "linux/amd64,linux/arm64", ".", "-f", "docker/Dockerfile.prod", "-t", "ghcr.io/owner/repo", "--cache-from", "type=gha", "--cache-from", "type=registry,ref=ghcr.io/owner/repo:cache", "--cache-to", "type=gha,mode=max"},
		},
		{
			name:       "cache_from only",
			platforms:  "linux/amd64",
			dockerfile: "Dockerfile",
			imageURL:   "ghcr.io/owner/repo",
			cacheFrom:  []string{"type=gha"},
			cacheTo:    nil,
			expected:   []string{"buildx", "build", "--pull", "--platform", "linux/amd64", ".", "-f", "Dockerfile", "-t", "ghcr.io/owner/repo", "--cache-from", "type=gha", "--load"},
		},
		{
			name:       "cache_to only",
			platforms:  "linux/amd64",
			dockerfile: "Dockerfile",
			imageURL:   "ghcr.io/owner/repo",
			cacheFrom:  nil,
			cacheTo:    []string{"type=gha,mode=max"},
			expected:   []string{"buildx", "build", "--pull", "--platform", "linux/amd64", ".", "-f", "Dockerfile", "-t", "ghcr.io/owner/repo", "--cache-to", "type=gha,mode=max", "--load"},
		},
		{
			name:       "multi-platform without cache skips load",
			platforms:  "linux/amd64,linux/arm64",
			dockerfile: "Dockerfile",
			imageURL:   "ghcr.io/owner/repo",
			cacheFrom:  nil,
			cacheTo:    nil,
			expected:   []string{"buildx", "build", "--pull", "--platform", "linux/amd64,linux/arm64", ".", "-f", "Dockerfile", "-t", "ghcr.io/owner/repo"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := BuildDockerBuildArgs(tt.platforms, tt.dockerfile, tt.imageURL, tt.cacheFrom, tt.cacheTo)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestComposeCacheFromCacheToParsing(t *testing.T) {
	composeYAML := `
services:
  web:
    build:
      context: .
      dockerfile: Dockerfile
      cache_from:
        - type=gha
      cache_to:
        - type=gha,mode=max
  api:
    build:
      context: ./api
      dockerfile: Dockerfile.api
      cache_from:
        - type=registry,ref=ghcr.io/owner/api:cache
  db:
    image: postgres:15
`
	parsedYaml, err := loader.ParseYAML([]byte(composeYAML))
	require.NoError(t, err)

	project, err := loader.LoadWithContext(context.Background(), types.ConfigDetails{
		ConfigFiles: []types.ConfigFile{{Config: parsedYaml}},
	})
	require.NoError(t, err)

	cacheFrom := make(map[string][]string)
	cacheTo := make(map[string][]string)

	for _, svc := range project.Services {
		if svc.Build != nil {
			if len(svc.Build.CacheFrom) > 0 {
				cacheFrom[svc.Name] = svc.Build.CacheFrom
			}
			if len(svc.Build.CacheTo) > 0 {
				cacheTo[svc.Name] = svc.Build.CacheTo
			}
		}
	}

	// web service has both cache_from and cache_to
	assert.Equal(t, []string{"type=gha"}, cacheFrom["web"])
	assert.Equal(t, []string{"type=gha,mode=max"}, cacheTo["web"])

	// api service has only cache_from
	assert.Equal(t, []string{"type=registry,ref=ghcr.io/owner/api:cache"}, cacheFrom["api"])
	assert.Empty(t, cacheTo["api"])

	// db service has no build context, no cache entries
	assert.Empty(t, cacheFrom["db"])
	assert.Empty(t, cacheTo["db"])
}

// TestReplaceBuildContextIntegration exercises the full compose-parse →
// ReplaceBuildContext pipeline, verifying that rendered specs contain no
// residual build: blocks that would cause backend "build context is not
// supported yet" errors.
func TestReplaceBuildContextIntegration(t *testing.T) {
	tests := []struct {
		name        string
		composeYAML string
		imageURLs   map[string]string // service name -> image URL
	}{
		{
			name: "cache_from and cache_to after context/dockerfile",
			composeYAML: `services:
  web:
    build:
      context: .
      dockerfile: Dockerfile
      cache_from:
        - type=gha
      cache_to:
        - type=gha,mode=max
    ports:
      - "8080:8080"
`,
			imageURLs: map[string]string{
				"web": "ghcr.io/owner/web:sha-abc123",
			},
		},
		{
			name: "cache_from and cache_to before context/dockerfile",
			composeYAML: `services:
  web:
    build:
      cache_from:
        - type=gha
      cache_to:
        - type=gha,mode=max
      context: .
      dockerfile: Dockerfile
    ports:
      - "8080:8080"
`,
			imageURLs: map[string]string{
				"web": "ghcr.io/owner/web:sha-abc123",
			},
		},
		{
			name: "cache interleaved with context and dockerfile",
			composeYAML: `services:
  api:
    build:
      context: ./api
      cache_from:
        - type=gha
        - type=registry,ref=ghcr.io/owner/api:cache
      dockerfile: Dockerfile
      cache_to:
        - type=gha,mode=max
    environment:
      DB_HOST: localhost
`,
			imageURLs: map[string]string{
				"api": "ghcr.io/owner/api:sha-def456",
			},
		},
		{
			name: "multiple services with mixed cache positions",
			composeYAML: `services:
  frontend:
    build:
      cache_from:
        - type=gha
      context: ./frontend
      dockerfile: Dockerfile
      cache_to:
        - type=gha,mode=max
    ports:
      - "3000:3000"
  backend:
    build:
      context: ./backend
      dockerfile: Dockerfile.prod
      cache_from:
        - type=gha
    environment:
      REDIS_URL: redis://redis:6379
  redis:
    image: redis:7-alpine
    ports:
      - "6379:6379"
`,
			imageURLs: map[string]string{
				"frontend": "ghcr.io/owner/frontend:sha-111",
				"backend":  "ghcr.io/owner/backend:sha-222",
			},
		},
		{
			name: "cache_from only no cache_to",
			composeYAML: `services:
  web:
    build:
      context: .
      dockerfile: Dockerfile
      cache_from:
        - type=gha
    ports:
      - "8080:8080"
`,
			imageURLs: map[string]string{
				"web": "ghcr.io/owner/web:sha-abc123",
			},
		},
		{
			name: "build with args alongside cache fields",
			composeYAML: `services:
  web:
    build:
      context: .
      dockerfile: Dockerfile
      args:
        NODE_ENV: production
        VERSION: "2.0"
      cache_from:
        - type=gha
      cache_to:
        - type=gha,mode=max
    ports:
      - "8080:8080"
`,
			imageURLs: map[string]string{
				"web": "ghcr.io/owner/web:sha-abc123",
			},
		},
		{
			name: "real-world ray-cluster layout",
			composeYAML: `version: '3'

x-omnistrate-integrations:
  - omnistrateLogging:
  - omnistrateMetrics:

x-omnistrate-service-plan:
  name: 'ray-jobs'
  tenancyType: 'OMNISTRATE_MULTI_TENANCY'

services:
  hello-world:
    build:
      cache_from:
        - type=gha
      cache_to:
        - type=gha,mode=max
      context: ./jobs/hello-world
      dockerfile: Dockerfile
    environment:
      RAY_ADDRESS: "ray://cluster:10001"
      SCRIPT_PATH: "submit_job.py"
    deploy:
      resources:
        reservations:
          cpus: "0.1"
          memory: 256M
        limits:
          cpus: "0.5"
          memory: 1G
    privileged: true
    platform: linux/amd64
  hello-cuda:
    build:
      context: ./jobs/hello-cuda
      dockerfile: Dockerfile
    environment:
      RAY_ADDRESS: "ray://cluster:10001"
      SCRIPT_PATH: "submit_job.py"
x-omnistrate-image-registry-attributes:
  ghcr.io:
    auth:
      password: ${{ secrets.GitHubPAT }}
      username: testuser
`,
			imageURLs: map[string]string{
				"hello-world": "ghcr.io/org/ray-cluster-hello-world:sha-abc",
				"hello-cuda":  "ghcr.io/org/ray-cluster-hello-cuda:sha-def",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Step 1: Parse the compose YAML exactly as build-from-repo does
			parsedYaml, err := loader.ParseYAML([]byte(tt.composeYAML))
			require.NoError(t, err, "failed to parse compose YAML")

			project, err := loader.LoadWithContext(context.Background(), types.ConfigDetails{
				ConfigFiles: []types.ConfigFile{{Config: parsedYaml}},
			})
			require.NoError(t, err, "failed to load compose project")

			// Step 2: Extract dockerfile paths and build image URL mapping
			// (mirrors build_from_repo.go logic)
			dockerfilePaths := make(map[string]string)
			for _, svc := range project.Services {
				if svc.Build != nil {
					absCtx, err := filepath.Abs(svc.Build.Context)
					require.NoError(t, err)
					dockerfilePaths[svc.Name] = filepath.Join(absCtx, svc.Build.Dockerfile)
				}
			}

			dockerPathsToImageUrls := make(map[string]string)
			for svcName, imageURL := range tt.imageURLs {
				if dp, ok := dockerfilePaths[svcName]; ok {
					dockerPathsToImageUrls[dp] = imageURL
				}
			}

			// Step 3: Apply ReplaceBuildContext
			rendered := utils.ReplaceBuildContext(tt.composeYAML, dockerPathsToImageUrls)

			// Step 4: Verify no build: blocks with context/dockerfile remain
			// This catches the exact "build context is not supported yet" error
			assert.NotContains(t, rendered, "context: ./",
				"rendered spec still contains build context — would cause backend rejection")
			assert.NotContains(t, rendered, "dockerfile: Dockerfile",
				"rendered spec still contains dockerfile reference — would cause backend rejection")

			// Step 5: Verify no orphaned cache fields remain
			assert.NotContains(t, rendered, "cache_from:",
				"rendered spec still contains orphaned cache_from")
			assert.NotContains(t, rendered, "cache_to:",
				"rendered spec still contains orphaned cache_to")

			// Step 6: Verify all expected image URLs are present
			for _, imageURL := range tt.imageURLs {
				assert.Contains(t, rendered, fmt.Sprintf(`image: "%s"`, imageURL),
					"expected image URL not found in rendered spec")
			}

			// Step 7: Verify non-build content is preserved
			if strings.Contains(tt.composeYAML, "ports:") {
				assert.Contains(t, rendered, "ports:",
					"non-build content (ports) was incorrectly removed")
			}
			if strings.Contains(tt.composeYAML, "environment:") {
				assert.Contains(t, rendered, "environment:",
					"non-build content (environment) was incorrectly removed")
			}
			if strings.Contains(tt.composeYAML, "x-omnistrate") {
				assert.Contains(t, rendered, "x-omnistrate",
					"omnistrate extensions were incorrectly removed")
			}

			// Step 8: Verify the rendered YAML is parseable
			_, err = loader.ParseYAML([]byte(rendered))
			assert.NoError(t, err, "rendered spec is not valid YAML")
		})
	}
}
