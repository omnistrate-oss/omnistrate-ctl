package utils

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/chelnak/ysmrr"
)

// EnsureCursorRestoration forces cursor restoration - should be called in cleanup
func EnsureCursorRestoration() {
	fmt.Print("\033[?25h") // Show cursor
	os.Stdout.Sync()
}

// StartSpinnerWithCleanup starts a spinner and sets up cleanup handlers for all exit paths
func StartSpinnerWithCleanup(sm ysmrr.SpinnerManager) {
	// Set up signal handler to restore cursor on interrupt (Ctrl+C)
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigChan
		sm.Stop()
		EnsureCursorRestoration()
		os.Exit(130) // Standard exit code for SIGINT
	}()

	sm.Start()
}

func HandleSpinnerError(spinner *ysmrr.Spinner, sm ysmrr.SpinnerManager, err error) {
	if spinner != nil {
		spinner.Error()
	}
	if sm != nil {
		sm.Stop()
	}
	EnsureCursorRestoration()
	PrintError(err)
}

func HandleSpinnerSuccess(spinner *ysmrr.Spinner, sm ysmrr.SpinnerManager, message string) {
	if spinner != nil {
		spinner.UpdateMessage(message)
		spinner.Complete()
	}
	if sm != nil {
		sm.Stop()
	}
	EnsureCursorRestoration()
}
