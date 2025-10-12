package mcp

import (
	"github.com/njayp/ophis"
	"github.com/njayp/ophis/tools"
	"github.com/spf13/cobra"
)

var config = &ophis.Config{
	// Customize command filtering and output handling
	GeneratorOptions: []tools.GeneratorOption{
		// Command filtering
		tools.AddFilter(tools.Exclude([]string{
			"mcp",
			"build",
			"domain",
			"alarms",
			"inspect",
			"helm",
			"instance adopt",
			"instance continue-deployment",
			"instance debug",
			"instance enable-debug-mode",
			"instance disable-debug-mode",
			"instance patch-deployment",
			"instance version-upgrade",
			"environment",
			"help",
			"completion",
			"login",
			"logout",
			"services-orchestration",
			"secret",
			"disable-feature",
			"enable-feature",
			"build-from-repo", // we only want build from mcp
		})),
		// Exclude hidden commands
		tools.AddFilter(tools.Hidden()),
		// Exclude commands that have no Run or RunE function
		tools.AddFilter(func(cmd *cobra.Command) bool {
			if cmd.Run == nil && cmd.RunE == nil {
				return false
			}
			return true
		}),
		// Filter non leaf commands
		tools.AddFilter(func(cmd *cobra.Command) bool {
			return len(cmd.Commands()) == 0
		}),
	},
}

var Cmd = ophis.Command(config)
