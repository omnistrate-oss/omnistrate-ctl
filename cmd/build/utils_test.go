package build

import (
	"encoding/base64"
	"os"
	"path/filepath"
	"testing"

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
	assert.Contains(t, err.Error(), "escapes base directory")
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
