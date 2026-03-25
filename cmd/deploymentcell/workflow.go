package deploymentcell

import (
	"github.com/spf13/cobra"
)

var workflowCmd = &cobra.Command{
	Use:          "workflow",
	Short:        "Manage deployment cell workflows",
	Long:         `Commands to list and describe workflows for deployment cells.`,
	Run:          runWorkflow,
	SilenceUsage: true,
}

func init() {
	workflowCmd.AddCommand(workflowListCmd)
	workflowCmd.AddCommand(workflowDescribeCmd)
	workflowCmd.AddCommand(workflowEventsCmd)
}

func runWorkflow(cmd *cobra.Command, args []string) {
	err := cmd.Help()
	if err != nil {
		return
	}
}
