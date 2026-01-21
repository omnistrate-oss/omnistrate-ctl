package vscode

import (
	"fmt"
	"os"

	"github.com/omnistrate-oss/omnistrate-ctl/cmd/mcp/cfgmgr"
	"github.com/spf13/cobra"
)

type disableFlags struct {
	configPath string
	serverName string
	workspace  bool
	configType string
}

func disableCommand() *cobra.Command {
	flags := &disableFlags{}
	cmd := &cobra.Command{
		Use:   "disable",
		Short: "Remove server from VSCode config",
		Long:  `Remove this application as an MCP server from VSCode`,
		RunE: func(_ *cobra.Command, _ []string) error {
			return runDisable(flags)
		},
	}

	cmd.Flags().StringVar(&flags.configPath, "config-path", "", "Path to VSCode config file")
	cmd.Flags().StringVar(&flags.serverName, "server-name", "", "Name of the MCP server to disable (default: derived from executable name)")
	cmd.Flags().BoolVar(&flags.workspace, "workspace", false, "Remove from workspace settings (.vscode/mcp.json) instead of user settings")
	cmd.Flags().StringVar(&flags.configType, "config-type", "", "Configuration type: 'workspace' or 'user' (default: user)")

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

	// Determine configuration type
	configType := UserConfig
	if flags.workspace || flags.configType == "workspace" {
		configType = WorkspaceConfig
	} else if flags.configType == "user" {
		configType = UserConfig
	} else if flags.configType != "" {
		return fmt.Errorf("invalid config type '%s': must be 'workspace' or 'user'", flags.configType)
	}

	// Create config manager
	manager := NewManager(flags.configPath, configType)

	// Check if server exists
	exists, err := manager.HasServer(serverName)
	if err != nil {
		return fmt.Errorf("failed to check if MCP server '%s' exists in VSCode configuration: %w", serverName, err)
	}
	if !exists {
		fmt.Printf("MCP server '%s' is not enabled in VSCode\n", serverName)
		return nil
	}

	// Backup config before modifying
	if err := manager.BackupConfig(); err != nil {
		fmt.Printf("Warning: failed to create backup: %v\n", err)
	}

	if err := manager.RemoveServer(serverName); err != nil {
		return fmt.Errorf("failed to remove MCP server '%s' from VSCode configuration: %w", serverName, err)
	}

	fmt.Printf("Successfully disabled MCP server '%s' in VSCode\n", serverName)
	fmt.Printf("\nTo complete removal, restart VSCode.\n")
	return nil
}
