package utils

import (
	"errors"
	"io"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPrintErrorWritesToStderr(t *testing.T) {
	require := require.New(t)

	// Set OMNISTRATE_DRY_RUN so PrintError doesn't call os.Exit
	t.Setenv("OMNISTRATE_DRY_RUN", "true")

	// Capture stderr
	origStderr := os.Stderr
	r, w, err := os.Pipe()
	require.NoError(err)
	os.Stderr = w

	// Capture stdout
	origStdout := os.Stdout
	rOut, wOut, err := os.Pipe()
	require.NoError(err)
	os.Stdout = wOut

	// Call PrintError
	PrintError(errors.New("test error message"))

	// Restore and read
	w.Close()
	wOut.Close()
	os.Stderr = origStderr
	os.Stdout = origStdout

	stderrBytes, err := io.ReadAll(r)
	require.NoError(err)
	stderrOutput := string(stderrBytes)

	stdoutBytes, err := io.ReadAll(rOut)
	require.NoError(err)
	stdoutOutput := string(stdoutBytes)

	// Error should appear on stderr, not stdout
	require.Contains(stderrOutput, "test error message")
	require.Empty(stdoutOutput, "expected nothing on stdout, got: %s", stdoutOutput)
}
