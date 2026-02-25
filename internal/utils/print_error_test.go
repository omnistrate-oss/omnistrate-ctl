package utils

import (
	"errors"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPrintErrorWritesToStderr(t *testing.T) {
	// Set OMNISTRATE_DRY_RUN so PrintError doesn't call os.Exit
	t.Setenv("OMNISTRATE_DRY_RUN", "true")

	// Capture stderr
	origStderr := os.Stderr
	r, w, err := os.Pipe()
	assert.NoError(t, err)
	os.Stderr = w

	// Capture stdout
	origStdout := os.Stdout
	rOut, wOut, err := os.Pipe()
	assert.NoError(t, err)
	os.Stdout = wOut

	// Call PrintError
	PrintError(errors.New("test error message"))

	// Restore and read
	w.Close()
	wOut.Close()
	os.Stderr = origStderr
	os.Stdout = origStdout

	stderrBytes, _ := io.ReadAll(r)
	stderrOutput := string(stderrBytes)

	stdoutBytes, _ := io.ReadAll(rOut)
	stdoutOutput := string(stdoutBytes)

	// Error should appear on stderr, not stdout
	assert.True(t, strings.Contains(stderrOutput, "test error message"),
		"expected error on stderr, got: %s", stderrOutput)
	assert.Empty(t, stdoutOutput,
		"expected nothing on stdout, got: %s", stdoutOutput)
}
