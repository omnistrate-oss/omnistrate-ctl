package deploymentcell

import (
	"github.com/spf13/cobra"
)

var amenitiesCmd = &cobra.Command{
	Use:   "amenities [operation] [flags]",
	Short: "Manage deployment cell amenities configuration",
	Long: `Manage organization-level amenities configuration and deployment cell synchronization.

This command helps you:
- Initialize organization-level amenities configuration
- Update amenities configuration for target environments
- Check deployment cells for configuration drift
- Sync deployment cells with organization templates
- Apply pending configuration changes

Available operations:
  init        Initialize organization-level amenities configuration
  update      Update organization amenities configuration for target environment
  check-drift Check deployment cell for configuration drift
  sync        Sync deployment cell with organization+environment template
  apply       Apply pending changes to deployment cell
  status      Show amenities status for deployment cell`,
	Run:          runAmenities,
	SilenceUsage: true,
}

func init() {
	amenitiesCmd.AddCommand(amenitiesInitCmd)
	amenitiesCmd.AddCommand(amenitiesUpdateCmd)
	amenitiesCmd.AddCommand(amenitiesCheckDriftCmd)
	amenitiesCmd.AddCommand(amenitiesSyncCmd)
	amenitiesCmd.AddCommand(amenitiesApplyCmd)
	amenitiesCmd.AddCommand(amenitiesStatusCmd)
	
	// Add amenities command to the main deployment cell command
	Cmd.AddCommand(amenitiesCmd)
}

func runAmenities(cmd *cobra.Command, args []string) {
	err := cmd.Help()
	if err != nil {
		return
	}
}