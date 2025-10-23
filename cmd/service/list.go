package service

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
	listExample = `# List services
omctl service list`
	defaultMaxNameLength = 30 // Maximum length of the name column in the table
)

var listCmd = &cobra.Command{
	Use:   "list [flags]",
	Short: "List services for your account",
	Long: `This command helps you list services for your account.
You can filter for specific services by using the filter flag.`,
	Example:      listExample,
	RunE:         runList,
	SilenceUsage: true,
}

func init() {
	listCmd.Flags().StringArrayP("filter", "f", []string{}, "Filter to apply to the list of services. E.g.: key1:value1,key2:value2, which filters services where key1 equals value1 and key2 equals value2. Allow use of multiple filters to form the logical OR operation. Supported keys: "+strings.Join(utils.GetSupportedFilterKeys(model.Service{}), ",")+". Check the examples for more details.")
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
	filterMaps, err := utils.ParseFilters(filters, utils.GetSupportedFilterKeys(model.Service{}))
	if err != nil {
		utils.PrintError(err)
		return err
	}

	// Validate user login
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
		msg := "Listing services..."
		spinner = sm.AddSpinner(msg)
		sm.Start()
	}

	// Retrieve services and services
	listRes, err := dataaccess.ListServices(cmd.Context(), token)
	if err != nil {
		utils.HandleSpinnerError(spinner, sm, err)
		return err
	}

	var formattedServices []model.Service

	// Process and filter services
	for _, service := range listRes.Services {
		formattedService := formatService(service, truncateNames)

		match, err := utils.MatchesFilters(formattedService, filterMaps)
		if err != nil {
			utils.HandleSpinnerError(spinner, sm, err)
			return err
		}

		if match {
			formattedServices = append(formattedServices, formattedService)
		}
	}

	// Handle case when no services match
	if len(formattedServices) == 0 {
		utils.HandleSpinnerSuccess(spinner, sm, "No services found")
	} else {
		utils.HandleSpinnerSuccess(spinner, sm, "Successfully retrieved services")
	}

	// Format output as requested
	err = utils.PrintTextTableJsonArrayOutput(output, formattedServices)
	if err != nil {
		utils.PrintError(err)
		return err
	}

	return nil
}

// Helper functions

func formatService(service openapiclient.DescribeServiceResult, truncateNames bool) model.Service {
	// Retrieve environments
	environments := make([]string, 0)
	for _, environment := range service.ServiceEnvironments {
		if environment.Name == "" {
			continue

		}
		environments = append(environments, environment.Name)
	}

	// Truncate service name if requested
	serviceName := service.Name
	if truncateNames {
		serviceName = utils.TruncateString(serviceName, defaultMaxNameLength)
	}

	return model.Service{
		ID:           service.Id,
		Name:         serviceName,
		Environments: strings.Join(environments, ","),
	}
}
