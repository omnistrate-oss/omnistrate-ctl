package operations

import (
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:   "operations",
	Short: "Operations and health monitoring commands",
	Long:  "Manage and monitor operational health of your services, deployment cells, and infrastructure.",
}

func init() {
	Cmd.AddCommand(healthCmd)
	Cmd.AddCommand(deploymentCellHealthCmd)
	Cmd.AddCommand(eventsCmd)
}