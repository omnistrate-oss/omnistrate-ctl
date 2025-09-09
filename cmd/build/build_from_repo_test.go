package build

import (
	"os"
	"os/exec"
	"path"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/chelnak/ysmrr"
)

func TestRenderEnvFileAndInterpolateVariables(t *testing.T) {
	// Skip if Docker is not available
	if _, err := exec.LookPath("docker"); err != nil {
		t.Skip("Docker not available, skipping integration test")
	}

	cwd := "testfiles"
	sm := ysmrr.NewSpinnerManager()
	filePath := path.Join(cwd, "experio.yaml")
	fileData, err := os.ReadFile(filePath)
	require.NoError(t, err)
	expectedFilePath := path.Join(cwd, "experio.yaml.rendered")
	expectedFileData, err := os.ReadFile(expectedFilePath)
	require.NoError(t, err)

	result, err := RenderFile(fileData, cwd, filePath, sm, nil)
	require.NoError(t, err, "Error rendering env file and interpolating variables: %v", err)
	require.Equal(t, strings.ReplaceAll(string(result), " ", ""), strings.ReplaceAll(string(expectedFileData), " ", ""), "Rendered file content does not match expected content")
}
