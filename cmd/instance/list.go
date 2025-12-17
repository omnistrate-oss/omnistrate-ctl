package instance

import (
	"strings"

	"github.com/chelnak/ysmrr"
	"github.com/omnistrate-oss/omnistrate-ctl/cmd/common"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/config"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/dataaccess"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/model"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/utils"
	"github.com/spf13/cobra"
)

const (
	listExample = `# List instance deployments of the service postgres in the prod and dev environments
omnistrate-ctl instance list -f="service:postgres,environment:Production" -f="service:postgres,environment:Dev"

# List instances with specific tags
omnistrate-ctl instance list --tag env=prod --tag team=backend

# Combine regular filters with tag filters
omnistrate-ctl instance list -f="service:postgres" --tag env=prod`
	defaultMaxNameLength = 30 // Maximum length of the name column in the table
)

var listCmd = &cobra.Command{
	Use:   "list [flags]",
	Short: "List instance deployments for your service",
	Long: `This command helps you list instance deployments for your service.
You can filter for specific instances by using the filter flag.`,
	Example:      listExample,
	RunE:         runList,
	SilenceUsage: true,
}

func init() {
	listCmd.Flags().StringArrayP("filter", "f", []string{}, "Filter to apply to the list of instances. E.g.: key1:value1,key2:value2, which filters instances where key1 equals value1 and key2 equals value2. Allow use of multiple filters to form the logical OR operation. Supported keys: "+strings.Join(utils.GetSupportedFilterKeys(model.Instance{}), ",")+". Check the examples for more details.")
	listCmd.Flags().StringArray("tag", []string{}, "Filter instances by tags. Specify tags as key=value pairs. Multiple --tag flags can be used to filter by multiple tags (all tags must match).")
	listCmd.Flags().Bool("truncate", false, "Truncate long names in the output")
}

func runList(cmd *cobra.Command, args []string) error {
	defer config.CleanupArgsAndFlags(cmd, &args)

	// Retrieve flags
	output, err := cmd.Flags().GetString("output")
	if err != nil {
		utils.PrintError(err)
		return err
	}
	filters, err := cmd.Flags().GetStringArray("filter")
	if err != nil {
		utils.PrintError(err)
		return err
	}
	tagFilters, err := cmd.Flags().GetStringArray("tag")
	if err != nil {
		utils.PrintError(err)
		return err
	}
	truncateNames, err := cmd.Flags().GetBool("truncate")
	if err != nil {
		utils.PrintError(err)
		return err
	}

	// Parse filters into a map
	filterMaps, err := utils.ParseFilters(filters, utils.GetSupportedFilterKeys(model.Instance{}))
	if err != nil {
		utils.PrintError(err)
		return err
	}

	// Parse tag filters
	parsedTagFilters, err := parseTagFilters(tagFilters)
	if err != nil {
		utils.PrintError(err)
		return err
	}

	// Validate user is currently logged in
	token, err := common.GetTokenWithLogin()
	if err != nil {
		utils.PrintError(err)
		return err
	}

	// Initialize spinner if output is not JSON
	var sm ysmrr.SpinnerManager
	var spinner *ysmrr.Spinner
	if output != common.OutputTypeJson {
		sm = ysmrr.NewSpinnerManager()
		spinner = sm.AddSpinner("Listing instance deployments...")
		sm.Start()
	}

	// Get all instances
	searchRes, err := dataaccess.SearchInventory(cmd.Context(), token, "resourceinstance:i")
	if err != nil {
		utils.PrintError(err)
		return err
	}

	formattedInstances := make([]model.Instance, 0)
	for i := range searchRes.ResourceInstanceResults {
		instance := searchRes.ResourceInstanceResults[i]
		if instance.Id == "" {
			continue
		}

		// Format instance
		formattedInstance := formatInstance(&instance, truncateNames)

		// Check if the instance matches the filters
		ok, err := utils.MatchesFilters(formattedInstance, filterMaps)
		if err != nil {
			utils.PrintError(err)
			return err
		}
		if !ok {
			continue
		}

		// Check if the instance matches the tag filters
		if !matchesTagFilters(formattedInstance.Tags, parsedTagFilters) {
			continue
		}

		formattedInstances = append(formattedInstances, formattedInstance)
	}

	if len(formattedInstances) == 0 {
		utils.HandleSpinnerSuccess(spinner, sm, "No instances found.")
	}

	// Print output
	err = utils.PrintTextTableJsonArrayOutput(output, formattedInstances)
	if err != nil {
		return err
	}

	return nil
}
