package utils

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"strings"
	"time"

	"github.com/omnistrate-oss/omnistrate-ctl/internal/config"
)

const crashLogFileName = "crash.log"

// CrashLogPath returns the path to the crash log file.
func CrashLogPath() string {
	return filepath.Join(config.ConfigDir(), "log", crashLogFileName)
}

// HandlePanic should be deferred at the top of the program entry point.
// It recovers from panics, writes a crash log, restores the terminal, and exits.
func HandlePanic() {
	r := recover()
	if r == nil {
		return
	}

	// Restore terminal state: show cursor and exit alt-screen
	fmt.Fprint(os.Stderr, "\033[?25h")   // show cursor
	fmt.Fprint(os.Stderr, "\033[?1049l") // exit alt-screen
	os.Stderr.Sync()

	stack := string(debug.Stack())
	crashLog := buildCrashLog(r, stack)
	logPath := CrashLogPath()

	// Ensure the config directory exists
	dir := filepath.Dir(logPath)
	_ = os.MkdirAll(dir, 0700)

	err := os.WriteFile(logPath, []byte(crashLog), 0600)
	if err != nil {
		// If we can't write the crash log, print to stderr as fallback
		fmt.Fprintf(os.Stderr, "\n%s\n", crashLog)
	} else {
		fmt.Fprintf(os.Stderr, "\nomnistrate-ctl crashed unexpectedly.\nCrash log saved to: %s\n", logPath)
	}

	os.Exit(2)
}

func buildCrashLog(panicValue interface{}, stack string) string {
	var b strings.Builder

	b.WriteString("=== omnistrate-ctl crash report ===\n")
	fmt.Fprintf(&b, "Time:    %s\n", time.Now().UTC().Format(time.RFC3339))
	fmt.Fprintf(&b, "Version: %s\n", config.Version)
	fmt.Fprintf(&b, "Commit:  %s\n", config.CommitID)
	fmt.Fprintf(&b, "OS/Arch: %s/%s\n", runtime.GOOS, runtime.GOARCH)
	fmt.Fprintf(&b, "Go:      %s\n", runtime.Version())
	fmt.Fprintf(&b, "\nPanic: %v\n", panicValue)
	fmt.Fprintf(&b, "\nStack trace:\n%s\n", stack)

	return b.String()
}
