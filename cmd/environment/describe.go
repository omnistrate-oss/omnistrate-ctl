package environment

import (
	"context"
	"fmt"
	"strings"

	"github.com/omnistrate-oss/omnistrate-ctl/cmd/common"

	"github.com/chelnak/ysmrr"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/config"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/dataaccess"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/model"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/utils"
	openapiclientv1 "github.com/omnistrate-oss/omnistrate-sdk-go/v1"
	"github.com/spf13/cobra"
)

const (
	describeExample = `# Describe environment
omnistrate-ctl environment describe [service-name] [environment-name]

# Describe environment by ID instead of name
omnistrate-ctl environment describe --service-id=[service-id] --environment-id=[environment-id]`
)

var describeCmd = &cobra.Command{
	Use:          "describe [service-name] [environment-name] [flags]",
	Short:        "Describe a Service Environment",
	Long:         `This command helps you get details of a service environment from your service. You can find details like SaaS portal status, SaaS portal URL, and promote status, etc.`,
	Example:      describeExample,
	RunE:         runDescribe,
	SilenceUsage: true,
}

func init() {
	describeCmd.Flags().StringP("service-id", "", "", "Service ID. Required if service name is not provided")
	describeCmd.Flags().StringP("environment-id", "", "", "Environment ID. Required if environment name is not provided")
	describeCmd.Flags().StringP("output", "o", "json", "Output format. Only json is supported.") // Override inherited flag
}

func runDescribe(cmd *cobra.Command, args []string) error {
	defer config.CleanupArgsAndFlags(cmd, &args)

	// Retrieve flags
	serviceID, _ := cmd.Flags().GetString("service-id")
	environmentID, _ := cmd.Flags().GetString("environment-id")
	output, _ := cmd.Flags().GetString("output")

	// Validate input arguments
	if err := validateDescribeArguments(args, serviceID, environmentID, output); err != nil {
		utils.PrintError(err)
		return err
	}

	// Set service and environment names if provided in args
	var serviceName, environmentName string
	if len(args) == 2 {
		serviceName, environmentName = args[0], args[1]
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
		spinner = sm.AddSpinner("Describing environment...")
		sm.Start()
	}

	// Retrieve service and environment details
	serviceID, serviceName, environmentID, err = getServiceEnvironment(cmd.Context(), token, serviceID, serviceName, environmentID, environmentName)
	if err != nil {
		utils.HandleSpinnerError(spinner, sm, err)
		return err
	}

	// Describe the environment
	environment, err := dataaccess.DescribeServiceEnvironment(cmd.Context(), token, serviceID, environmentID)
	if err != nil {
		utils.HandleSpinnerError(spinner, sm, err)
		return err
	}

	// Get the source environment name
	sourceEnvName := ""
	if environment.SourceEnvironmentId != nil {
		sourceEnv, err := dataaccess.DescribeServiceEnvironment(cmd.Context(), token, serviceID, *environment.SourceEnvironmentId)
		if err != nil {
			utils.HandleSpinnerError(spinner, sm, err)
			return err
		}
		sourceEnvName = sourceEnv.Name
	}

	// Format the environment details
	formattedEnvironment := formatEnvironmentDetails(cmd.Context(), token, serviceID, serviceName, sourceEnvName, environment)

	// Handle output based on format
	if spinner != nil {
		spinner.UpdateMessage("Successfully retrieved environment details")
		spinner.Complete()
		sm.Stop()
	}

	if err = utils.PrintTextTableJsonOutput(output, formattedEnvironment); err != nil {
		return err
	}

	return nil
}

// Helper functions

func validateDescribeArguments(args []string, serviceID, environmentID, output string) error {
	if len(args) == 0 && (serviceID == "" || environmentID == "") {
		return fmt.Errorf("please provide the service name and environment name or the service ID and environment ID")
	}
	if len(args) > 0 && len(args) != 2 {
		return fmt.Errorf("invalid arguments: %s. Need 2 arguments: [service-name] [environment-name]", strings.Join(args, " "))
	}
	if output != "json" {
		return fmt.Errorf("only json output is supported")
	}
	return nil
}

func formatEnvironmentDetails(ctx context.Context, token, serviceID, serviceName, sourceEnvName string, environment *openapiclientv1.DescribeServiceEnvironmentResult) model.DetailedEnvironment {
	formattedEnvironment := model.DetailedEnvironment{
		EnvironmentID:    environment.Id,
		EnvironmentName:  environment.Name,
		EnvironmentType:  environment.Type,
		ServiceID:        environment.ServiceId,
		SourceEnvName:    sourceEnvName,
		ServiceName:      serviceName,
		SaaSPortalStatus: getSaaSPortalStatus(environment),
		SaaSPortalURL:    getSaaSPortalURL(environment),
		PromoteStatus:    getPromoteStatus(ctx, token, serviceID, environment),
	}

	return formattedEnvironment
}

func getSaaSPortalStatus(environment *openapiclientv1.DescribeServiceEnvironmentResult) string {
	if environment.SaasPortalStatus != nil {
		return *environment.SaasPortalStatus
	}
	return ""
}

func getSaaSPortalURL(environment *openapiclientv1.DescribeServiceEnvironmentResult) string {
	if environment.SaasPortalUrl != nil {
		return *environment.SaasPortalUrl
	}
	return ""
}
