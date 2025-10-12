package domain

import (
	"strings"

	"github.com/omnistrate-oss/omnistrate-ctl/cmd/common"

	"github.com/omnistrate-oss/omnistrate-ctl/internal/config"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/dataaccess"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/model"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/utils"
	openapiclientv1 "github.com/omnistrate-oss/omnistrate-sdk-go/v1"
	"github.com/spf13/cobra"
)

const (
	listExample = `# List domains
omctl domain list`
)

var listCmd = &cobra.Command{
	Use:   "list [flags]",
	Short: "List SaaS Portal Custom Domains",
	Long: `This command helps you list SaaS Portal Custom Domains.
You can filter for specific domains by using the filter flag.`,
	Example:      listExample,
	RunE:         runList,
	SilenceUsage: true,
}

func init() {
	listCmd.Flags().StringArrayP("filter", "f", []string{}, "Filter to apply to the list of domains. E.g.: key1:value1,key2:value2, which filters domains where key1 equals value1 and key2 equals value2. Allow use of multiple filters to form the logical OR operation. Supported keys: "+strings.Join(utils.GetSupportedFilterKeys(model.Domain{}), ",")+". Check the examples for more details.")
}

func runList(cmd *cobra.Command, args []string) error {
	defer config.CleanupArgsAndFlags(cmd, &args)

	// Retrieve command-line flags
	output, _ := cmd.Flags().GetString("output")
	filters, _ := cmd.Flags().GetStringArray("filter")

	// Parse and validate filters
	filterMaps, err := utils.ParseFilters(filters, utils.GetSupportedFilterKeys(model.Domain{}))
	if err != nil {
		utils.PrintError(err)
		return err
	}

	// Ensure user is logged in
	token, err := common.GetTokenWithLogin()
	if err != nil {
		utils.PrintError(err)
		return err
	}

	// Retrieve domains and domains
	listRes, err := dataaccess.ListDomains(cmd.Context(), token)
	if err != nil {
		utils.PrintError(err)
		return err
	}

	var formattedDomains []model.Domain

	// Process and filter domains
	for _, domain := range listRes.CustomDomains {
		formattedDomain := formatDomain(domain)

		match, err := utils.MatchesFilters(formattedDomain, filterMaps)
		if err != nil {
			utils.PrintError(err)
			return err
		}

		if match {
			formattedDomains = append(formattedDomains, formattedDomain)
		}
	}

	// Handle case when no domains match
	if len(formattedDomains) == 0 {
		utils.PrintInfo("No domains found.")
		return nil
	}

	// Format output as requested
	err = utils.PrintTextTableJsonArrayOutput(output, formattedDomains)
	if err != nil {
		return err
	}

	return nil
}

// Helper functions

func formatDomain(domain openapiclientv1.CustomDomain) model.Domain {
	return model.Domain{
		EnvironmentType: domain.EnvironmentType,
		Name:            domain.Name,
		Domain:          domain.CustomDomain,
		Status:          domain.Status,
		ClusterEndpoint: domain.ClusterEndpoint,
	}
}
