package instance

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/omnistrate-oss/omnistrate-ctl/cmd/common"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/config"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/dataaccess"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/model"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/utils"
	openapiclientfleet "github.com/omnistrate-oss/omnistrate-sdk-go/fleet"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

const (
	createExample = `# Create an instance deployment
omnistrate-ctl instance create --service=mysql --environment=dev --plan=mysql --version=latest --resource=mySQL --cloud-provider=aws --region=ca-central-1 --param '{"databaseName":"default","password":"a_secure_password","rootPassword":"a_secure_root_password","username":"user"}'

# Create an instance deployment with parameters from a file
omnistrate-ctl instance create --service=mysql --environment=dev --plan=mysql --version=latest --resource=mySQL --cloud-provider=aws --region=ca-central-1 --param-file /path/to/params.json

# Create an instance deployment with custom tags
omnistrate-ctl instance create --service=mysql --environment=dev --plan=mysql --version=latest --resource=mySQL --cloud-provider=aws --region=ca-central-1 --param-file /path/to/params.json --tags environment=dev,owner=team

# Create an instance deployment and wait for completion with progress tracking
omnistrate-ctl instance create --service=mysql --environment=dev --plan=mysql --version=latest --resource=mySQL --cloud-provider=aws --region=ca-central-1 --param-file /path/to/params.json --wait

# Create an instance deployment with workflow breakpoints
omnistrate-ctl instance create --service=mysql --environment=dev --plan=mysql --version=latest --resource=mySQL --cloud-provider=aws --region=ca-central-1 --param-file /path/to/params.json --breakpoints writer,reader

# Create a BYOA instance deployment using a customer account onboarding instance
omnistrate-ctl instance create --service=Nebius --environment=dev --plan='Nebius BYOA Compute Variants' --resource=NebiusRedis --cloud-provider=nebius --region=eu-north1 --customer-account-id instance-cg1tthkj0`

	customerAccountConfigIDParamKey = "cloud_provider_account_config_id"
	serviceModelTypeBYOA            = "BYOA"
)

var InstanceID string

var createCmd = &cobra.Command{
	Use:          "create --service=[service] --environment=[environment] --plan=[plan] --version=[version] --resource=[resource] --cloud-provider=[aws|gcp|azure|nebius] --region=[region] [--param=param] [--param-file=file-path] [--customer-account-id=instance-id] [--tags key=value,key2=value2] [--breakpoints id-or-key,id-or-key]",
	Short:        "Create an instance deployment",
	Long:         `This command helps you create an instance deployment for your service.`,
	Example:      createExample,
	RunE:         runCreate,
	SilenceUsage: true,
}

func init() {
	createCmd.Flags().String("service", "", "Service name")
	createCmd.Flags().String("environment", "", "Environment name")
	createCmd.Flags().String("plan", "", "Service plan name")
	createCmd.Flags().String("version", "preferred", "Service plan version (latest|preferred|1.0 etc.)")
	createCmd.Flags().String("resource", "", "Resource name")
	createCmd.Flags().String("cloud-provider", "", "Cloud provider (aws|gcp|azure|nebius)")
	createCmd.Flags().String("region", "", "Region code (e.g. us-east-2, us-central1)")
	createCmd.Flags().String("param", "", "Parameters for the instance deployment")
	createCmd.Flags().String("param-file", "", "Json file containing parameters for the instance deployment")
	createCmd.Flags().String("customer-account-id", "", "Customer BYOA account onboarding instance ID to inject as the cloud account. Use 'omnistrate-ctl account customer list' or 'omnistrate-ctl account customer describe <instance-id>' to find it.")
	createCmd.Flags().String("tags", "", "Custom tags to add to the instance deployment (format: key=value,key2=value2)")
	createCmd.Flags().String("breakpoints", "", "Workflow breakpoint resource IDs or resource keys (comma-separated)")
	createCmd.Flags().StringP("subscription-id", "", "", "Subscription ID to use for the instance deployment. If not provided, instance deployment will be created in your own subscription.")
	createCmd.Flags().Bool("wait", false, "Wait for deployment to complete and show progress")

	if err := createCmd.MarkFlagRequired("service"); err != nil {
		return
	}
	if err := createCmd.MarkFlagRequired("environment"); err != nil {
		return
	}
	if err := createCmd.MarkFlagRequired("plan"); err != nil {
		return
	}
	if err := createCmd.MarkFlagRequired("resource"); err != nil {
		return
	}
	if err := createCmd.MarkFlagRequired("cloud-provider"); err != nil {
		return
	}
	if err := createCmd.MarkFlagRequired("region"); err != nil {
		return
	}
	if err := createCmd.MarkFlagFilename("param-file"); err != nil {
		return
	}

	createCmd.Args = cobra.NoArgs // Require no arguments
}

func runCreate(cmd *cobra.Command, args []string) error {
	defer config.CleanupArgsAndFlags(cmd, &args)

	// Retrieve flags
	service, err := cmd.Flags().GetString("service")
	if err != nil {
		utils.PrintError(err)
		return err
	}
	environment, err := cmd.Flags().GetString("environment")
	if err != nil {
		utils.PrintError(err)
		return err
	}
	plan, err := cmd.Flags().GetString("plan")
	if err != nil {
		utils.PrintError(err)
		return err
	}
	version, err := cmd.Flags().GetString("version")
	if err != nil {
		utils.PrintError(err)
		return err
	}
	version = strings.Trim(version, "\"") // Remove quotes
	resource, err := cmd.Flags().GetString("resource")
	if err != nil {
		utils.PrintError(err)
		return err
	}
	cloudProvider, err := cmd.Flags().GetString("cloud-provider")
	if err != nil {
		utils.PrintError(err)
		return err
	}
	region, err := cmd.Flags().GetString("region")
	if err != nil {
		utils.PrintError(err)
		return err
	}
	param, err := cmd.Flags().GetString("param")
	if err != nil {
		utils.PrintError(err)
		return err
	}
	paramFile, err := cmd.Flags().GetString("param-file")
	if err != nil {
		utils.PrintError(err)
		return err
	}
	customerAccountID, err := cmd.Flags().GetString("customer-account-id")
	if err != nil {
		utils.PrintError(err)
		return err
	}
	subscriptionID, err := cmd.Flags().GetString("subscription-id")
	if err != nil {
		utils.PrintError(err)
		return err
	}
	waitFlag, err := cmd.Flags().GetBool("wait")
	if err != nil {
		utils.PrintError(err)
		return err
	}
	output, err := cmd.Flags().GetString("output")
	if err != nil {
		utils.PrintError(err)
		return err
	}
	customTags, tagsProvided, err := parseCustomTags(cmd)
	if err != nil {
		utils.PrintError(err)
		return err
	}
	breakpointsRaw, err := cmd.Flags().GetString("breakpoints")
	if err != nil {
		utils.PrintError(err)
		return err
	}
	workflowBreakpoints, err := parseWorkflowBreakpoints(breakpointsRaw)
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
	var sm utils.SpinnerManager
	var spinner *utils.Spinner
	if output != "json" {
		sm = utils.NewSpinnerManager()
		msg := "Creating instance..."
		spinner = sm.AddSpinner(msg)
		sm.Start()
	}

	// Check if resource exists
	serviceID, _, productTierID, _, err := getResource(cmd.Context(), token, service, environment, plan, resource)
	if err != nil {
		utils.HandleSpinnerError(spinner, sm, err)
		return err
	}

	// Get the version
	switch version {
	case "latest":
		version, err = dataaccess.FindLatestVersion(cmd.Context(), token, serviceID, productTierID)
		if err != nil {
			utils.HandleSpinnerError(spinner, sm, err)
			return err
		}
	case "preferred":
		version, err = dataaccess.FindPreferredVersion(cmd.Context(), token, serviceID, productTierID)
		if err != nil {
			utils.HandleSpinnerError(spinner, sm, err)
			return err
		}
	}

	// Check if the version exists
	_, err = dataaccess.DescribeVersionSet(cmd.Context(), token, serviceID, productTierID, version)
	if err != nil {
		if strings.Contains(err.Error(), "Version set not found") {
			err = errors.New(fmt.Sprintf("version %s not found", version))
		}
		utils.HandleSpinnerError(spinner, sm, err)
		return err
	}

	// Describe service offering
	res, err := dataaccess.DescribeServiceOffering(cmd.Context(), token, serviceID, productTierID, version)
	if err != nil {
		utils.HandleSpinnerError(spinner, sm, err)
		return err
	}
	offering := res.ConsumptionDescribeServiceOfferingResult.Offerings[0]

	// Format parameters
	formattedParams, err := common.FormatParams(param, paramFile)
	if err != nil {
		utils.HandleSpinnerError(spinner, sm, err)
		return err
	}
	formattedParams, err = applyCustomerAccountIDParam(formattedParams, &offering, customerAccountID)
	if err != nil {
		utils.HandleSpinnerError(spinner, sm, err)
		return err
	}

	var resourceKey string
	found := false
	for _, resourceEntity := range offering.ResourceParameters {
		if strings.EqualFold(resourceEntity.Name, resource) {
			found = true
			resourceKey = resourceEntity.UrlKey
		}
	}

	if !found {
		err = fmt.Errorf("resource %s not found in the service offering", resource)
		utils.HandleSpinnerError(spinner, sm, err)
		return err
	}

	request := openapiclientfleet.FleetCreateResourceInstanceRequest2{
		ProductTierVersion: &version,
		CloudProvider:      &cloudProvider,
		Region:             &region,
		RequestParams:      formattedParams,
		NetworkType:        nil,
	}
	if tagsProvided {
		request.CustomTags = customTags
	}
	if len(workflowBreakpoints) > 0 {
		request.WorkflowBreakpoints = workflowBreakpoints
	}
	if subscriptionID != "" {
		request.SubscriptionId = utils.ToPtr(subscriptionID)
	}
	instance, err := dataaccess.CreateResourceInstance(cmd.Context(), token,
		res.ConsumptionDescribeServiceOfferingResult.ServiceProviderId,
		res.ConsumptionDescribeServiceOfferingResult.ServiceURLKey,
		offering.ServiceAPIVersion,
		offering.ServiceEnvironmentURLKey,
		offering.ServiceModelURLKey,
		offering.ProductTierURLKey,
		resourceKey,
		request)
	if err != nil {
		utils.HandleSpinnerError(spinner, sm, err)
		return err
	}

	if res == nil || instance.Id == nil {
		err = errors.New("failed to create instance")
		utils.HandleSpinnerError(spinner, sm, err)
		return err
	}

	utils.HandleSpinnerSuccess(spinner, sm, "Successfully created instance")

	// Search for the instance
	searchRes, err := dataaccess.SearchInventory(cmd.Context(), token, fmt.Sprintf("resourceinstance:%s", *instance.Id))
	if err != nil {
		utils.PrintError(err)
		return err
	}

	if len(searchRes.ResourceInstanceResults) == 0 {
		err = errors.New("failed to find the created instance")
		utils.PrintError(err)
		return err
	}

	// Format instance
	formattedInstance := formatInstance(&searchRes.ResourceInstanceResults[0], false)
	InstanceID = formattedInstance.InstanceID

	// Print output
	if err = utils.PrintTextTableJsonOutput(output, formattedInstance); err != nil {
		return err
	}

	// Display workflow resource-wise data if output is not JSON and wait flag is enabled
	if output != "json" && waitFlag {
		fmt.Println("🔄 Deployment progress...")
		err = DisplayWorkflowResourceDataWithSpinners(cmd.Context(), token, formattedInstance.InstanceID, "create")
		if err != nil {
			// Handle spinner error if deployment monitoring fails
			fmt.Fprintln(os.Stderr, "❌ Deployment failed")
			return err
		} else {
			fmt.Println("✅ Deployment successful")
		}
	}

	return nil
}

// Helper functions

func getResource(ctx context.Context, token, serviceNameArg, environmentArg, planNameArg, resourceNameArg string) (serviceID, environmentID, productTierID, resourceID string, err error) {
	searchRes, err := dataaccess.SearchInventory(ctx, token, fmt.Sprintf("resource:%s", resourceNameArg))
	if err != nil {
		return
	}

	found := false
	for _, res := range searchRes.ResourceResults {
		if res.Id == "" {
			continue
		}
		if strings.EqualFold(res.Name, resourceNameArg) &&
			strings.EqualFold(res.ServiceName, serviceNameArg) &&
			strings.EqualFold(res.ProductTierName, planNameArg) &&
			strings.EqualFold(res.ServiceEnvironmentName, environmentArg) {
			found = true
			serviceID = res.ServiceId
			environmentID = res.ServiceEnvironmentId
			productTierID = res.ProductTierId
			resourceID = res.Id
			break
		}
	}

	if !found {
		err = fmt.Errorf("target resource not found. Please check input values and try again")
		return
	}

	return
}

func formatInstance(instance *openapiclientfleet.ResourceInstanceSearchRecord, truncateNames bool) model.Instance {
	planName := ""
	if instance.ProductTierName != nil {
		planName = *instance.ProductTierName
	}
	planVersion := ""
	if instance.ProductTierVersion != nil {
		planVersion = *instance.ProductTierVersion
	}
	serviceName := instance.ServiceName
	if truncateNames {
		serviceName = utils.TruncateString(serviceName, defaultMaxNameLength)
		planName = utils.TruncateString(planName, defaultMaxNameLength)
	}
	subscriptionID := ""
	if instance.SubscriptionId != nil {
		subscriptionID = *instance.SubscriptionId
	}

	// Format tags as comma-separated key=value pairs
	tags := formatTags(instance.CustomTags)

	formattedInstance := model.Instance{
		InstanceID:     instance.Id,
		Service:        serviceName,
		Environment:    instance.ServiceEnvironmentName,
		Plan:           planName,
		Version:        planVersion,
		Resource:       instance.ResourceName,
		CloudProvider:  instance.CloudProvider,
		Region:         instance.RegionCode,
		Status:         instance.Status,
		SubscriptionID: subscriptionID,
		Tags:           tags,
	}

	return formattedInstance
}

func parseWorkflowBreakpoints(valuesCSV string) ([]openapiclientfleet.WorkflowBreakpoint, error) {
	if len(valuesCSV) == 0 {
		return nil, nil
	}

	var breakpoints []openapiclientfleet.WorkflowBreakpoint
	seen := make(map[string]struct{})

	// Split the input by comma and trim spaces to get individual IDs or keys
	values := strings.Split(valuesCSV, ",")

	for _, v := range values {
		idOrKey := strings.TrimSpace(v)
		if idOrKey == "" {
			continue
		}
		if _, exists := seen[idOrKey]; exists {
			continue
		}
		seen[idOrKey] = struct{}{}
		breakpoints = append(breakpoints, openapiclientfleet.WorkflowBreakpoint{Id: idOrKey})
	}

	if len(breakpoints) == 0 {
		return nil, fmt.Errorf("breakpoints flag provided but no valid resource IDs/keys found")
	}

	return breakpoints, nil
}

func applyCustomerAccountIDParam(
	params map[string]any,
	offering *openapiclientfleet.ServiceOffering,
	customerAccountID string,
) (map[string]any, error) {
	customerAccountID = strings.TrimSpace(customerAccountID)
	if offering == nil {
		return nil, fmt.Errorf("service offering is required to validate customer account inputs")
	}

	existingCustomerAccountID := customerAccountParamValue(params)
	if strings.EqualFold(offering.ServiceModelType, serviceModelTypeBYOA) &&
		customerAccountID == "" &&
		existingCustomerAccountID == "" {
		return nil, fmt.Errorf(
			"selected service plan is BYOA and requires a customer cloud account. Provide --customer-account-id or set %s in --param/--param-file. Use 'omnistrate-ctl account customer list' to find the onboarding instance ID",
			customerAccountConfigIDParamKey,
		)
	}

	if customerAccountID == "" {
		return params, nil
	}
	if !strings.EqualFold(offering.ServiceModelType, serviceModelTypeBYOA) {
		return nil, fmt.Errorf(
			"--customer-account-id is only supported for BYOA service plans, got %q",
			offering.ServiceModelType,
		)
	}

	if existingCustomerAccountID != "" {
		return nil, fmt.Errorf(
			"%s is already set in --param/--param-file; remove it and use --customer-account-id instead",
			customerAccountConfigIDParamKey,
		)
	}

	if params == nil {
		params = make(map[string]any)
	}
	params[customerAccountConfigIDParamKey] = customerAccountID
	return params, nil
}

func customerAccountParamValue(params map[string]any) string {
	if params == nil {
		return ""
	}

	value, ok := params[customerAccountConfigIDParamKey]
	if !ok || value == nil {
		return ""
	}

	switch typed := value.(type) {
	case string:
		return strings.TrimSpace(typed)
	default:
		return strings.TrimSpace(fmt.Sprint(typed))
	}
}
