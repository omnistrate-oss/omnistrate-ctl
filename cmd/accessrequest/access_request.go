package accessrequest

import "github.com/spf13/cobra"

// Cmd is the root command for service-provider deployment-cell access requests.
var Cmd = &cobra.Command{
	Use:   "access-request [operation] [flags]",
	Short: "Manage deployment-cell access requests",
	Long:  "List, describe, approve, deny, and revoke service-provider deployment-cell access requests.",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, _ []string) error {
		return cmd.Help()
	},
	SilenceUsage: true,
}

func init() {
	Cmd.AddCommand(describeCmd)
	Cmd.AddCommand(listCmd)
	Cmd.AddCommand(approveCmd)
	Cmd.AddCommand(denyCmd)
	Cmd.AddCommand(revokeCmd)
}
