package utils

import (
	"fmt"
	"os"

	"github.com/chelnak/ysmrr"
)

func HandleSpinnerError(spinner *ysmrr.Spinner, sm ysmrr.SpinnerManager, err error) {
	if spinner != nil {
		spinner.Error()
	}
	if sm != nil {
		sm.Stop()
	}
	// Ensure cursor is restored
	fmt.Print("\033[?25h")
	os.Stdout.Sync()
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
	// Ensure cursor is restored
	fmt.Print("\033[?25h")
	os.Stdout.Sync()
}
