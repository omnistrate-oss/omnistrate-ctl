package instance

import "github.com/spf13/cobra"

var breakpointCmd = &cobra.Command{
	Use:          "breakpoint [operation] [flags]",
	Short:        "Manage instance workflow breakpoints",
	Long:         "Manage workflow breakpoints for an instance.",
	Run:          runBreakpoint,
	SilenceUsage: true,
}

func init() {
	breakpointCmd.AddCommand(breakpointListCmd)
	breakpointCmd.AddCommand(breakpointResumeCmd)
}

func runBreakpoint(cmd *cobra.Command, args []string) {
	err := cmd.Help()
	if err != nil {
		return
	}
}
