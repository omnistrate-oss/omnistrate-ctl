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
			"agent install",
			"build",
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
