package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/fatih/color"
	"github.com/mitchellh/go-wordwrap"
	"github.com/omnistrate-oss/omnistrate-ctl/cmd/account"
	"github.com/omnistrate-oss/omnistrate-ctl/cmd/agent"
	"github.com/omnistrate-oss/omnistrate-ctl/cmd/alarms"
	"github.com/omnistrate-oss/omnistrate-ctl/cmd/audit"
	"github.com/omnistrate-oss/omnistrate-ctl/cmd/auth/login"
	"github.com/omnistrate-oss/omnistrate-ctl/cmd/auth/logout"
	"github.com/omnistrate-oss/omnistrate-ctl/cmd/build"
	"github.com/omnistrate-oss/omnistrate-ctl/cmd/cost"
	"github.com/omnistrate-oss/omnistrate-ctl/cmd/customnetwork"
	"github.com/omnistrate-oss/omnistrate-ctl/cmd/deploy"
	"github.com/omnistrate-oss/omnistrate-ctl/cmd/deploymentcell"
	"github.com/omnistrate-oss/omnistrate-ctl/cmd/docs"
	"github.com/omnistrate-oss/omnistrate-ctl/cmd/domain"
	"github.com/omnistrate-oss/omnistrate-ctl/cmd/environment"
	"github.com/omnistrate-oss/omnistrate-ctl/cmd/helm"
	"github.com/omnistrate-oss/omnistrate-ctl/cmd/inspect"
	"github.com/omnistrate-oss/omnistrate-ctl/cmd/instance"
	"github.com/omnistrate-oss/omnistrate-ctl/cmd/mcp"
	"github.com/omnistrate-oss/omnistrate-ctl/cmd/operations"
	"github.com/omnistrate-oss/omnistrate-ctl/cmd/secret"
	"github.com/omnistrate-oss/omnistrate-ctl/cmd/service"
	"github.com/omnistrate-oss/omnistrate-ctl/cmd/serviceplan"
	"github.com/omnistrate-oss/omnistrate-ctl/cmd/servicesorchestration"
	"github.com/omnistrate-oss/omnistrate-ctl/cmd/subscription"
	"github.com/omnistrate-oss/omnistrate-ctl/cmd/upgrade"
	"github.com/omnistrate-oss/omnistrate-ctl/cmd/workflow"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/config"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/utils"
	"github.com/spf13/cobra"
)

const versionDescription = "Omnistrate CTL %s"

// RootCmd represents the base command when called without any subcommands
var RootCmd = &cobra.Command{
	Use:   "omnistrate-ctl",
	Short: "Manage your Omnistrate SaaS from the command line",
	Long: wordwrap.WrapString(`
Omnistrate CTL is the command line interface to build, manage, and operate software or agent distributions anywhere - across clouds, regions, customer-controlled environments, and on-premises - using Omnistrate platform.

Key Features:
- Build your software or agent distributions: Create service plan from code, docker images, docker compose, helm, terraform and more.
- Debug and Test your service deployments: Easily debug and test your service deployments in your development environment.
- Automate your CI/CD pipelines: Integrate Omnistrate CTL into your CI/CD pipelines to automate the build, deployment, and management of your service plans.
- Operate at Scale: Manage large-scale deployments with robust tools and automation.

Resources:
- Getting Started: https://docs.omnistrate.com/getting-started/
- Quick Start: https://docs.omnistrate.com/getting-started/getting-started-with-ctl/
- AI Agent Setup: https://docs.omnistrate.com/getting-started/mcp-server/
- CTL Manual: https://ctl.omnistrate.cloud/

Available Commands:

`, 100),
	Run:               runRoot,
	DisableAutoGenTag: true,
	Aliases:           []string{"omctl"},
}

func runRoot(cmd *cobra.Command, args []string) {
	// Check if the version flag is set
	versionFlag, err := cmd.Flags().GetBool("version")
	if err == nil && versionFlag {
		fmt.Println(fmt.Sprintf(versionDescription, config.Version))
		return
	}

	printLogo()
	err = cmd.Help()
	if err != nil {
		return
	}
}

// printLogo prints an ASCII logo, which was generated with figlet
func printLogo() {
	fmt.Println()
	colors := []color.Attribute{
		color.FgRed, color.FgYellow, color.FgGreen, color.FgCyan, color.FgBlue, color.FgMagenta,
	}
	for i, r := range figletStr {
		fmt.Printf("%s", color.New(colors[i%len(colors)]).SprintFunc()(string(r)))
	}
}

const figletStr = `                  _     __           __     
 ___  __ _  ___  (_)__ / /________ _/ /____ 
/ _ \/  ' \/ _ \/ (_-</ __/ __/ _ ` + "`" + `/ __/ -_)
\___/_/_/_/_//_/_/___/\__/_/  \_,_/\__/\__/ 

`

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the RootCmd.
func Execute() {
	ctx := context.Background()
	utils.ConfigureLoggingFromEnvOnce()
	err := RootCmd.ExecuteContext(ctx)
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	RootCmd.PersistentFlags().BoolP("version", "v", false, "Print the version number of omnistrate-ctl")
	RootCmd.PersistentFlags().StringP("output", "o", "table", "Output format (text|table|json)")

	RootCmd.AddCommand(login.LoginCmd)
	RootCmd.AddCommand(logout.LogoutCmd)

	RootCmd.AddCommand(build.BuildCmd)
	RootCmd.AddCommand(build.BuildFromRepoCmd)
	RootCmd.AddCommand(deploy.DeployCmd)

	RootCmd.AddCommand(service.Cmd)
	RootCmd.AddCommand(account.Cmd)
	RootCmd.AddCommand(agent.Cmd)
	RootCmd.AddCommand(alarms.Cmd)
	RootCmd.AddCommand(docs.Cmd)
	RootCmd.AddCommand(domain.Cmd)
	RootCmd.AddCommand(upgrade.Cmd)
	RootCmd.AddCommand(helm.Cmd)
	RootCmd.AddCommand(instance.Cmd)
	RootCmd.AddCommand(serviceplan.Cmd)
	RootCmd.AddCommand(subscription.Cmd)
	RootCmd.AddCommand(environment.Cmd)
	RootCmd.AddCommand(customnetwork.Cmd)
	RootCmd.AddCommand(deploymentcell.Cmd)
	RootCmd.AddCommand(servicesorchestration.Cmd)
	RootCmd.AddCommand(inspect.Cmd)
	RootCmd.AddCommand(secret.Cmd)
	RootCmd.AddCommand(workflow.Cmd)
	RootCmd.AddCommand(cost.Cmd)
	RootCmd.AddCommand(operations.Cmd)
	RootCmd.AddCommand(audit.Cmd)
	RootCmd.AddCommand(mcp.Cmd)

	// Hide the default completion command
	RootCmd.Root().CompletionOptions.DisableDefaultCmd = true
}
