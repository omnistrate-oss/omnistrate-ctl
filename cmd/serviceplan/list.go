package serviceplan

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/omnistrate-oss/omnistrate-ctl/cmd/common"

	"github.com/omnistrate-oss/omnistrate-ctl/internal/config"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/dataaccess"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/model"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/utils"
	openapiclient "github.com/omnistrate-oss/omnistrate-sdk-go/v1"
	"github.com/spf13/cobra"
)

const (
	listExample = `# List service plans of the service postgres in the prod and dev environments
omnistrate-ctl service-plan list -f="service_name:postgres,environment:prod" -f="service:postgres,environment:dev"`
	defaultMaxNameLength = 30 // Maximum length of the name column in the table
)

func newListCmd(cfg servicePlanCommandConfig) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list [flags]",
		Short: "List Service Plans for your service",
		Long: `This command helps you list Service Plans for your service.
You can filter for specific service plans by using the filter flag.`,
		Example:      servicePlanExample(cfg.commandPath, listExample),
		RunE:         runList,
		SilenceUsage: true,
	}

	if cfg.listBrowserDefault {
		cmd.Annotations = map[string]string{"serviceplan-list-browser-default": "true"}
	}

	cmd.Flags().StringArrayP("filter", "f", []string{}, "Filter to apply to the list of service plans. E.g.: key1:value1,key2:value2, which filters service plans where key1 equals value1 and key2 equals value2. Allow use of multiple filters to form the logical OR operation. Supported keys: "+strings.Join(utils.GetSupportedFilterKeys(model.ServicePlanVersion{}), ",")+". Check the examples for more details.")
	cmd.Flags().Bool("truncate", false, "Truncate long names in the output")
	interactiveDescription := "Launch interactive list with fuzzy search and selection"
	if cfg.listBrowserDefault {
		interactiveDescription = "Launch interactive plan browser"
	}
	cmd.Flags().BoolP("interactive", "i", false, interactiveDescription)
	cmd.Args = cobra.NoArgs
	return cmd
}

func runList(cmd *cobra.Command, args []string) error {
	defer config.CleanupArgsAndFlags(cmd, &args)

	// Retrieve command-line flags
	output, _ := cmd.Flags().GetString("output")
	filters, _ := cmd.Flags().GetStringArray("filter")
	truncateNames, _ := cmd.Flags().GetBool("truncate")
	interactive, _ := cmd.Flags().GetBool("interactive")

	// Parse and validate filters
	filterMaps, err := utils.ParseFilters(filters, utils.GetSupportedFilterKeys(model.ServicePlanVersion{}))
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

	// Initialize spinner if output is not JSON and not interactive
	var sm utils.SpinnerManager
	var spinner *utils.Spinner
	if output != "json" && !shouldRunPlanBrowser(cmd, output, interactive) {
		sm = utils.NewSpinnerManager()
		spinner = sm.AddSpinner("Listing service plans...")
		sm.Start()
	}

	// List services
	listRes, err := dataaccess.ListServices(cmd.Context(), token)
	if err != nil {
		utils.HandleSpinnerError(spinner, sm, err)
		return err
	}

	formattedServicePlans, err := formatServicePlansFromServices(listRes.Services, filterMaps, truncateNames)
	if err != nil {
		utils.HandleSpinnerError(spinner, sm, err)
		return err
	}

	// Handle case when no service plans match
	if len(formattedServicePlans) == 0 {
		utils.HandleSpinnerSuccess(spinner, sm, "No service plans found.")
	} else {
		utils.HandleSpinnerSuccess(spinner, sm, "Service plans retrieved successfully.")
	}

	if shouldRunPlanBrowser(cmd, output, interactive) {
		catalog := buildServicePlanBrowserCatalog(listRes.Services, filterMaps)
		return runServicePlanBrowser(cmd.Context(), token, catalog)
	}

	// Interactive mode
	if interactive {
		return runInteractiveServicePlanList(formattedServicePlans)
	}

	// Format output as requested
	err = utils.PrintTextTableJsonArrayOutput(output, formattedServicePlans)
	if err != nil {
		return err
	}

	return nil
}

func runInteractiveServicePlanList(plans []model.ServicePlan) error {
	items := make([]utils.InteractiveListItem, len(plans))
	for i, plan := range plans {
		rawJSON, _ := json.Marshal(plan)
		items[i] = utils.NewInteractiveListItem(
			plan.PlanName,
			fmt.Sprintf("%s · %s · %s · %s/%s", plan.PlanID, plan.ServiceName, plan.Environment, plan.DeploymentType, plan.TenancyType),
			plan.PlanID,
			"",
			rawJSON,
		)
	}

	selected, err := utils.RunInteractiveList(utils.InteractiveListConfig{
		Title:    "Service Plans",
		Items:    items,
		ShowJSON: true,
	})
	if err != nil {
		return err
	}

	if selected != nil {
		var prettyJSON json.RawMessage
		if err := json.Unmarshal([]byte(selected.JSONData()), &prettyJSON); err == nil {
			data, _ := json.MarshalIndent(prettyJSON, "", "    ")
			fmt.Println(string(data))
		}
	}

	return nil
}

// Helper functions

func shouldRunPlanBrowser(cmd *cobra.Command, output string, interactive bool) bool {
	return output != "json" && (interactive || cmd.Annotations["serviceplan-list-browser-default"] == "true")
}

func formatServicePlansFromServices(services []openapiclient.DescribeServiceResult, filterMaps []map[string]string, truncateNames bool) ([]model.ServicePlan, error) {
	var formattedServicePlans []model.ServicePlan

	for _, service := range services {
		for _, env := range service.ServiceEnvironments {
			for _, servicePlan := range env.ServicePlans {
				formattedServicePlan := formatServicePlan(service.Id, service.Name, env.Name, servicePlan, truncateNames)

				match, err := utils.MatchesFilters(formattedServicePlan, filterMaps)
				if err != nil {
					return nil, err
				}

				if match {
					formattedServicePlans = append(formattedServicePlans, formattedServicePlan)
				}
			}
		}
	}

	return formattedServicePlans, nil
}

func formatServicePlan(serviceID, serviceName, envName string, servicePlan openapiclient.ServicePlan, truncateNames bool) model.ServicePlan {
	planName := servicePlan.Name

	if truncateNames {
		serviceName = utils.TruncateString(serviceName, defaultMaxNameLength)
		envName = utils.TruncateString(envName, defaultMaxNameLength)
		planName = utils.TruncateString(planName, defaultMaxNameLength)
	}

	return model.ServicePlan{
		PlanID:         servicePlan.ProductTierID,
		PlanName:       planName,
		ServiceID:      serviceID,
		ServiceName:    serviceName,
		Environment:    envName,
		DeploymentType: servicePlan.TierType,
		TenancyType:    servicePlan.ModelType,
	}
}
