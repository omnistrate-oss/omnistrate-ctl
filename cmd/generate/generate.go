package generate

import (
	"github.com/spf13/cobra"
)

// Cmd represents the generate command
var Cmd = &cobra.Command{
	Use:   "generate",
	Short: "Generate various resources and specifications",
	Long: `Generate various resources and specifications for Omnistrate services.
This command provides subcommands to generate different types of configurations and specs.`,
}
