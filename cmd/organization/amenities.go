package organization

import (
	"github.com/spf13/cobra"
)

var amenitiesCmd = &cobra.Command{
	Use:   "amenities [operation] [flags]",
	Short: "Manage organization amenities configuration templates",
	Long: `Manage organization-level amenities configuration templates.

This command helps you:
- Initialize organization-level amenities configuration templates
- Update amenities configuration templates for target environments

These templates define the organization's amenities policies that can be applied
to deployment cells through drift detection and synchronization operations.

Available operations:
  init        Initialize organization-level amenities configuration template
  update      Update organization amenities configuration template for target environment`,
	Run:          runAmenities,
	SilenceUsage: true,
}

func init() {
	amenitiesCmd.AddCommand(amenitiesInitCmd)
	amenitiesCmd.AddCommand(amenitiesUpdateCmd)
}

func runAmenities(cmd *cobra.Command, args []string) {
	err := cmd.Help()
	if err != nil {
		return
	}
}