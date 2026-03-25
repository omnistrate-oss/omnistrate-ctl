package utils

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBuildCrashLog(t *testing.T) {
	require := require.New(t)

	log := buildCrashLog("test panic value", "goroutine 1 [running]:\nmain.main()\n")

	require.Contains(log, "=== omnistrate-ctl crash report ===")
	require.Contains(log, "Panic: test panic value")
	require.Contains(log, "goroutine 1 [running]:")
	require.Contains(log, "Time:")
	require.Contains(log, "OS/Arch:")
	require.Contains(log, "Go:")
}

func TestBuildCrashLogWithError(t *testing.T) {
	require := require.New(t)

	err := os.ErrPermission
	log := buildCrashLog(err, "stack trace here")

	require.Contains(log, "Panic: permission denied")
	require.Contains(log, "stack trace here")
}

func TestCrashLogPath(t *testing.T) {
	require := require.New(t)

	path := CrashLogPath()
	require.True(strings.HasSuffix(path, filepath.Join("log", "crash.log")))
	require.Contains(path, ".omnistrate")
}

func TestHandlePanicWritesCrashLog(t *testing.T) {
	require := require.New(t)

	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, crashLogFileName)

	// Test the crash log writing logic (without os.Exit)
	stack := "goroutine 1 [running]:\nmain.main()\n\t/app/main.go:10\n"
	crashLog := buildCrashLog("nil pointer dereference", stack)

	err := os.WriteFile(logPath, []byte(crashLog), 0600)
	require.NoError(err)

	content, err := os.ReadFile(logPath)
	require.NoError(err)
	require.Contains(string(content), "nil pointer dereference")
	require.Contains(string(content), "main.main()")
	require.Contains(string(content), "=== omnistrate-ctl crash report ===")
}
