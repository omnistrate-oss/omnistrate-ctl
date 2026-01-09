package vscode

import (
	"fmt"
	"os"

	"github.com/omnistrate-oss/omnistrate-ctl/cmd/mcp/cfgmgr"
	"github.com/spf13/cobra"
)

type enableFlags struct {
	configPath string
	logLevel   string
	serverName string
	workspace  bool
	configType string
}

func enableCommand() *cobra.Command {
	flags := &enableFlags{}
	cmd := &cobra.Command{
		Use:   "enable",
		Short: "Add server to VSCode config",
		Long:  `Add this application as an MCP server in VSCode`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runEnable(cmd, flags)
		},
	}

	cmd.Flags().StringVar(&flags.logLevel, "log-level", "", "Log level (debug, info, warn, error)")
	cmd.Flags().StringVar(&flags.configPath, "config-path", "", "Path to VSCode config file")
	cmd.Flags().StringVar(&flags.serverName, "server-name", "", "Name for the MCP server (default: derived from executable name)")
	cmd.Flags().BoolVar(&flags.workspace, "workspace", false, "Add to workspace settings (.vscode/mcp.json) instead of user settings")
	cmd.Flags().StringVar(&flags.configType, "config-type", "", "Configuration type: 'workspace' or 'user' (default: user)")

	return cmd
}

func runEnable(cmd *cobra.Command, flags *enableFlags) error {
	// Get the current executable path
	executablePath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get executable path for MCP server registration: %w", err)
	}

	// Validate the executable but PRESERVE the original path (including symlinks)
	executablePath, err = cfgmgr.ValidateExecutable(executablePath)
	if err != nil {
		return err
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

	// Determine server name
	serverName := flags.serverName
	if serverName == "" {
		serverName = cfgmgr.DeriveServerName(executablePath)
		if serverName == "" {
			return fmt.Errorf("MCP server name cannot be empty: unable to derive name from executable path '%s'", executablePath)
		}
	}

	// Check if server already exists
	exists, err := manager.HasServer(serverName)
	if err != nil {
		return fmt.Errorf("failed to check if MCP server '%s' exists in VSCode configuration: %w", serverName, err)
	}
	if exists {
		fmt.Printf("MCP server '%s' is already enabled in VSCode\n", serverName)
		return nil
	}

	// Build server configuration
	server := MCPServer{
		Type:    "stdio",
		Command: executablePath,
		Args:    append(cfgmgr.GetMCPCommandPath(cmd), cfgmgr.StartCommandName),
	}

	// Add log level to args if specified
	if flags.logLevel != "" {
		server.Args = append(server.Args, "--log-level", flags.logLevel)
	}

	// Backup config before modifying
	if err := manager.BackupConfig(); err != nil {
		fmt.Printf("Warning: failed to create backup: %v\n", err)
	}

	if err := manager.AddServer(serverName, server); err != nil {
		return fmt.Errorf("failed to add MCP server '%s' to VSCode configuration: %w", serverName, err)
	}

	configTypeStr := "user"
	if configType == WorkspaceConfig {
		configTypeStr = "workspace"
	}

	fmt.Printf("Successfully enabled MCP server '%s' in VSCode (%s configuration)\n", serverName, configTypeStr)
	fmt.Printf("Executable: %s\n", executablePath)
	fmt.Printf("Args: %v\n", server.Args)
	fmt.Printf("Configuration file: %s\n", manager.GetConfigPath())
	fmt.Printf("\nTo use this server:\n")
	fmt.Printf("1. Open GitHub Copilot Chat\n")
	fmt.Printf("2. Use agent mode to access MCP tools\n")
	return nil
}
