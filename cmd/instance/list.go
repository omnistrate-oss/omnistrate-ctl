package instance

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/omnistrate-oss/omnistrate-ctl/cmd/common"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/config"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/dataaccess"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/model"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/utils"
	openapiclientfleet "github.com/omnistrate-oss/omnistrate-sdk-go/fleet"
	openapiclientv1 "github.com/omnistrate-oss/omnistrate-sdk-go/v1"
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

	// Get all instances using the paginated Fleet instances API.
	listedInstances, err := fetchListedInstances(
		cmd.Context(),
		token,
		dataaccess.ListServices,
		dataaccess.ListServiceEnvironments,
		dataaccess.ListAllResourceInstances,
	)
	if err != nil {
		utils.HandleSpinnerError(spinner, sm, err)
		return err
	}

	formattedInstances := make([]model.Instance, 0)
	for i := range listedInstances {
		instance := listedInstances[i]
		formattedInstance := formatListedInstance(&instance, truncateNames)
		if formattedInstance.InstanceID == "" {
			continue
		}

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

type listServicesFunc func(context.Context, string) (*openapiclientv1.ListServiceResult, error)
type listServiceEnvironmentsFunc func(context.Context, string, string) (*openapiclientv1.ListServiceEnvironmentsResult, error)
type listResourceInstancesFunc func(context.Context, string, string, string, *dataaccess.ListResourceInstanceOptions) ([]openapiclientfleet.ResourceInstance, error)

func fetchListedInstances(
	ctx context.Context,
	token string,
	listServices listServicesFunc,
	listServiceEnvironments listServiceEnvironmentsFunc,
	listResourceInstances listResourceInstancesFunc,
) ([]openapiclientfleet.ResourceInstance, error) {
	services, err := listServices(ctx, token)
	if err != nil {
		return nil, err
	}

	filter := "excludeCloudAccounts"
	exclude := true
	options := &dataaccess.ListResourceInstanceOptions{
		Filter:                  &filter,
		ExcludeDetail:           &exclude,
		ExcludeNetworkTopology:  &exclude,
		ExcludeHAStatus:         &exclude,
		ExcludeIntegrations:     &exclude,
		ExcludeMaintenanceTasks: &exclude,
	}

	instances := make([]openapiclientfleet.ResourceInstance, 0)
	for _, serviceID := range services.GetIds() {
		environments, err := listServiceEnvironments(ctx, token, serviceID)
		if err != nil {
			if isSkippableListInstancesError(err) {
				continue
			}
			return nil, err
		}

		for _, environmentID := range environments.GetIds() {
			environmentInstances, err := listResourceInstances(ctx, token, serviceID, environmentID, options)
			if err != nil {
				if isSkippableListInstancesError(err) {
					continue
				}
				return nil, err
			}
			instances = append(instances, environmentInstances...)
		}
	}

	return instances, nil
}

func isSkippableListInstancesError(err error) bool {
	if err == nil {
		return false
	}

	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "not_found") && strings.Contains(msg, "host cluster not found") ||
		strings.Contains(msg, "bad_request") && strings.Contains(msg, "invalid request: service not found") ||
		strings.Contains(msg, "auth_failure")
}

func formatListedInstance(instance *openapiclientfleet.ResourceInstance, truncateNames bool) model.Instance {
	if instance == nil {
		return model.Instance{}
	}

	details := instance.GetConsumptionResourceInstanceResult()

	serviceName := instance.GetServiceName()
	planName := instance.GetProductTierName()
	if truncateNames {
		serviceName = utils.TruncateString(serviceName, defaultMaxNameLength)
		planName = utils.TruncateString(planName, defaultMaxNameLength)
	}

	subscriptionID := instance.GetSubscriptionId()
	if details.SubscriptionId != nil {
		subscriptionID = *details.SubscriptionId
	}

	version := instance.GetTierVersion()
	if details.TierVersion != nil {
		version = *details.TierVersion
	}

	return model.Instance{
		InstanceID:     details.GetId(),
		Service:        serviceName,
		Environment:    instance.GetServiceEnvName(),
		Plan:           planName,
		Version:        version,
		Resource:       listedInstanceResourceName(instance),
		CloudProvider:  instance.GetCloudProvider(),
		Region:         details.GetRegion(),
		Status:         details.GetStatus(),
		SubscriptionID: subscriptionID,
		Tags:           formatTags(details.CustomTags),
	}
}

func listedInstanceResourceName(instance *openapiclientfleet.ResourceInstance) string {
	if instance == nil {
		return ""
	}

	resourceID := instance.ConsumptionResourceInstanceResult.GetResourceID()
	for _, resource := range instance.GetResourceVersionSummaries() {
		if resourceID != "" && resource.GetResourceId() == resourceID {
			return resource.GetResourceName()
		}
	}

	for _, resource := range instance.GetResourceVersionSummaries() {
		if resource.GetResourceName() != "" {
			return resource.GetResourceName()
		}
	}

	return ""
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
