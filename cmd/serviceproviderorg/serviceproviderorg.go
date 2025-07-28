package serviceproviderorg

import (
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:   "serviceproviderorg",
	Short: "Manage service provider organization configuration",
	Long: `Manage service provider organization-level configuration including amenities templates.

The service provider organization commands allow you to manage organization-level
settings that apply across all services and deployment cells.

This includes:
- Amenities configuration templates
- Organization-level defaults
- Environment-specific settings`,
	Aliases: []string{"sporg", "org"},
}

func init() {
	Cmd.AddCommand(initDefaultTemplateCmd)
	Cmd.AddCommand(updateServiceProviderOrgCmd)
}
