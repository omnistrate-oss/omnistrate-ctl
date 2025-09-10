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
			"help",
			"completion",
			"login",
			"logout",
			"services-orchestration",
			"secret",
			"inspect",
			"debug",
			"disable-feature",
			"enable-feature",
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
