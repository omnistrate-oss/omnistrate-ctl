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
