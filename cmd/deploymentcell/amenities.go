package deploymentcell

import (
	"github.com/spf13/cobra"
)

var amenitiesCmd = &cobra.Command{
	Use:   "amenities [operation] [flags]",
	Short: "Manage deployment cell amenities synchronization",
	Long: `Manage deployment cell amenities synchronization with organization templates.

This command helps you:
- Check deployment cells for configuration drift against organization templates
- Sync deployment cells with organization+environment templates
- Apply pending configuration changes to deployment cells

These operations work with deployment cells to align them with organization-level
amenities templates. Use the 'organization amenities' commands to manage the
templates themselves.

Available operations:
  check-drift Check deployment cell for configuration drift
  sync        Sync deployment cell with organization+environment template
  apply       Apply pending changes to deployment cell`,
	Run:          runAmenities,
	SilenceUsage: true,
}

func init() {
	amenitiesCmd.AddCommand(amenitiesCheckDriftCmd)
	amenitiesCmd.AddCommand(amenitiesSyncCmd)
	amenitiesCmd.AddCommand(amenitiesApplyCmd)
	
	// Add amenities command to the main deployment cell command
	Cmd.AddCommand(amenitiesCmd)
}

func runAmenities(cmd *cobra.Command, args []string) {
	err := cmd.Help()
	if err != nil {
		return
	}
}