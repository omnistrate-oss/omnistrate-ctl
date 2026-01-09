package vscode

import (
	"fmt"
	"os"

	"github.com/omnistrate-oss/omnistrate-ctl/cmd/mcp/cfgmgr"
	"github.com/spf13/cobra"
)

type listFlags struct {
	configPath string
	workspace  bool
	configType string
}

func listCommand() *cobra.Command {
	flags := &listFlags{}
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List configured MCP servers",
		Long:  `List all MCP servers configured in VSCode`,
		RunE: func(_ *cobra.Command, _ []string) error {
			return runList(flags)
		},
	}

	cmd.Flags().StringVar(&flags.configPath, "config-path", "", "Path to VSCode config file")
	cmd.Flags().BoolVar(&flags.workspace, "workspace", false, "List workspace settings (.vscode/mcp.json) instead of user settings")
	cmd.Flags().StringVar(&flags.configType, "config-type", "", "Configuration type: 'workspace' or 'user' (default: user)")

	return cmd
}

func runList(flags *listFlags) error {
	// Determine configuration type
	configType := UserConfig
	if flags.workspace || flags.configType == "workspace" {
		configType = WorkspaceConfig
	} else if flags.configType == "user" {
		configType = UserConfig
	} else if flags.configType != "" {
		return fmt.Errorf("invalid config type '%s': must be 'workspace' or 'user'", flags.configType)
	}

	manager := NewManager(flags.configPath, configType)

	servers, err := manager.ListServers()
	if err != nil {
		return fmt.Errorf("failed to list MCP servers: %w", err)
	}

	if len(servers) == 0 {
		fmt.Println("No MCP servers configured")
		return nil
	}

	// Get current executable name for highlighting
	currentName := ""
	if execPath, err := os.Executable(); err == nil {
		currentName = cfgmgr.DeriveServerName(execPath)
	}

	fmt.Printf("Configured MCP servers in %s:\n\n", manager.GetConfigPath())
	for name, server := range servers {
		marker := ""
		if name == currentName {
			marker = " (this application)"
		}
		fmt.Printf("  %s%s\n", name, marker)
		if server.Type != "" {
			fmt.Printf("    Type: %s\n", server.Type)
		}
		fmt.Printf("    Command: %s\n", server.Command)
		if len(server.Args) > 0 {
			fmt.Printf("    Args: %v\n", server.Args)
		}
		if len(server.Env) > 0 {
			fmt.Printf("    Env: %v\n", server.Env)
		}
		fmt.Println()
	}

	return nil
}
