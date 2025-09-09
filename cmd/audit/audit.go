package audit

import (
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:   "audit",
	Short: "Audit events and logging management",
	Long:  "Access and manage audit events for services, instances, and operations.",
}

func init() {
	Cmd.AddCommand(listCmd)
}