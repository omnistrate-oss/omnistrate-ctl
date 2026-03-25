package claude

import (
	"github.com/spf13/cobra"
)

// Command creates the claude subcommand for MCP configuration
func Command() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "claude",
		Short: "Configure Claude Desktop MCP servers",
		Long:  `Configure MCP servers for Claude Desktop`,
	}

	cmd.AddCommand(enableCommand(), disableCommand(), listCommand())
	return cmd
}
