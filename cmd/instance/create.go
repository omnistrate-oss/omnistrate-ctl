package instance

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/chelnak/ysmrr"
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
omctl instance create --service=mysql --environment=dev --plan=mysql --version=latest --resource=mySQL --cloud-provider=aws --region=ca-central-1 --param '{"databaseName":"default","password":"a_secure_password","rootPassword":"a_secure_root_password","username":"user"}'

# Create an instance deployment with parameters from a file
omctl instance create --service=mysql --environment=dev --plan=mysql --version=latest --resource=mySQL --cloud-provider=aws --region=ca-central-1 --param-file /path/to/params.json`
)

var InstanceID string

var createCmd = &cobra.Command{
	Use:          "create --service=[service] --environment=[environment] --plan=[plan] --version=[version] --resource=[resource] --cloud-provider=[aws|gcp] --region=[region] [--param=param] [--param-file=file-path]",
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
	createCmd.Flags().String("cloud-provider", "", "Cloud provider (aws|gcp)")
	createCmd.Flags().String("region", "", "Region code (e.g. us-east-2, us-central1)")
	createCmd.Flags().String("param", "", "Parameters for the instance deployment")
	createCmd.Flags().String("param-file", "", "Json file containing parameters for the instance deployment")
	createCmd.Flags().StringP("subscription-id", "", "", "Subscription ID to use for the instance deployment. If not provided, instance deployment will be created in your own subscription.")

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
	subscriptionID, err := cmd.Flags().GetString("subscription-id")
	if err != nil {
		utils.PrintError(err)
		return err
	}
	output, err := cmd.Flags().GetString("output")
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

	// Display workflow resource-wise data if output is not JSON
	if output != "json" {

		fmt.Println("Deployment progress...")
		err = displayWorkflowResourceDataWithSpinners(cmd.Context(), token, formattedInstance.InstanceID)
		if err != nil {
			// Handle spinner error if deployment monitoring fails
			fmt.Printf("Failed to display resource data -- %s",err)
		}
		fmt.Println("Deployment successful")
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
	}

	return formattedInstance
}

// ResourceSpinner holds spinner information for a resource
type ResourceSpinner struct {
	ResourceName string
	Spinner      *ysmrr.Spinner
}

// displayWorkflowResourceDataWithSpinners creates individual spinners for each resource and updates them dynamically
func displayWorkflowResourceDataWithSpinners(ctx context.Context, token, instanceID string) error {
	// Search for the instance to get service details
	searchRes, err := dataaccess.SearchInventory(ctx, token, fmt.Sprintf("resourceinstance:%s", instanceID))
	if err != nil {
		return err
	}

	if len(searchRes.ResourceInstanceResults) == 0 {
		return fmt.Errorf("instance not found")
	}

	instance := searchRes.ResourceInstanceResults[0]
	
	// Initialize spinner manager
	var sm ysmrr.SpinnerManager
	sm = ysmrr.NewSpinnerManager()
	
	// Track resource spinners
	var resourceSpinners []ResourceSpinner
	
	// Function to create or update spinners for each resource
	createOrUpdateSpinners := func(resourcesData []dataaccess.ResourceWorkflowData) {
		// If this is the first time, create spinners for each resource
		if len(resourceSpinners) == 0 {
			for _, resourceData := range resourcesData {
				spinner := sm.AddSpinner(fmt.Sprintf("%s: Initializing...", resourceData.ResourceName))
				resourceSpinners = append(resourceSpinners, ResourceSpinner{
					ResourceName: resourceData.ResourceName,
					Spinner:      spinner,
				})
			}
			sm.Start()
		}
		
		// Update each resource spinner with current status
		for i, resourceData := range resourcesData {
			if i < len(resourceSpinners) {
				// Get status icons for each category
				bootstrapEventType := getHighestPriorityEventType(resourceData.EventsByCategory.Bootstrap)
				bootstrapIcon := getEventStatusIconFromType(bootstrapEventType)
				
				storageEventType := getHighestPriorityEventType(resourceData.EventsByCategory.Storage)
				storageIcon := getEventStatusIconFromType(storageEventType)
				
				networkEventType := getHighestPriorityEventType(resourceData.EventsByCategory.Network)
				networkIcon := getEventStatusIconFromType(networkEventType)
				
				computeEventType := getHighestPriorityEventType(resourceData.EventsByCategory.Compute)
				computeIcon := getEventStatusIconFromType(computeEventType)

				deploymentEventType := getHighestPriorityEventType(resourceData.EventsByCategory.Deployment)
				deploymentIcon := getEventStatusIconFromType(deploymentEventType)

				monitoringEventType := getHighestPriorityEventType(resourceData.EventsByCategory.Monitoring)
				monitoringIcon := getEventStatusIconFromType(monitoringEventType)

				// Create dynamic message for this resource
				message := fmt.Sprintf("%s | Bootstrap: %s | Storage: %s | Network: %s | Compute: %s | Deployment: %s | Monitoring: %s",
					resourceData.ResourceName,
					bootstrapIcon,
					storageIcon,
					networkIcon,
					computeIcon,
					deploymentIcon,
					monitoringIcon)
				
				// Update spinner message
				resourceSpinners[i].Spinner.UpdateMessage(message)
			}
		}
	}
	
	// Function to complete spinners when deployment is done
	completeSpinners := func(resourcesData []dataaccess.ResourceWorkflowData, workflowInfo *dataaccess.WorkflowInfo) {
		for i, resourceData := range resourcesData {
			if i < len(resourceSpinners) {
				if strings.ToLower(workflowInfo.WorkflowStatus) == "success" {
					resourceSpinners[i].Spinner.UpdateMessage(fmt.Sprintf("%s: Deployment completed successfully", resourceData.ResourceName))
					resourceSpinners[i].Spinner.Complete()
				} else {
					resourceSpinners[i].Spinner.UpdateMessage(fmt.Sprintf("%s: Deployment failed - %s", resourceData.ResourceName, workflowInfo.WorkflowStatus))
					resourceSpinners[i].Spinner.Error()
				}
			}
		}
		sm.Stop()
	}
	
	// Function to fetch and display current workflow status for all resources
	displayCurrentStatus := func() (bool, error) {
		// Get workflow events for all resources in the instance
		resourcesData, workflowInfo, err := dataaccess.GetDebugEventsForAllResources(
			ctx, token,
			instance.ServiceId,
			instance.ServiceEnvironmentId,
			instanceID,
		)
		if err != nil {
			return false, err
		}

		if workflowInfo == nil {
			return true, nil // Stop polling if no workflow data
		}

		// Check if workflow is complete
		isWorkflowComplete := strings.ToLower(workflowInfo.WorkflowStatus) == "success" ||
					 strings.ToLower(workflowInfo.WorkflowStatus) == "failed" ||
					 strings.ToLower(workflowInfo.WorkflowStatus) == "cancelled"

		if len(resourcesData) == 0 {
			return isWorkflowComplete, nil
		}

		// Create or update spinners for each resource
		createOrUpdateSpinners(resourcesData)
		
		// If workflow is complete, complete all spinners and stop
		if isWorkflowComplete {
			completeSpinners(resourcesData, workflowInfo)
			return true, nil
		}

		return false, nil
	}

	// Initial display
	isComplete, err := displayCurrentStatus()
	if err != nil {
		return err
	}

	// If workflow is already complete, don't start polling
	if isComplete {
		return nil
	}

	// Start polling every 10 seconds
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		isComplete, err := displayCurrentStatus()
		if err != nil {
			// Handle error but continue polling
			continue
		}

		// Stop polling when workflow is complete
		if isComplete {
			break
		}
	}

	return nil
}


// getHighestPriorityEventType checks all events in a category and returns the highest priority event type
func getHighestPriorityEventType(events []dataaccess.CustomWorkflowEvent) string {
	if len(events) == 0 {
		return ""
	}

	// Check in priority order
	// 1. First check for failed events (highest priority)
	for _, event := range events {
		if event.EventType == "WorkflowStepFailed" || event.EventType == "WorkflowFailed" {
			return event.EventType
		}
	}

	// 2. Then check for completed events
	for _, event := range events {
		if event.EventType == "WorkflowStepCompleted" {
			return event.EventType
		}
	}

	// 3. Then check for debug or started events
	for _, event := range events {
		if event.EventType == "WorkflowStepDebug" || event.EventType == "WorkflowStepStarted" {
			return event.EventType
		}
	}

	// 4. If none of the above, return the last event type
	if len(events) > 0 {
		return events[len(events)-1].EventType
	}

	return ""
}

// getEventStatusIconFromType returns an appropriate icon based on event type string
func getEventStatusIconFromType(eventType string) string {
	switch eventType {
	case "WorkflowStepFailed", "WorkflowFailed":
		return "❌"
	case "WorkflowStepCompleted":
		return "✅"
	case "WorkflowStepDebug", "WorkflowStepStarted":
		return "🔄"
	default:
		return "🟡"
	}
}
