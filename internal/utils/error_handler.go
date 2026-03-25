package utils

import (
	"fmt"
	"os"
)

// EnsureCursorRestoration forces cursor restoration - should be called in cleanup
func EnsureCursorRestoration() {
	fmt.Print("\033[?25h") // Show cursor
	os.Stdout.Sync()
}

func HandleSpinnerError(spinner *Spinner, sm SpinnerManager, err error) {
	if spinner != nil {
		spinner.Error()
	}
	if sm != nil {
		sm.Stop()
	}

	if spinner != nil || sm != nil {
		EnsureCursorRestoration()
	}

	PrintError(err)
}

func HandleSpinnerSuccess(spinner *Spinner, sm SpinnerManager, message string) {
	if spinner != nil {
		spinner.UpdateMessage(message)
		spinner.Complete()
	}
	if sm != nil {
		sm.Stop()
	}

	if spinner != nil || sm != nil {
		EnsureCursorRestoration()
	}
}
