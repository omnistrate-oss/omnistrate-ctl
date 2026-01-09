package claude

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
}

func enableCommand() *cobra.Command {
	flags := &enableFlags{}
	cmd := &cobra.Command{
		Use:   "enable",
		Short: "Add server to Claude config",
		Long:  `Add this application as an MCP server in Claude Desktop`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runEnable(cmd, flags)
		},
	}

	cmd.Flags().StringVar(&flags.logLevel, "log-level", "", "Log level (debug, info, warn, error)")
	cmd.Flags().StringVar(&flags.configPath, "config-path", "", "Path to Claude config file")
	cmd.Flags().StringVar(&flags.serverName, "server-name", "", "Name for the MCP server (default: derived from executable name)")

	return cmd
}

func runEnable(cmd *cobra.Command, flags *enableFlags) error {
	// Get the current executable path
	executablePath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get executable path for MCP server registration: %w", err)
	}

	// Validate the executable but PRESERVE the original path (including symlinks)
	// This is the key fix - we don't resolve symlinks in the returned path
	executablePath, err = cfgmgr.ValidateExecutable(executablePath)
	if err != nil {
		return err
	}

	// Create config manager
	manager := NewManager(flags.configPath)

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
		return fmt.Errorf("failed to check if MCP server '%s' exists in Claude configuration: %w", serverName, err)
	}
	if exists {
		fmt.Printf("MCP server '%s' is already enabled\n", serverName)
		return nil
	}

	// Build server configuration
	server := MCPServer{
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
		return fmt.Errorf("failed to add MCP server '%s' to Claude configuration: %w", serverName, err)
	}

	fmt.Printf("Successfully enabled MCP server '%s'\n", serverName)
	fmt.Printf("Executable: %s\n", executablePath)
	fmt.Printf("Args: %v\n", server.Args)
	fmt.Printf("\nTo use this server, restart Claude Desktop.\n")
	return nil
}
