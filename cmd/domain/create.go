package domain

import (
	"fmt"
	"strings"

	"github.com/omnistrate-oss/omnistrate-ctl/cmd/common"
	openapiclientv1 "github.com/omnistrate-oss/omnistrate-sdk-go/v1"

	"github.com/omnistrate-oss/omnistrate-ctl/internal/dataaccess"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/utils"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

const (
	createExample = `# Create a custom domain for dev environment
omnistrate-ctl domain create dev --domain=abc.dev --environment-type=dev

# Create a custom domain for prod environment
omnistrate-ctl domain create abc.cloud --domain=abc.cloud --environment-type=prod`
)

var createCmd = &cobra.Command{
	Use:          "create [flags]",
	Short:        "Create a Custom Domain",
	Long:         `This command helps you create a Custom Domain.`,
	Example:      createExample,
	RunE:         runCreate,
	SilenceUsage: true,
}

func init() {
	createCmd.Args = cobra.ExactArgs(1) // Require exactly one argument

	createCmd.Flags().String("domain", "", "Custom domain")
	createCmd.Flags().String("environment-type", "", "Type of environment. Valid options include: 'dev', 'prod', 'qa', 'canary', 'staging', 'private'")

	err := createCmd.MarkFlagRequired("domain")
	if err != nil {
		return
	}
	err = createCmd.MarkFlagRequired("environment-type")
	if err != nil {
		return
	}
}

func runCreate(cmd *cobra.Command, args []string) error {
	// Get flags
	domain, _ := cmd.Flags().GetString("domain")
	environmentType, _ := cmd.Flags().GetString("environment-type")
	output, _ := cmd.Flags().GetString("output")

	// Validate user is currently logged in
	token, err := common.GetTokenWithLogin()
	if err != nil {
		utils.PrintError(err)
		return err
	}

	domains, err := dataaccess.ListDomains(cmd.Context(), token)
	if err != nil {
		utils.PrintError(err)
		return err
	}

	for _, d := range domains.CustomDomains {
		// Check if domain with the same name already exists
		if d.Name == args[0] {
			err = errors.New("domain with the same name already exists, please choose a different name. You can use 'omnistrate-ctl get domain' to list all existing domains.")
			utils.PrintError(err)
			return err
		}

		// Check if domain is registered
		if d.CustomDomain == domain {
			err = errors.New(fmt.Sprintf("%s is already registered with another domain, please choose a different domain. You can use 'omnistrate-ctl get domain' to list all existing domains.", domain))
			utils.PrintError(err)
			return err
		}

		// Check if domain of the same environment type already exists
		if d.EnvironmentType == strings.ToUpper(environmentType) {
			err = errors.New("domain with the same environment type already exists, please choose a different environment type. You can use 'omnistrate-ctl get domain' to list all existing domains.")
			utils.PrintError(err)
			return err
		}
	}

	// Create domain
	err = dataaccess.CreateDomain(cmd.Context(), token, args[0], "Custom domain for "+environmentType+" environment", strings.ToUpper(environmentType), domain)
	if err != nil {
		utils.PrintError(err)
		return err
	}
	if output != "json" {
		utils.PrintSuccess("Domain created successfully")
	}

	domains, err = dataaccess.ListDomains(cmd.Context(), token)
	if err != nil {
		utils.PrintError(err)
		return err
	}

	var customDomain openapiclientv1.CustomDomain
	for _, d := range domains.CustomDomains {
		if d.Name == args[0] {
			customDomain = d
			break
		}
	}

	err = utils.PrintTextTableJsonOutput(output, customDomain)
	if err != nil {
		return err
	}

	if output != "json" {
		dataaccess.PrintNextStepVerifyDomainMsg(customDomain.ClusterEndpoint)
	}

	return nil
}
