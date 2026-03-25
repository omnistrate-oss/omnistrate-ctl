package environment

import (
	"encoding/json"
	"fmt"
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
	listExample = `# List environments of the service postgres in the prod and dev environment types
omnistrate-ctl environment list -f="service_name:postgres,environment_type:PROD" -f="service:postgres,environment_type:DEV"`
	defaultMaxNameLength = 30 // Maximum length of the name column in the table
)

var listCmd = &cobra.Command{
	Use:   "list [flags]",
	Short: "List environments for your service",
	Long: `This command helps you list environments for your service.
You can filter for specific environments by using the filter flag.`,
	Example:      listExample,
	RunE:         runList,
	SilenceUsage: true,
}

func init() {
	listCmd.Flags().StringArrayP("filter", "f", []string{}, "Filter to apply to the list of environments. E.g.: key1:value1,key2:value2, which filters environments where key1 equals value1 and key2 equals value2. Allow use of multiple filters to form the logical OR operation. Supported keys: "+strings.Join(utils.GetSupportedFilterKeys(model.Environment{}), ",")+". Check the examples for more details.")
	listCmd.Flags().Bool("truncate", false, "Truncate long names in the output")
	listCmd.Flags().BoolP("interactive", "i", false, "Launch interactive list with fuzzy search and selection")
}

func runList(cmd *cobra.Command, args []string) error {
	defer config.CleanupArgsAndFlags(cmd, &args)

	// Retrieve command-line flags
	output, _ := cmd.Flags().GetString("output")
	filters, _ := cmd.Flags().GetStringArray("filter")
	truncateNames, _ := cmd.Flags().GetBool("truncate")
	interactive, _ := cmd.Flags().GetBool("interactive")

	// Parse and validate filters
	filterMaps, err := utils.ParseFilters(filters, utils.GetSupportedFilterKeys(model.Environment{}))
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
	if output != "json" && !interactive {
		sm = utils.NewSpinnerManager()
		spinner = sm.AddSpinner("Retrieving environments...")
		sm.Start()
	}

	// Retrieve services and environments
	services, err := dataaccess.ListServices(cmd.Context(), token)
	if err != nil {
		utils.HandleSpinnerError(spinner, sm, err)
		return err
	}

	formattedEnvironments := make([]model.Environment, 0)

	// Process and filter environments
	for _, service := range services.Services {
		for _, environment := range service.ServiceEnvironments {
			if environment.Name == "" {
				continue

			}
			env := formatEnvironment(service, environment, truncateNames)

			match, err := utils.MatchesFilters(env, filterMaps)
			if err != nil {
				utils.HandleSpinnerError(spinner, sm, err)
				return err
			}

			if match {
				formattedEnvironments = append(formattedEnvironments, env)
			}
		}
	}

	// Handle case when no environments match
	if len(formattedEnvironments) == 0 {
		utils.HandleSpinnerSuccess(spinner, sm, "No environments found")
	} else {
		utils.HandleSpinnerSuccess(spinner, sm, "Successfully retrieved environments")
	}

	// Interactive mode
	if interactive {
		return runInteractiveEnvironmentList(formattedEnvironments)
	}

	// Format output as requested
	err = utils.PrintTextTableJsonArrayOutput(output, formattedEnvironments)
	if err != nil {
		return err
	}

	return nil
}

func runInteractiveEnvironmentList(envs []model.Environment) error {
	items := make([]utils.InteractiveListItem, len(envs))
	for i, env := range envs {
		rawJSON, _ := json.Marshal(env)
		items[i] = utils.NewInteractiveListItem(
			env.EnvironmentName,
			fmt.Sprintf("%s · %s · %s · %s", env.EnvironmentID, env.EnvironmentType, env.ServiceName, env.ServiceID),
			env.EnvironmentID,
			"",
			rawJSON,
		)
	}

	selected, err := utils.RunInteractiveList(utils.InteractiveListConfig{
		Title:    "Environments",
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

func formatEnvironment(service openapiclientv1.DescribeServiceResult, environment openapiclientv1.ServiceEnvironment, truncateNames bool) model.Environment {
	serviceName := service.Name
	envName := environment.Name

	if truncateNames {
		serviceName = utils.TruncateString(serviceName, defaultMaxNameLength)
		envName = utils.TruncateString(envName, defaultMaxNameLength)
	}

	envType := ""
	if environment.Type != nil {
		envType = *environment.Type
	}

	sourceEnvName := ""
	if environment.SourceEnvironmentName != nil {
		sourceEnvName = *environment.SourceEnvironmentName
	}

	return model.Environment{
		EnvironmentID:   environment.Id,
		EnvironmentName: envName,
		EnvironmentType: envType,
		ServiceID:       service.Id,
		ServiceName:     serviceName,
		SourceEnvName:   sourceEnvName,
	}
}
