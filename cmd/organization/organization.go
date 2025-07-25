package organization

import (
	"github.com/spf13/cobra"
)

// Cmd represents the organization command
var Cmd = &cobra.Command{
	Use:   "organization [command]",
	Short: "Manage organization-level configurations",
	Long: `Manage organization-level configurations including amenities templates.

This command provides access to organization-level management operations including:
- Initialize and update amenities configuration templates
- Manage environment-specific organization settings

These operations affect organization-wide policies and templates that can be
applied to deployment cells within the organization.`,
	Run:          runOrganization,
	SilenceUsage: true,
}

func init() {
	// Add organization amenities command
	Cmd.AddCommand(amenitiesCmd)
}

func runOrganization(cmd *cobra.Command, args []string) {
	err := cmd.Help()
	if err != nil {
		return
	}
}