package serviceplan

import (
	"strings"

	"github.com/omnistrate-oss/omnistrate-ctl/cmd/common"

	"github.com/chelnak/ysmrr"
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

var listCmd = &cobra.Command{
	Use:   "list [flags]",
	Short: "List Service Plans for your service",
	Long: `This command helps you list Service Plans for your service.
You can filter for specific service plans by using the filter flag.`,
	Example:      listExample,
	RunE:         runList,
	SilenceUsage: true,
}

func init() {

	listCmd.Flags().StringArrayP("filter", "f", []string{}, "Filter to apply to the list of service plans. E.g.: key1:value1,key2:value2, which filters service plans where key1 equals value1 and key2 equals value2. Allow use of multiple filters to form the logical OR operation. Supported keys: "+strings.Join(utils.GetSupportedFilterKeys(model.ServicePlanVersion{}), ",")+". Check the examples for more details.")
	listCmd.Flags().Bool("truncate", false, "Truncate long names in the output")
	listCmd.Args = cobra.NoArgs
}

func runList(cmd *cobra.Command, args []string) error {
	defer config.CleanupArgsAndFlags(cmd, &args)

	// Retrieve command-line flags
	output, _ := cmd.Flags().GetString("output")
	filters, _ := cmd.Flags().GetStringArray("filter")
	truncateNames, _ := cmd.Flags().GetBool("truncate")

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

	// Initialize spinner if output is not JSON
	var sm ysmrr.SpinnerManager
	var spinner *ysmrr.Spinner
	if output != "json" {
		sm = ysmrr.NewSpinnerManager()
		spinner = sm.AddSpinner("Listing service plans...")
		sm.Start()
	}

	// List services
	listRes, err := dataaccess.ListServices(cmd.Context(), token)
	if err != nil {
		utils.HandleSpinnerError(spinner, sm, err)
		return err
	}

	var formattedServicePlans []model.ServicePlan

	// Process and filter service plans
	for _, service := range listRes.Services {
		for _, env := range service.ServiceEnvironments {
			for _, servicePlan := range env.ServicePlans {
				formattedServicePlan := formatServicePlan(service.Id, service.Name, env.Name, servicePlan, truncateNames)

				match, err := utils.MatchesFilters(formattedServicePlan, filterMaps)
				if err != nil {
					utils.HandleSpinnerError(spinner, sm, err)
					return err
				}

				if match {
					formattedServicePlans = append(formattedServicePlans, formattedServicePlan)
				}
			}
		}
	}

	// Handle case when no service plans match
	if len(formattedServicePlans) == 0 {
		utils.HandleSpinnerSuccess(spinner, sm, "No service plans found.")
	} else {
		utils.HandleSpinnerSuccess(spinner, sm, "Service plans retrieved successfully.")
	}

	// Format output as requested
	err = utils.PrintTextTableJsonArrayOutput(output, formattedServicePlans)
	if err != nil {
		return err
	}

	return nil
}

// Helper functions

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
