package environment

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/omnistrate-oss/omnistrate-ctl/cmd/common"
	openapiclientv1 "github.com/omnistrate-oss/omnistrate-sdk-go/v1"

	"github.com/chelnak/ysmrr"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/config"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/dataaccess"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/utils"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

const (
	createExample = `# Create environment
omnistrate-ctl environment create [service-name] [environment-name] --type=[type] --source=[source]

# Create environment by ID instead of name
omnistrate-ctl environment create [environment-name] --service-id=[service-id] --type=[type] --source=[source]`
)

var EnvironmentID string

var createCmd = &cobra.Command{
	Use:          "create [service-name] [environment-name] [flags]",
	Short:        "Create a Service Environment",
	Long:         `This command helps you create a new environment for your service.`,
	Example:      createExample,
	RunE:         runCreate,
	SilenceUsage: true,
}

func init() {
	createCmd.Flags().StringP("description", "", "", "Environment description")
	createCmd.Flags().StringP("type", "", "", "Type of environment. Valid options include: 'dev', 'prod', 'qa', 'canary', 'staging', 'private'")
	createCmd.Flags().StringP("source", "", "", "Source environment name")
	createCmd.Flags().StringP("service-id", "", "", "Service ID. Required if service name is not provided")

	err := createCmd.MarkFlagRequired("type")
	if err != nil {
		return
	}

	createCmd.Args = cobra.MinimumNArgs(1)
	createCmd.Args = cobra.MaximumNArgs(2)
}

func runCreate(cmd *cobra.Command, args []string) error {
	defer config.CleanupArgsAndFlags(cmd, &args)

	// Get flags
	description, _ := cmd.Flags().GetString("description")
	envType, _ := cmd.Flags().GetString("type")
	sourceEnvName, _ := cmd.Flags().GetString("source")
	output, _ := cmd.Flags().GetString("output")
	serviceID, _ := cmd.Flags().GetString("service-id")
	envName := args[len(args)-1]

	// Set service name if provided in args
	var serviceName string
	if len(args) == 2 {
		serviceName = args[0]
	}

	// Validate input arguments
	if err := validateCreateArguments(args, serviceID); err != nil {
		utils.PrintError(err)
		return err
	}

	// Validate environment type
	if err := ValidateEnvironmentType(envType); err != nil {
		utils.PrintError(err)
		return err
	}

	// Validate user is logged in
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
		spinner = sm.AddSpinner("Creating environment...")
		sm.Start()
	}

	// Check if the service exists
	serviceID, serviceName, err = getService(cmd.Context(), token, serviceID, serviceName)
	if err != nil {
		utils.HandleSpinnerError(spinner, sm, err)
		return err
	}

	// Check if source environment exists
	var sourceEnvID *string
	if strings.TrimSpace(sourceEnvName) != "" {
		sourceEnvID, err = getSourceEnvironmentID(cmd.Context(), token, serviceID, sourceEnvName)
		if err != nil {
			utils.HandleSpinnerError(spinner, sm, err)
			return err
		}
	}

	// Create the environment
	if description == "" {
		description = fmt.Sprintf("%s environment for service %s", envType, serviceName)
	}

	visibility := getVisibility(envType)
	defaultDeploymentConfigID, err := dataaccess.GetDefaultDeploymentConfigID(cmd.Context(), token)
	if err != nil {
		utils.HandleSpinnerError(spinner, sm, err)
		return err
	}

	publicKeyPtr := getPublicKeyPtr(visibility)
	environmentID, err := dataaccess.CreateServiceEnvironment(
		cmd.Context(), token,
		envName,
		description,
		serviceID,
		visibility,
		envType,
		sourceEnvID,
		defaultDeploymentConfigID,
		true,
		publicKeyPtr)
	if err != nil {
		utils.HandleSpinnerError(spinner, sm, err)
		return err
	}

	utils.HandleSpinnerSuccess(spinner, sm, "Successfully created environment")

	// Describe the environment
	environment, err := dataaccess.DescribeServiceEnvironment(cmd.Context(), token, serviceID, environmentID)
	if err != nil {
		utils.PrintError(err)
		return err
	}

	// Format and print the environment
	formattedEnvironment := formatEnvironmentDetails(cmd.Context(), token, serviceID, serviceName, sourceEnvName, environment)

	err = utils.PrintTextTableJsonOutput(output, formattedEnvironment)
	if err != nil {
		utils.PrintError(err)
		return err
	}

	EnvironmentID = environmentID

	return nil
}

// Helper functions

func validateCreateArguments(args []string, serviceID string) error {
	if len(args) == 1 && serviceID == "" {
		return fmt.Errorf("please provide service name or id")
	}
	if len(args) == 2 && serviceID != "" {
		return fmt.Errorf("please provide either service name or id, not both")
	}
	return nil
}

func ValidateEnvironmentType(envType string) error {
	if !slices.Contains([]string{"dev", "qa", "staging", "canary", "prod", "private"}, strings.ToLower(envType)) {
		return fmt.Errorf("invalid environment type: %s", envType)
	}
	return nil
}

func getService(ctx context.Context, token, serviceIDArg, serviceNameArg string) (serviceID string, serviceName string, err error) {
	searchRes, err := dataaccess.SearchInventory(ctx, token, "service:s")
	if err != nil {
		return "", "", err
	}

	for _, service := range searchRes.ServiceResults {
		if serviceIDArg == service.Id || strings.EqualFold(service.Name, serviceNameArg) {
			return service.Id, service.Name, nil
		}
	}

	return "", "", errors.New("service not found")
}

func getSourceEnvironmentID(ctx context.Context, token, serviceID, sourceEnvName string) (*string, error) {
	if sourceEnvName == "" {
		return nil, nil
	}

	describeServiceRes, err := dataaccess.DescribeService(ctx, token, serviceID)
	if err != nil {
		return nil, err
	}

	for _, env := range describeServiceRes.ServiceEnvironments {
		if strings.EqualFold(env.Name, sourceEnvName) {
			return utils.ToPtr(env.Id), nil
		}
	}

	return nil, errors.New("source environment not found. Please provide a valid source environment name")
}

func getVisibility(envType string) string {
	switch strings.ToLower(envType) {
	case "dev", "qa", "staging", "canary", "private":
		return "PRIVATE"
	case "prod":
		return "PUBLIC"
	default:
		return ""
	}
}

func getPublicKeyPtr(visibility string) *string {
	if visibility == "PRIVATE" {
		return utils.ToPtr(config.GetDefaultServiceAuthPublicKey())
	}
	return nil
}

func getPromoteStatus(ctx context.Context, token, serviceID string, environment *openapiclientv1.DescribeServiceEnvironmentResult) string {
	if !utils.CheckIfNilOrEmpty(environment.SourceEnvironmentId) {
		promoteRes, err := dataaccess.PromoteServiceEnvironmentStatus(ctx, token, serviceID, *environment.SourceEnvironmentId)
		if err == nil {
			for _, res := range promoteRes {
				if res.TargetEnvironmentID == environment.Id {
					return res.Status
				}
			}
		}
	}
	return ""
}
