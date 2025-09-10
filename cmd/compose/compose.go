package compose

import (
	"github.com/spf13/cobra"
)

// Cmd represents the compose command
var Cmd = &cobra.Command{
	Use:   "compose",
	Short: "Generate and manage Docker Compose specifications",
	Long: `Generate and manage Docker Compose specifications with Omnistrate extensions.
This command provides subcommands to generate new compose files and update existing ones.`,
}
