package serviceplan

import (
	"strings"

	"github.com/omnistrate-oss/omnistrate-ctl/internal/utils"
	"github.com/spf13/cobra"
)

const (
	legacyServicePlanCommandPath = "service-plan"
	nestedServicePlanCommandPath = "service plan"
	servicePlanDeprecationNotice = "`service-plan` is deprecated; use `service plan` instead."
)

type servicePlanCommandConfig struct {
	commandPath        string
	use                string
	legacyNotice       bool
	listBrowserDefault bool
}

var Cmd = NewLegacyCommand()

func NewLegacyCommand() *cobra.Command {
	return newCommand(servicePlanCommandConfig{
		commandPath:  legacyServicePlanCommandPath,
		use:          "service-plan [operation] [flags]",
		legacyNotice: true,
	})
}

func NewNestedCommand() *cobra.Command {
	return newCommand(servicePlanCommandConfig{
		commandPath:        nestedServicePlanCommandPath,
		use:                "plan [operation] [flags]",
		listBrowserDefault: true,
	})
}

func newCommand(cfg servicePlanCommandConfig) *cobra.Command {
	cmd := &cobra.Command{
		Use:          cfg.use,
		Short:        "Manage Service Plans for your service",
		Long:         `This command helps you manage the service plans for your service.`,
		Run:          run,
		SilenceUsage: true,
	}

	if cfg.legacyNotice {
		cmd.PersistentPreRun = runLegacyNotice
	}

	cmd.AddCommand(newDeleteCmd(cfg.commandPath))
	cmd.AddCommand(newReleaseCmd(cfg.commandPath))
	cmd.AddCommand(newSetDefaultCmd(cfg.commandPath))
	cmd.AddCommand(newDescribeCmd(cfg.commandPath))
	cmd.AddCommand(newDescribeVersionCmd(cfg.commandPath))
	cmd.AddCommand(newListCmd(cfg))
	cmd.AddCommand(newListVersionsCmd(cfg.commandPath))
	cmd.AddCommand(newEnableCmd(cfg.commandPath))
	cmd.AddCommand(newDisableCmd(cfg.commandPath))
	cmd.AddCommand(newUpdateCmd(cfg.commandPath))

	return cmd
}

func run(cmd *cobra.Command, args []string) {
	err := cmd.Help()
	if err != nil {
		return
	}
}

func runLegacyNotice(cmd *cobra.Command, args []string) {
	output, _ := cmd.Flags().GetString(OutputFlag)
	if shouldPrintLegacyServicePlanNotice(output) {
		utils.PrintWarning(servicePlanDeprecationNotice)
	}
}

func shouldPrintLegacyServicePlanNotice(output string) bool {
	return output != "json"
}

func servicePlanExample(commandPath, example string) string {
	return strings.ReplaceAll(example, "omnistrate-ctl service-plan", "omnistrate-ctl "+commandPath)
}
