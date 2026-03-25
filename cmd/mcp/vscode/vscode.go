package vscode

import (
	"github.com/spf13/cobra"
)

// Command creates the vscode subcommand for MCP configuration
func Command() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "vscode",
		Short: "Configure VSCode MCP servers",
		Long:  `Configure MCP servers for VSCode`,
	}

	cmd.AddCommand(enableCommand(), disableCommand(), listCommand())
	return cmd
}
