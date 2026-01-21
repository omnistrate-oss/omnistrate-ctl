package serviceplan

import (
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/omnistrate-oss/omnistrate-ctl/cmd/common"

	"github.com/chelnak/ysmrr"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/config"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/dataaccess"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/model"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/utils"
	openapiclientfleet "github.com/omnistrate-oss/omnistrate-sdk-go/fleet"
	"github.com/spf13/cobra"
)

const (
	listVersionsExample = `# List service plan versions of the service postgres in the prod and dev environments
omnistrate-ctl service-plan list-versions postgres postgres -f="service_name:postgres,environment:prod" -f="service:postgres,environment:dev"`
)

var listVersionsCmd = &cobra.Command{
	Use:   "list-versions [service-name] [plan-name] [flags]",
	Short: "List Versions of a specific Service Plan",
	Long: `This command helps you list Versions of a specific Service Plan.
You can filter for specific service plan versions by using the filter flag.`,
	Example:      listVersionsExample,
	RunE:         runListVersions,
	SilenceUsage: true,
}

func init() {
	listVersionsCmd.Flags().StringP("service-id", "", "", "Service ID. Required if service name is not provided")
	listVersionsCmd.Flags().StringP("plan-id", "", "", "Plan ID. Required if plan name is not provided")
	listVersionsCmd.Flags().IntP("limit", "", -1, "List only the latest N service plan versions")
	listVersionsCmd.Flags().IntP("latest-n", "", -1, "List only the latest N service plan versions")
	listVersionsCmd.Flags().StringP("environment", "", "", "Environment name. Use this flag with service name and plan name to describe the version in a specific environment")

	listVersionsCmd.Flags().StringArrayP("filter", "f", []string{}, "Filter to apply to the list of service plan versions. E.g.: key1:value1,key2:value2, which filters service plans where key1 equals value1 and key2 equals value2. Allow use of multiple filters to form the logical OR operation. Supported keys: "+strings.Join(utils.GetSupportedFilterKeys(model.ServicePlanVersion{}), ",")+". Check the examples for more details.")
	listVersionsCmd.Flags().Bool("truncate", false, "Truncate long names in the output")
	err := listVersionsCmd.Flags().MarkHidden("latest-n")
	if err != nil {
		return
	}
}

func runListVersions(cmd *cobra.Command, args []string) error {
	defer config.CleanupArgsAndFlags(cmd, &args)

	// Retrieve command-line flags
	serviceID, _ := cmd.Flags().GetString("service-id")
	planID, _ := cmd.Flags().GetString("plan-id")
	latestN, _ := cmd.Flags().GetInt("latest-n")
	limit, _ := cmd.Flags().GetInt("limit")
	output, _ := cmd.Flags().GetString("output")
	filters, _ := cmd.Flags().GetStringArray("filter")
	truncateNames, _ := cmd.Flags().GetBool("truncate")
	environment, _ := cmd.Flags().GetString("environment")

	// Temporary workaround to support both latest-n and limit flags
	if limit != -1 {
		latestN = limit
	}

	// Validate input arguments
	if err := validateListVersionsArguments(args, serviceID, planID); err != nil {
		utils.PrintError(err)
		return err
	}

	// Parse and validate filters
	filterMaps, err := utils.ParseFilters(filters, utils.GetSupportedFilterKeys(model.ServicePlanVersion{}))
	if err != nil {
		utils.PrintError(err)
		return err
	}

	// Set service and service plan names if provided in args
	var serviceName, planName string
	if len(args) == 2 {
		serviceName, planName = args[0], args[1]
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
		spinner = sm.AddSpinner("Listing service plan versions...")
		sm.Start()
	}

	// Check if the service plan exists
	_, _, planID, _, err = getServicePlan(cmd.Context(), token, serviceID, serviceName, planID, planName, environment)
	if err != nil {
		utils.HandleSpinnerError(spinner, sm, err)
		return err
	}

	// Search service plans versions
	searchRes, err := dataaccess.SearchInventory(cmd.Context(), token, fmt.Sprintf("serviceplan:%s", planID))
	if err != nil {
		utils.PrintError(err)
		return err
	}

	// Filter out the latest N versions if latestN flag is provided
	latestNServicePlanVersions := filterLatestNVersions(searchRes.ServicePlanResults, latestN)

	var formattedServicePlanVersions []model.ServicePlanVersion

	// Process and filter service plans
	for _, servicePlanVersion := range latestNServicePlanVersions {
		formattedServicePlanVersion := formatServicePlanVersion(servicePlanVersion, truncateNames)

		match, err := utils.MatchesFilters(formattedServicePlanVersion, filterMaps)
		if err != nil {
			utils.HandleSpinnerError(spinner, sm, err)
			return err
		}

		if match {
			formattedServicePlanVersions = append(formattedServicePlanVersions, formattedServicePlanVersion)
		}
	}

	// Handle case when no service plans match
	if len(formattedServicePlanVersions) == 0 {
		utils.HandleSpinnerSuccess(spinner, sm, "No service plan versions found.")
		return nil
	}

	utils.HandleSpinnerSuccess(spinner, sm, "Service plan versions retrieved successfully")

	// Format output as requested
	err = utils.PrintTextTableJsonArrayOutput(output, formattedServicePlanVersions)
	if err != nil {
		return err
	}

	return nil
}

// Helper functions

func formatServicePlanVersion(servicePlan openapiclientfleet.ServicePlanSearchRecord, truncateNames bool) model.ServicePlanVersion {
	serviceName := servicePlan.ServiceName
	envName := servicePlan.ServiceEnvironmentName
	planName := servicePlan.Name

	if truncateNames {
		serviceName = utils.TruncateString(serviceName, defaultMaxNameLength)
		envName = utils.TruncateString(envName, defaultMaxNameLength)
		planName = utils.TruncateString(planName, defaultMaxNameLength)
	}

	var releaseDescription string
	if servicePlan.VersionName != nil {
		releaseDescription = *servicePlan.VersionName
	}

	return model.ServicePlanVersion{
		PlanID:             servicePlan.Id,
		PlanName:           planName,
		ServiceID:          servicePlan.ServiceId,
		ServiceName:        serviceName,
		Environment:        envName,
		Version:            servicePlan.Version,
		ReleaseDescription: releaseDescription,
		VersionSetStatus:   servicePlan.VersionSetStatus,
	}
}

func validateListVersionsArguments(args []string, serviceID, planID string) error {
	if len(args) == 0 && (serviceID == "" || planID == "") {
		return fmt.Errorf("please provide the service name and service plan name or the service ID and service plan ID")
	}
	if len(args) > 0 && len(args) != 2 {
		return fmt.Errorf("invalid arguments: %s. Need 2 arguments: [service-name] [plan-name]", strings.Join(args, " "))
	}
	return nil
}

func filterLatestNVersions(servicePlans []openapiclientfleet.ServicePlanSearchRecord, latestN int) []openapiclientfleet.ServicePlanSearchRecord {
	slices.SortFunc(servicePlans, func(a, b openapiclientfleet.ServicePlanSearchRecord) int {
		if a.ReleasedAt != nil && b.ReleasedAt != nil {
			ta, _ := time.Parse(time.RFC3339, *a.ReleasedAt)
			tb, _ := time.Parse(time.RFC3339, *b.ReleasedAt)
			if ta.After(tb) {
				return -1
			} else if ta.Before(tb) {
				return 1
			} else {
				return 0
			}
		} else if a.ReleasedAt == nil && b.ReleasedAt != nil {
			return 1
		} else if a.ReleasedAt != nil && b.ReleasedAt == nil {
			return -1
		}

		return 0
	})

	if latestN == -1 {
		return servicePlans
	}

	if len(servicePlans) <= latestN {
		return servicePlans
	}
	return servicePlans[:latestN]
}
