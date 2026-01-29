package mcp

import (
	"github.com/njayp/ophis"
	"github.com/spf13/cobra"

	"github.com/omnistrate-oss/omnistrate-ctl/cmd/mcp/claude"
	"github.com/omnistrate-oss/omnistrate-ctl/cmd/mcp/vscode"
)

// excludedCommands lists command paths to exclude from MCP exposure
var excludedCommands = []string{
	"mcp",
	"cost cloud-provider",
	"cost deployment-cell",
	"cost region",
	"cost user",
	"agent init",
	"audit",
	"build",
	"build-from-repo",
	"custom-network create",
	"custom-network delete",
	"custom-network update",
	"deploy",
	"domain",
	"alarms",
	"inspect",
	"helm delete",
	"helm describe",
	"helm list",
	"helm list-installations",
	"helm save",
	"deployment-cell apply-pending-changes",
	"deployment-cell debug",
	"deployment-cell describe-template",
	"deployment-cell generate-template",
	"deployment-cell update-template",
	"deployment-cell delete-nodepool",
	"deployment-cell describe-nodepool",
	"deployment-cell list-nodepools",
	"deployment-cell scale-down-nodepool",
	"deployment-cell scale-up-nodepool",
	"instance adopt",
	"instance get-deployment",
	"instance continue-deployment",
	"instance enable-debug-mode",
	"instance disable-debug-mode",
	"instance patch-deployment",
	"instance version-upgrade",
	"environment",
	"help",
	"completion",
	"operations",
	"login",
	"logout",
	"services-orchestration",
	"service-plan disable-feature",
	"service-plan enable-feature",
	"service-plan release",
	"service-plan set-default",
}

var config = &ophis.Config{
	// Shorten tool names to stay under Claude's 64-character limit
	// Changes "omnistrate-ctl_cost_by-cell_list" to "omctl_cost_by-cell_list"
	ToolNamePrefix: "omctl",
	// Command filtering using new Selectors API
	// Note: Hidden/deprecated commands are automatically excluded by ophis
	Selectors: []ophis.Selector{
		{
			CmdSelector: func(cmd *cobra.Command) bool {
				// Exclude specified commands
				if ophis.AllowCmdsContaining(excludedCommands...)(cmd) {
					return false
				}
				// Only include leaf commands (no subcommands)
				if len(cmd.Commands()) > 0 {
					return false
				}
				return true
			},
		},
	},
}

// Cmd is the MCP command with custom claude/vscode subcommands that preserve symlink paths.
// This fixes the issue where Homebrew upgrades break the MCP configuration because
// the ophis library resolves symlinks to version-specific paths.
var Cmd = buildMCPCommand()

func buildMCPCommand() *cobra.Command {
	// Get the base ophis command (includes start, tools, claude, vscode)
	cmd := ophis.Command(config)

	// Remove the ophis claude and vscode commands that have the symlink resolution bug
	removeSubcommand(cmd, "claude")
	removeSubcommand(cmd, "vscode")

	// Add our custom claude and vscode commands that preserve symlink paths
	cmd.AddCommand(claude.Command())
	cmd.AddCommand(vscode.Command())

	return cmd
}

// removeSubcommand removes a subcommand by name from a parent command
func removeSubcommand(parent *cobra.Command, name string) {
	for _, sub := range parent.Commands() {
		if sub.Name() == name {
			parent.RemoveCommand(sub)
			return
		}
	}
}
