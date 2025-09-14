package serviceplan

import (
	"context"
	"errors"
	"fmt"
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
	describeExample = `# Describe service plan
omctl service-plan describe [service-name] [plan-name]

# Describe service plan by ID instead of name
omctl service-plan describe --service-id [service-id] --plan-id [plan-id]`
)

var describeCmd = &cobra.Command{
	Use:          "describe [service-name] [plan-name] [flags]",
	Short:        "Describe a Service Plan",
	Long:         `This command helps you get details of a Service Plan for your service.`,
	Example:      describeExample,
	RunE:         runDescribe,
	SilenceUsage: true,
}

func init() {
	describeCmd.Flags().StringP("environment", "", "", "Environment name. Use this flag with service name and plan name to describe the service plan in a specific environment")
	describeCmd.Flags().StringP("output", "o", "json", "Output format. Only json is supported")
	describeCmd.Flags().StringP("service-id", "", "", "Service ID. Required if service name is not provided")
	describeCmd.Flags().StringP("plan-id", "", "", "Environment ID. Required if plan name is not provided")
}

func runDescribe(cmd *cobra.Command, args []string) error {
	defer config.CleanupArgsAndFlags(cmd, &args)

	// Retrieve flags
	serviceID, _ := cmd.Flags().GetString("service-id")
	planID, _ := cmd.Flags().GetString("plan-id")
	output, _ := cmd.Flags().GetString("output")
	environment, _ := cmd.Flags().GetString("environment")

	// Validate input arguments
	if err := validateDescribeArguments(args, serviceID, planID, output); err != nil {
		utils.PrintError(err)
		return err
	}

	// Set service and service plan names if provided in args
	var serviceName, planName string
	if len(args) == 2 {
		serviceName, planName = args[0], args[1]
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
		spinner = sm.AddSpinner("Describing service plan...")
		sm.Start()
	}

	// Check if the service plan exists
	serviceID, serviceName, planID, _, environment, err = getServicePlan(cmd.Context(), token, serviceID, serviceName, planID, planName, environment)
	if err != nil {
		utils.HandleSpinnerError(spinner, sm, err)
		return err
	}

	// Describe the service plan
	servicePlan, err := dataaccess.DescribeProductTier(cmd.Context(), token, serviceID, planID)
	if err != nil {
		utils.HandleSpinnerError(spinner, sm, err)
		return err
	}

	// Format the service plan details
	formattedServicePlan, err := formatServicePlanDetails(cmd.Context(), token, serviceName, planName, environment, servicePlan)
	if err != nil {
		utils.HandleSpinnerError(spinner, sm, err)
		return err
	}

	// Handle output based on format
	utils.HandleSpinnerSuccess(spinner, sm, "Service plan details retrieved successfully")

	if err = utils.PrintTextTableJsonOutput(output, formattedServicePlan); err != nil {
		return err
	}

	return nil
}

// Helper functions

func validateDescribeArguments(args []string, serviceID, planID, output string) error {
	if len(args) == 0 && (serviceID == "" || planID == "") {
		return fmt.Errorf("please provide the service name and service plan name or the service ID and service plan ID")
	}
	if len(args) > 0 && len(args) != 2 {
		return fmt.Errorf("invalid arguments: %s. Need 2 arguments: [service-name] [plan-name]", strings.Join(args, " "))
	}
	if output != "json" {
		return errors.New("only json output is supported")
	}
	return nil
}

func formatServicePlanDetails(ctx context.Context, token, serviceName, planName, environment string, productTier *openapiclient.DescribeProductTierResult) (model.ServicePlanDetails, error) {
	// Get service model
	serviceModel, err := dataaccess.DescribeServiceModel(ctx, token, productTier.ServiceId, productTier.ServiceModelId)
	if err != nil {
		return model.ServicePlanDetails{}, err
	}

	if productTier.ApiGroups == nil || len(*productTier.ApiGroups) == 0 {
		return model.ServicePlanDetails{}, fmt.Errorf("no resources found in the service plan")
	}

	// Get resource details
	var resources []model.Resource
	for resourceID := range *productTier.ApiGroups {
		// Get resource details
		desRes, err := dataaccess.DescribeResource(ctx, token, productTier.ServiceId, resourceID, nil, nil)
		if err != nil {
			return model.ServicePlanDetails{}, err
		}
		resource := model.Resource{
			ResourceID:          desRes.Id,
			ResourceName:        desRes.Name,
			ResourceDescription: desRes.Description,
			ResourceType:        desRes.ResourceType,
		}

		if desRes.ActionHooks != nil {
			resource.ActionHooks = desRes.ActionHooks
		}
		if desRes.AdditionalSecurityContext != nil {
			resource.AdditionalSecurityContext = desRes.AdditionalSecurityContext
		}
		if desRes.BackupConfiguration != nil {
			resource.BackupConfiguration = desRes.BackupConfiguration
		}
		if desRes.Capabilities != nil {
			resource.Capabilities = desRes.Capabilities
		}
		if desRes.CustomLabels != nil {
			resource.CustomLabels = desRes.CustomLabels
		}
		if desRes.CustomSysCTLs != nil {
			resource.CustomSysCTLs = desRes.CustomSysCTLs
		}
		if desRes.CustomULimits != nil {
			resource.CustomULimits = desRes.CustomULimits
		}
		if desRes.Dependencies != nil {
			resource.Dependencies = desRes.Dependencies
		}
		if desRes.EnvironmentVariables != nil {
			resource.EnvironmentVariables = desRes.EnvironmentVariables
		}
		if desRes.FileSystemConfiguration != nil {
			resource.FileSystemConfiguration = desRes.FileSystemConfiguration
		}
		if desRes.HelmChartConfiguration != nil {
			resource.HelmChartConfiguration = desRes.HelmChartConfiguration
		}
		if desRes.KustomizeConfiguration != nil {
			resource.KustomizeConfiguration = desRes.KustomizeConfiguration
		}
		if desRes.L4LoadBalancerConfiguration != nil {
			resource.L4LoadBalancerConfiguration = desRes.L4LoadBalancerConfiguration
		}
		if desRes.L7LoadBalancerConfiguration != nil {
			resource.L7LoadBalancerConfiguration = desRes.L7LoadBalancerConfiguration
		}
		if desRes.OperatorCRDConfiguration != nil {
			resource.OperatorCRDConfiguration = desRes.OperatorCRDConfiguration
		}

		resources = append(resources, resource)
	}

	// Describe pending changes
	pendingChanges, err := dataaccess.DescribePendingChanges(ctx, token, productTier.ServiceId, serviceModel.ServiceApiId, productTier.Id)
	if err != nil {
		return model.ServicePlanDetails{}, err
	}

	formattedPendingChanges := make(map[string]model.ResourceChangeSet)
	for resourceID, cs := range pendingChanges.ResourceChangeSets {
		formattedChangeSet := model.ResourceChangeSet{
			ResourceChanges:           cs.ResourceChanges,
			ProductTierFeatureChanges: cs.ProductTierFeatureChanges,
			ImageConfigChanges:        cs.ImageConfigChanges,
			InfraConfigChanges:        cs.InfraConfigChanges,
		}
		if cs.ResourceName != nil {
			formattedChangeSet.ResourceName = *cs.ResourceName
		}
		formattedPendingChanges[resourceID] = formattedChangeSet
	}

	formattedServicePlan := model.ServicePlanDetails{
		PlanID:          productTier.Id,
		PlanName:        planName,
		ServiceID:       productTier.ServiceId,
		ServiceName:     serviceName,
		Environment:     environment,
		DeploymentType:  productTier.TierType,
		TenancyType:     serviceModel.ModelType,
		EnabledFeatures: productTier.EnabledFeatures,
		Resources:       resources,
		PendingChanges:  formattedPendingChanges,
	}

	return formattedServicePlan, nil
}
