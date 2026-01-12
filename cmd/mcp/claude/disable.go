package claude

import (
	"fmt"
	"os"

	"github.com/omnistrate-oss/omnistrate-ctl/cmd/mcp/cfgmgr"
	"github.com/spf13/cobra"
)

type disableFlags struct {
	configPath string
	serverName string
}

func disableCommand() *cobra.Command {
	flags := &disableFlags{}
	cmd := &cobra.Command{
		Use:   "disable",
		Short: "Remove server from Claude config",
		Long:  `Remove this application as an MCP server from Claude Desktop`,
		RunE: func(_ *cobra.Command, _ []string) error {
			return runDisable(flags)
		},
	}

	cmd.Flags().StringVar(&flags.configPath, "config-path", "", "Path to Claude config file")
	cmd.Flags().StringVar(&flags.serverName, "server-name", "", "Name of the MCP server to disable (default: derived from executable name)")

	return cmd
}

func runDisable(flags *disableFlags) error {
	// Determine server name
	serverName := flags.serverName
	if serverName == "" {
		executablePath, err := os.Executable()
		if err != nil {
			return fmt.Errorf("failed to get executable path for determining server name: %w", err)
		}
		serverName = cfgmgr.DeriveServerName(executablePath)
		if serverName == "" {
			return fmt.Errorf("MCP server name cannot be empty: unable to derive name from executable path '%s'", executablePath)
		}
	}

	// Create config manager
	manager := NewManager(flags.configPath)

	// Check if server exists
	exists, err := manager.HasServer(serverName)
	if err != nil {
		return fmt.Errorf("failed to check if MCP server '%s' exists in Claude configuration: %w", serverName, err)
	}
	if !exists {
		fmt.Printf("MCP server '%s' is not enabled\n", serverName)
		return nil
	}

	// Backup config before modifying
	if err := manager.BackupConfig(); err != nil {
		fmt.Printf("Warning: failed to create backup: %v\n", err)
	}

	if err := manager.RemoveServer(serverName); err != nil {
		return fmt.Errorf("failed to remove MCP server '%s' from Claude configuration: %w", serverName, err)
	}

	fmt.Printf("Successfully disabled MCP server '%s'\n", serverName)
	fmt.Printf("\nTo complete removal, restart Claude Desktop.\n")
	return nil
}
