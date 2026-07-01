package instance

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/omnistrate-oss/omnistrate-ctl/cmd/common"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/config"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/dataaccess"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/model"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/utils"
	openapiclientfleet "github.com/omnistrate-oss/omnistrate-sdk-go/fleet"
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
	listCmd.Flags().BoolP("interactive", "i", false, "Launch interactive list with fuzzy search and selection")
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
	interactive, err := cmd.Flags().GetBool("interactive")
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

	// Initialize spinner if output is not JSON and not interactive
	var sm utils.SpinnerManager
	var spinner *utils.Spinner
	if output != common.OutputTypeJson && !interactive {
		sm = utils.NewSpinnerManager()
		spinner = sm.AddSpinner("Listing instance deployments...")
		sm.Start()
	}

	// Get instances matching supported filters from the backend, then keep the
	// existing local checks as a final compatibility guard.
	searchFilters := buildResourceInstanceSearchFilters(filterMaps, parsedTagFilters)
	var searchRes *openapiclientfleet.SearchInventoryResult
	if hasResourceInstanceFilters(searchFilters) {
		searchRes, err = dataaccess.SearchInventory(cmd.Context(), token, "resourceinstance:i", searchFilters)
	} else {
		searchRes, err = dataaccess.SearchInventory(cmd.Context(), token, "resourceinstance:i")
	}
	if err != nil {
		utils.HandleSpinnerError(spinner, sm, err)
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
			utils.HandleSpinnerError(spinner, sm, err)
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
	} else {
		utils.HandleSpinnerSuccess(spinner, sm, fmt.Sprintf("Found %d instance(s).", len(formattedInstances)))
	}

	// Interactive mode: launch TUI list
	if interactive {
		return runInteractiveInstanceList(formattedInstances)
	}

	// Print output
	err = utils.PrintTextTableJsonArrayOutput(output, formattedInstances)
	if err != nil {
		return err
	}

	return nil
}

func hasResourceInstanceFilters(filters openapiclientfleet.SearchInventoryFilters) bool {
	return filters.ResourceInstance != nil &&
		(len(filters.ResourceInstance.Predicates) > 0 || len(filters.ResourceInstance.Tags) > 0)
}

func buildResourceInstanceSearchFilters(filterMaps []map[string]string, tagFilters map[string]string) openapiclientfleet.SearchInventoryFilters {
	filters := openapiclientfleet.SearchInventoryFilters{
		ResourceInstance: &openapiclientfleet.ResourceInstanceSearchFilters{},
	}

	for _, filterMap := range filterMaps {
		predicate := openapiclientfleet.ResourceInstanceFilterGroup{}
		setResourceInstanceFilterGroupField(filterMap["instance_id"], predicate.SetInstanceId)
		setResourceInstanceFilterGroupField(filterMap["service"], predicate.SetServiceName)
		setResourceInstanceFilterGroupField(filterMap["environment"], predicate.SetEnvironmentName)
		setResourceInstanceFilterGroupField(filterMap["plan"], predicate.SetProductTierName)
		setResourceInstanceFilterGroupField(filterMap["version"], predicate.SetProductTierVersion)
		setResourceInstanceFilterGroupField(filterMap["resource"], predicate.SetResourceName)
		setResourceInstanceFilterGroupField(filterMap["cloud_provider"], predicate.SetCloudProvider)
		setResourceInstanceFilterGroupField(filterMap["region"], predicate.SetRegionCode)
		setResourceInstanceFilterGroupField(filterMap["status"], predicate.SetStatus)
		setResourceInstanceFilterGroupField(filterMap["subscription_id"], predicate.SetSubscriptionId)

		if hasResourceInstanceFilterGroupFields(predicate) {
			filters.ResourceInstance.Predicates = append(filters.ResourceInstance.Predicates, predicate)
		}
	}

	for key, value := range tagFilters {
		filters.ResourceInstance.Tags = append(filters.ResourceInstance.Tags, openapiclientfleet.ResourceInstanceTagFilter{
			Key:   key,
			Value: value,
		})
	}

	return filters
}

func setResourceInstanceFilterGroupField(value string, setter func(string)) {
	if value != "" {
		setter(value)
	}
}

func hasResourceInstanceFilterGroupFields(f openapiclientfleet.ResourceInstanceFilterGroup) bool {
	return f.HasInstanceId() ||
		f.HasServiceName() ||
		f.HasEnvironmentName() ||
		f.HasProductTierName() ||
		f.HasProductTierVersion() ||
		f.HasResourceName() ||
		f.HasCloudProvider() ||
		f.HasRegionCode() ||
		f.HasStatus() ||
		f.HasSubscriptionId()
}

func runInteractiveInstanceList(instances []model.Instance) error {
	items := make([]utils.InteractiveListItem, len(instances))
	for i, inst := range instances {
		rawJSON, _ := json.Marshal(inst)
		desc := fmt.Sprintf("%s · %s · %s/%s · %s",
			inst.Service, inst.Plan, inst.CloudProvider, inst.Region, inst.InstanceID)
		items[i] = utils.NewInteractiveListItem(
			inst.InstanceID,
			desc,
			inst.InstanceID,
			inst.Status,
			rawJSON,
		)
	}

	selected, err := utils.RunInteractiveList(utils.InteractiveListConfig{
		Title:    "Instance Deployments",
		Items:    items,
		ShowJSON: true,
	})
	if err != nil {
		return err
	}

	if selected != nil {
		// Print the selected instance's JSON detail
		var prettyJSON json.RawMessage
		if err := json.Unmarshal([]byte(selected.JSONData()), &prettyJSON); err == nil {
			data, _ := json.MarshalIndent(prettyJSON, "", "    ")
			fmt.Println(string(data))
		}
	}

	return nil
}
