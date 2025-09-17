package deploy

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/omnistrate-oss/omnistrate-ctl/cmd/build"
	"github.com/omnistrate-oss/omnistrate-ctl/cmd/common"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/config"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/dataaccess"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/utils"

	"github.com/chelnak/ysmrr"
	"github.com/compose-spec/compose-go/loader"
	"github.com/compose-spec/compose-go/types"
	openapiclientfleet "github.com/omnistrate-oss/omnistrate-sdk-go/fleet"
	openapiclient "github.com/omnistrate-oss/omnistrate-sdk-go/v1"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

const (
	defaultDevEnvName  = "Development"
	defaultProdEnvName = "Production"
	deployExample      = `
# Deploy a service using a spec file (automatically creates/upgrades instances)
omctl deploy spec.yaml

# Deploy a service with a custom product name
omctl deploy spec.yaml --product-name "My Service"

# Deploy a service with release description
omctl deploy spec.yaml --release-description "v1.0.0-alpha"

# Deploy with custom subscription name
omctl deploy spec.yaml --subscription-name "my-custom-subscription"
`
)

// DeployCmd represents the deploy command
var DeployCmd = &cobra.Command{
	Use:     "deploy [spec-file]",
	Short:   "Deploy a service using a spec file",
	Long:    "Deploy a service using a spec file. This command builds the service in DEV, creates/checks PROD environment, promotes to PROD, marks as preferred, subscribes, and automatically creates/upgrades instances.",
	Example: deployExample,
	Args:    cobra.MaximumNArgs(1),
	RunE:    runDeploy,
}

func init() {
	DeployCmd.Flags().String("product-name", "", "Specify a custom service name. If not provided, directory name will be used.")
	DeployCmd.Flags().String("release-description", "", "Release description for the version")
	DeployCmd.Flags().String("subscription-name", "", "Subscription name for service subscription")
}

func runDeploy(cmd *cobra.Command, args []string) error {
	defer config.CleanupArgsAndFlags(cmd, &args)

	// Get the spec file path
	var specFile string
	if len(args) > 0 {
		specFile = args[0]
	} else {
		// Try to find a default spec file
		possibleFiles := []string{"compose.yaml", "compose.yml", "docker-compose.yaml", "docker-compose.yml", "spec.yaml"}
		for _, file := range possibleFiles {
			if _, err := os.Stat(file); err == nil {
				specFile = file
				break
			}
		}
		if specFile == "" {
			return errors.New("no spec file provided and no default spec file found (compose.yaml, compose.yml, docker-compose.yaml, docker-compose.yml, spec.yaml)")
		}
	}

	// Convert to absolute path
	absSpecFile, err := filepath.Abs(specFile)
	if err != nil {
		return errors.Wrap(err, "failed to get absolute path for spec file")
	}

	// Check if spec file exists
	if _, err := os.Stat(absSpecFile); os.IsNotExist(err) {
		return errors.Errorf("spec file does not exist: %s", absSpecFile)
	}

	// Get flags
	productName, err := cmd.Flags().GetString("product-name")
	if err != nil {
		return err
	}

	releaseDescription, err := cmd.Flags().GetString("release-description")
	if err != nil {
		return err
	}

	// Note: subscription-name flag is used for custom subscription naming
	// Instance creation/upgrade is handled automatically

	// Initialize spinner manager
	sm := ysmrr.NewSpinnerManager()
	sm.Start()
	defer sm.Stop()

	// Step 0: Validate user is logged in
	spinner := sm.AddSpinner("Checking if user is logged in")
	time.Sleep(1 * time.Second)
	spinner.Complete()
	sm.Stop()

	token, err := common.GetTokenWithLogin()
	if err != nil {
		return err
	}

	sm = ysmrr.NewSpinnerManager()
	sm.Start()

	// Step 1: Read and parse the spec file
	spinner = sm.AddSpinner("Reading and parsing spec file")
	fileData, err := os.ReadFile(absSpecFile)
	if err != nil {
		utils.HandleSpinnerError(spinner, sm, err)
		return err
	}

	// Process template expressions recursively
	processedData, err := processTemplateExpressions(fileData, filepath.Dir(absSpecFile))
	if err != nil {
		utils.HandleSpinnerError(spinner, sm, err)
		return err
	}

	// Determine spec type
	specType, err := determineSpecType(processedData)
	if err != nil {
		utils.HandleSpinnerError(spinner, sm, err)
		return err
	}

	spinner.UpdateMessage(fmt.Sprintf("Reading and parsing spec file: %s (%s)", filepath.Base(absSpecFile), specType))
	spinner.Complete()

	// Step 2: Determine service name
	spinner = sm.AddSpinner("Determining service name")
	serviceNameToUse := productName
	if serviceNameToUse == "" {
		// Use directory name as default
		serviceNameToUse = filepath.Base(filepath.Dir(absSpecFile))
		if serviceNameToUse == "." || serviceNameToUse == "/" {
			serviceNameToUse = "my-service"
		}
	}
	spinner.UpdateMessage(fmt.Sprintf("Determining service name: %s", serviceNameToUse))
	spinner.Complete()

	// Step 3: Build service in DEV environment with release-as-preferred
	spinner = sm.AddSpinner("Building service in DEV environment")
	
	// Prepare release description pointer
	var releaseDescriptionPtr *string
	if releaseDescription != "" {
		releaseDescriptionPtr = &releaseDescription
	}

	serviceID, devEnvironmentID, devPlanID, undefinedResources, err := buildServiceInDev(
		cmd.Context(),
		processedData,
		token,
		serviceNameToUse,
		specType,
		releaseDescriptionPtr,
	)
	if err != nil {
		utils.HandleSpinnerError(spinner, sm, err)
		return err
	}

	spinner.UpdateMessage(fmt.Sprintf("Building service in DEV environment: built service %s (ID: %s)", serviceNameToUse, serviceID))
	spinner.Complete()

	// Print warning if there are any undefined resources
	if len(undefinedResources) > 0 {
		sm.Stop()
		utils.PrintWarning("The following resources appear in the service plan but were not defined in the spec:")
		for resourceName, resourceID := range undefinedResources {
			utils.PrintWarning(fmt.Sprintf("  %s: %s", resourceName, resourceID))
		}
		utils.PrintWarning("These resources were not processed during the build.")
		sm = ysmrr.NewSpinnerManager()
		sm.Start()
	}

	// Step 4: Check if production environment exists
	spinner = sm.AddSpinner("Checking if the production environment is set up")
	time.Sleep(1 * time.Second) // Add a delay to show the spinner
	prodEnvironmentID, err := checkIfProdEnvExists(cmd.Context(), token, serviceID)
	if err != nil {
		utils.HandleSpinnerError(spinner, sm, err)
		return err
	}
	
	yesOrNo := "No"
	if prodEnvironmentID != "" {
		yesOrNo = "Yes"
	}
	spinner.UpdateMessage(fmt.Sprintf("Checking if the production environment is set up: %s", yesOrNo))
	spinner.Complete()

	// Step 5: Create production environment if it doesn't exist
	if prodEnvironmentID == "" {
		spinner = sm.AddSpinner("Creating a production environment")
		prodEnvironmentID, err = createProdEnv(cmd.Context(), token, serviceID, devEnvironmentID)
		if err != nil {
			utils.HandleSpinnerError(spinner, sm, err)
			return err
		}
		spinner.UpdateMessage(fmt.Sprintf("Creating a production environment: created environment %s (environment ID: %s)", defaultProdEnvName, prodEnvironmentID))
		spinner.Complete()
	}

	// Step 6: Promote the service to the production environment
	spinner = sm.AddSpinner(fmt.Sprintf("Promoting the service to the %s environment", defaultProdEnvName))
	err = dataaccess.PromoteServiceEnvironment(cmd.Context(), token, serviceID, devEnvironmentID)
	if err != nil {
		utils.PrintError(err)
		return err
	}
	spinner.UpdateMessage("Promoting the service to the production environment: Success")
	spinner.Complete()

	// Step 7: Set service plan as preferred in production
	spinner = sm.AddSpinner("Setting service plan as preferred in production")
	
	// Get service details to check production plans
	service, err := dataaccess.DescribeService(cmd.Context(), token, serviceID)
	if err != nil {
		utils.HandleSpinnerError(spinner, sm, err)
		return err
	}
	
	// Find the production environment and check if it has service plans
	var hasProductionPlans bool
	var prodPlanID string
	
	for _, env := range service.ServiceEnvironments {
		if env.Id == prodEnvironmentID {
			if len(env.ServicePlans) > 0 {
				hasProductionPlans = true
				// Get dev product tier details to match with production plan
				devProductTier, err := dataaccess.DescribeProductTier(cmd.Context(), token, serviceID, devPlanID)
				if err != nil {
					utils.HandleSpinnerError(spinner, sm, err)
					return err
				}
				
				// Find the production plan with the same name as the dev plan
				for _, plan := range env.ServicePlans {
					if plan.Name == devProductTier.Name {
						prodPlanID = plan.ProductTierID
						break
					}
				}
			}
			break
		}
	}

	if !hasProductionPlans {
		spinner.UpdateMessage("Setting service plan as preferred in production: Skipped (no plans available - promotion required)")
		spinner.Complete()
		fmt.Printf("Note: Service plan preference cannot be set until promotion is completed.\n")
		fmt.Printf("After promoting, you can set the preferred plan using the serviceplan commands.\n\n")
	} else if prodPlanID == "" {
		spinner.UpdateMessage("Setting service plan as preferred in production: Skipped (matching plan not found)")
		spinner.Complete()
		fmt.Printf("Warning: Could not find matching production plan for preference setting.\n\n")
	} else {
		// Find the latest version of the production plan
		targetVersion, err := dataaccess.FindLatestVersion(cmd.Context(), token, serviceID, prodPlanID)
		if err != nil {
			utils.HandleSpinnerError(spinner, sm, err)
			return err
		}

		// Set as preferred
		_, err = dataaccess.SetDefaultServicePlan(cmd.Context(), token, serviceID, prodPlanID, targetVersion)
		if err != nil {
			utils.HandleSpinnerError(spinner, sm, err)
			return err
		}
		spinner.UpdateMessage("Setting service plan as preferred in production: Success")
		spinner.Complete()
	}

	// Step 8: Create subscription for the production service
	spinner = sm.AddSpinner("Creating subscription to the production service")
	
	// Get subscription flags or use defaults
	subscriptionName, _ := cmd.Flags().GetString("subscription-name")
	if subscriptionName == "" {
		subscriptionName = fmt.Sprintf("%s-subscription", serviceNameToUse)
	}
	
	// Create subscription if we have production plans
	var subscriptionID string
	var isServiceProvider bool
	if hasProductionPlans && prodPlanID != "" {
		// Get current user ID for subscription
		user, err := dataaccess.DescribeUser(cmd.Context(), token)
		if err != nil {
			utils.HandleSpinnerError(spinner, sm, err)
			return err
		}
		
		subscriptionOpts := &dataaccess.CreateSubscriptionOnBehalfOptions{
			ProductTierID:            prodPlanID,
			OnBehalfOfCustomerUserID: user.Id,
		}
		
		subscriptionResp, err := dataaccess.CreateSubscriptionOnBehalf(cmd.Context(), token, serviceID, prodEnvironmentID, subscriptionOpts)
		if err != nil {
			// Check if this is the service provider org error
			if strings.Contains(err.Error(), "cannot create subscription on behalf of customer user in service provider org") {
				isServiceProvider = true
				spinner.UpdateMessage("Creating subscription to the production service: Skipped (service provider org - will create instance directly)")
				spinner.Complete()
			} else {
				utils.HandleSpinnerError(spinner, sm, err)
				return err
			}
		} else {
			subscriptionID = *subscriptionResp.Id
			spinner.UpdateMessage(fmt.Sprintf("Creating subscription to the production service: created subscription %s (ID: %s)", subscriptionName, subscriptionID))
			spinner.Complete()
		}
	} else {
		spinner.UpdateMessage("Creating subscription to the production service: Skipped (no production plans available)")
		spinner.Complete()
	}

	// Step 9: Create or upgrade instance deployment automatically
	var instanceID string
	var instanceName string
	
	// Proceed with instance creation either via subscription or directly for service providers
	if subscriptionID != "" || isServiceProvider {
		// Generate default instance name from service name
		instanceName = fmt.Sprintf("%s-instance", serviceNameToUse)
		
		if subscriptionID != "" {
			// Normal subscription-based flow
			spinner = sm.AddSpinner("Checking for existing instances")
			
			// Check if instance already exists for this subscription
			existingInstanceID, err := findExistingInstance(cmd.Context(), token, subscriptionID)
			if err != nil {
				spinner.UpdateMessage("Checking for existing instances: Failed")
				spinner.Complete()
				fmt.Printf("Warning: Failed to check for existing instances: %s\n", err.Error())
			} else if existingInstanceID != "" {
				// Instance exists - upgrade it
				spinner.UpdateMessage("Checking for existing instances: Found existing instance")
				spinner.Complete()
				
				spinner = sm.AddSpinner(fmt.Sprintf("Upgrading existing instance: %s", existingInstanceID))
				upgradeErr := upgradeExistingInstance(cmd.Context(), token, existingInstanceID, serviceID, prodPlanID)
				if upgradeErr != nil {
					spinner.UpdateMessage(fmt.Sprintf("Upgrading existing instance: Failed (%s)", upgradeErr.Error()))
					spinner.Complete()
					fmt.Printf("Warning: Instance upgrade failed: %s\n", upgradeErr.Error())
				} else {
					instanceID = existingInstanceID
					spinner.UpdateMessage(fmt.Sprintf("Upgrading existing instance: Success (ID: %s)", instanceID))
					spinner.Complete()
				}
			} else {
				// No instance exists - create new one
				spinner.UpdateMessage("Checking for existing instances: No existing instances found")
				spinner.Complete()
				
				
				// Try to create instance with default parameters
				createdInstanceID, err := createInstanceWithDefaults(cmd.Context(), token, serviceID, prodEnvironmentID, prodPlanID, subscriptionID)
				if err != nil {
					spinner.UpdateMessage(fmt.Sprintf("Creating new instance deployment: Failed (%s)", err.Error()))
					spinner.Complete()
					fmt.Printf("Error: Failed to create instance: %s\n", err.Error())
				} else {
					instanceID = createdInstanceID
					spinner.UpdateMessage(fmt.Sprintf("Creating new instance deployment: Success (ID: %s)", instanceID))
					spinner.Complete()
				}
			}
		} else if isServiceProvider {
			
			// Try to create instance without subscription for service provider
			createdInstanceID, err := createInstanceWithoutSubscription(cmd.Context(), token, serviceID, prodEnvironmentID, prodPlanID)
			if err != nil {
				spinner.UpdateMessage(fmt.Sprintf("Creating instance for service provider: Failed (%s)", err.Error()))
				spinner.Complete()
				fmt.Printf("Error: Failed to create service provider instance: %s\n", err.Error())
			} else {
				instanceID = createdInstanceID
				spinner.UpdateMessage(fmt.Sprintf("Creating instance for service provider: Success (ID: %s)", instanceID))
				spinner.Complete()
			}
		}
	} else {
		fmt.Println("Warning: No subscription created and not a service provider org - instance creation skipped")
	}

	// Step 10: Success message - completed deployment
	spinner = sm.AddSpinner("Deployment workflow completed")
	spinner.UpdateMessage("Deployment workflow completed: Service built, promoted, and ready for instances")
	spinner.Complete()

	sm.Stop()

	// Success message
	fmt.Println()
	fmt.Println("🎉 Deployment completed successfully!")
	fmt.Printf("   Service: %s (ID: %s)\n", serviceNameToUse, serviceID)
	fmt.Printf("   Production Environment: %s (ID: %s)\n", defaultProdEnvName, prodEnvironmentID)
	if subscriptionID != "" {
		fmt.Printf("   Subscription: %s (ID: %s)\n", subscriptionName, subscriptionID)
	} else if isServiceProvider {
		fmt.Printf("   Organization: Service Provider (direct instance deployment)\n")
	}
	if instanceID != "" {
		fmt.Printf("   Instance: %s (ID: %s)\n", instanceName, instanceID)
	}
	fmt.Println()
	fmt.Println("Next steps:")
	if instanceID != "" {
		fmt.Println("1. Monitor your instance deployment status")
		fmt.Println("2. Configure monitoring and alerting for your instances")
		fmt.Println("3. Set up custom domains and SSL certificates if required")
	} else if subscriptionID != "" {
		fmt.Println("1. Create instance deployments using: omctl instance create")
		fmt.Println("2. Configure monitoring and alerting for your instances")
		fmt.Println("3. Set up custom domains and SSL certificates if required")
	} else {
		fmt.Println("1. Subscribe to your service in the Omnistrate UI")
		fmt.Println("2. Create instance deployments as needed")
		fmt.Println("3. Configure monitoring and alerting for your instances")
		fmt.Println("4. Set up custom domains and SSL certificates if required")
	}
	fmt.Println()

	return nil
}

// buildServiceInDev builds a service in DEV environment with release-as-preferred
func buildServiceInDev(ctx context.Context, fileData []byte, token, name, specType string, releaseDescription *string) (serviceID string, environmentID string, productTierID string, undefinedResources map[string]string, err error) {
	if name == "" {
		return "", "", "", make(map[string]string), errors.New("name is required")
	}

	if specType == "" {
		return "", "", "", make(map[string]string), errors.New("specType is required")
	}

	switch specType {
	case build.ServicePlanSpecType:
		request := openapiclient.BuildServiceFromServicePlanSpecRequest2{
			Name:               name,
			Description:        nil,
			ServiceLogoURL:     nil,
			Environment:        nil,
			EnvironmentType:    nil,
			FileContent:        base64.StdEncoding.EncodeToString(fileData),
			Release:            utils.ToPtr(true),
			ReleaseAsPreferred: utils.ToPtr(true),
			ReleaseVersionName: releaseDescription,
			Dryrun:             utils.ToPtr(false),
		}

		buildRes, err := dataaccess.BuildServiceFromServicePlanSpec(ctx, token, request)
		if err != nil {
			return "", "", "", make(map[string]string), err
		}
		if buildRes == nil {
			return "", "", "", make(map[string]string), errors.New("empty response from server")
		}

		undefinedResources := make(map[string]string)
		if buildRes.UndefinedResources != nil {
			undefinedResources = *buildRes.UndefinedResources
		}

		return buildRes.ServiceID, buildRes.ServiceEnvironmentID, buildRes.ProductTierID, undefinedResources, nil

	case build.DockerComposeSpecType:
		request := openapiclient.BuildServiceFromComposeSpecRequest2{
			Name:               name,
			Description:        nil,
			ServiceLogoURL:     nil,
			Environment:        nil,
			EnvironmentType:    nil,
			FileContent:        base64.StdEncoding.EncodeToString(fileData),
			Release:            utils.ToPtr(true),
			ReleaseAsPreferred: utils.ToPtr(true),
			ReleaseVersionName: releaseDescription,
			Dryrun:             utils.ToPtr(false),
		}

		buildRes, err := dataaccess.BuildServiceFromComposeSpec(ctx, token, request)
		if err != nil {
			return "", "", "", make(map[string]string), err
		}
		if buildRes == nil {
			return "", "", "", make(map[string]string), errors.New("empty response from server")
		}

		undefinedResources = make(map[string]string)
		if buildRes.UndefinedResources != nil {
			undefinedResources = *buildRes.UndefinedResources
		}

		return buildRes.ServiceID, buildRes.ServiceEnvironmentID, buildRes.ProductTierID, undefinedResources, nil

	default:
		return "", "", "", make(map[string]string), errors.Errorf("unsupported spec type: %s (supported: %s, %s)", specType, build.ServicePlanSpecType, build.DockerComposeSpecType)
	}
}

// processTemplateExpressions processes template expressions like {{ $file:path }} recursively
func processTemplateExpressions(data []byte, baseDir string) ([]byte, error) {
	content := string(data)
	
	// Pattern to match {{ $file:path }}
	re := regexp.MustCompile(`(?m)^(?P<indent>[ \t]*)?(?P<key>[\S\t ]*)?{{\s*\$file:(?P<filepath>[^\s}]+)\s*}}`)
	
	for {
		if !re.MatchString(content) {
			break
		}
		
		var processingErr error
		content = re.ReplaceAllStringFunc(content, func(match string) string {
			submatches := re.FindStringSubmatch(match)
			if len(submatches) < 4 {
				processingErr = fmt.Errorf("invalid file reference: %s", match)
				return match
			}
			
			indent := submatches[1]
			key := submatches[2]
			filePath := submatches[3]
			
			if filePath == "" {
				processingErr = fmt.Errorf("empty file path in reference: %s", match)
				return match
			}
			
			// Resolve file path
			var fullPath string
			if filepath.IsAbs(filePath) {
				fullPath = filePath
			} else {
				fullPath = filepath.Join(baseDir, filePath)
			}
			
			// Read file content
			fileContent, err := os.ReadFile(fullPath)
			if err != nil {
				processingErr = fmt.Errorf("failed to read file %s: %v", fullPath, err)
				return match
			}
			
			// Process nested template expressions
			processedContent, err := processTemplateExpressions(fileContent, filepath.Dir(fullPath))
			if err != nil {
				processingErr = fmt.Errorf("failed to process templates in %s: %v", fullPath, err)
				return match
			}
			
			// Apply indentation
			lines := strings.Split(string(processedContent), "\n")
			result := make([]string, len(lines))
			
			for i, line := range lines {
				if i == 0 {
					result[i] = indent + key + line
				} else if strings.TrimSpace(line) != "" {
					result[i] = indent + line
				} else {
					result[i] = line
				}
			}
			
			return strings.Join(result, "\n")
		})
		
		if processingErr != nil {
			return nil, processingErr
		}
	}
	
	return []byte(content), nil
}

// determineSpecType determines whether the spec is DockerCompose or ServicePlanSpec
func determineSpecType(data []byte) (string, error) {
	content := string(data)
	
	// First check for obvious Docker Compose indicators
	if strings.Contains(content, "version:") && strings.Contains(content, "services:") {
		return build.DockerComposeSpecType, nil
	}
	
	// Check for volumes at root level (Docker Compose indicator)
	if strings.Contains(content, "volumes:") && strings.Contains(content, "services:") {
		return build.DockerComposeSpecType, nil
	}
	
	// Try to parse as YAML first
	parsedYaml, err := loader.ParseYAML(data)
	if err != nil {
		// If not valid YAML, check for compose content
		if strings.Contains(content, "services:") {
			return build.DockerComposeSpecType, nil
		}
		return "", errors.Wrap(err, "failed to parse spec file")
	}

	// Try to parse as compose project
	_, err = loader.LoadWithContext(context.Background(), types.ConfigDetails{
		ConfigFiles: []types.ConfigFile{
			{
				Config: parsedYaml,
			},
		},
	})
	
	if err == nil {
		return build.DockerComposeSpecType, nil
	}
	
	// Default to ServicePlanSpec if not a valid compose file
	return build.ServicePlanSpecType, nil
}

// Helper functions (similar to build_from_repo.go)

func checkIfProdEnvExists(ctx context.Context, token string, serviceID string) (string, error) {
	prodEnvironment, err := dataaccess.FindEnvironment(ctx, token, serviceID, "prod")
	if errors.As(err, &dataaccess.ErrEnvironmentNotFound) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return prodEnvironment.Id, nil
}

func createProdEnv(ctx context.Context, token string, serviceID string, devEnvironmentID string) (string, error) {
	// Get default deployment config ID
	defaultDeploymentConfigID, err := dataaccess.GetDefaultDeploymentConfigID(ctx, token)
	if err != nil {
		return "", err
	}

	prodEnvironmentID, err := dataaccess.CreateServiceEnvironment(ctx, token,
		defaultProdEnvName,
		"Production environment",
		serviceID,
		"PUBLIC",
		"PROD",
		utils.ToPtr(devEnvironmentID),
		defaultDeploymentConfigID,
		true,
		nil,
	)
	if err != nil {
		return "", err
	}

	return prodEnvironmentID, nil
}

// createInstanceWithDefaults creates an instance with default parameters for the first resource found
func createInstanceWithDefaults(ctx context.Context, token, serviceID, environmentID, productTierID, subscriptionID string) (string, error) {
	// Get the latest version
	version, err := dataaccess.FindLatestVersion(ctx, token, serviceID, productTierID)
	if err != nil {
		return "", fmt.Errorf("failed to find latest version: %w", err)
	}

	// Describe service offering
	res, err := dataaccess.DescribeServiceOffering(ctx, token, serviceID, productTierID, version)
	if err != nil {
		return "", fmt.Errorf("failed to describe service offering: %w", err)
	}

	if len(res.ConsumptionDescribeServiceOfferingResult.Offerings) == 0 {
		return "", fmt.Errorf("no service offerings found")
	}

	offering := res.ConsumptionDescribeServiceOfferingResult.Offerings[0]

	// Find the first resource with parameters
	if len(offering.ResourceParameters) == 0 {
		return "", fmt.Errorf("no resources found in service offering")
	}

	// Use the first resource
	resourceEntity := offering.ResourceParameters[0]
	resourceKey := resourceEntity.UrlKey

	// Create default parameters with common sensible defaults
	// Using minimal defaults to reduce validation errors
	defaultParams := map[string]interface{}{
	
	}

	// Try to use default cloud provider and region (AWS us-east-1 as fallback)
	cloudProvider := "aws"
	region := "ap-south-1"

	// Create the instance request
	request := openapiclientfleet.FleetCreateResourceInstanceRequest2{
		ProductTierVersion: &version,
		CloudProvider:      &cloudProvider,
		Region:             &region,
		RequestParams:      defaultParams,
		NetworkType:        nil,
		SubscriptionId:     utils.ToPtr(subscriptionID),
	}

	// Create the instance
	instance, err := dataaccess.CreateResourceInstance(ctx, token,
		res.ConsumptionDescribeServiceOfferingResult.ServiceProviderId,
		res.ConsumptionDescribeServiceOfferingResult.ServiceURLKey,
		offering.ServiceAPIVersion,
		offering.ServiceEnvironmentURLKey,
		offering.ServiceModelURLKey,
		offering.ProductTierURLKey,
		resourceKey,
		request)
	if err != nil {
		return "", fmt.Errorf("failed to create resource instance: %w", err)
	}

	if instance == nil || instance.Id == nil {
		return "", fmt.Errorf("instance creation returned empty result")
	}

	return *instance.Id, nil
}

// createInstanceWithoutSubscription creates an instance directly for service provider orgs (without subscription)
func createInstanceWithoutSubscription(ctx context.Context, token, serviceID, environmentID, productTierID string) (string, error) {
	// Get the latest version
	version, err := dataaccess.FindLatestVersion(ctx, token, serviceID, productTierID)
	if err != nil {
		return "", fmt.Errorf("failed to find latest version: %w", err)
	}

	// Describe service offering
	res, err := dataaccess.DescribeServiceOffering(ctx, token, serviceID, productTierID, version)
	if err != nil {
		return "", fmt.Errorf("failed to describe service offering: %w", err)
	}

	if len(res.ConsumptionDescribeServiceOfferingResult.Offerings) == 0 {
		return "", fmt.Errorf("no service offerings found")
	}

	offering := res.ConsumptionDescribeServiceOfferingResult.Offerings[0]

	// Find the first resource with parameters
	if len(offering.ResourceParameters) == 0 {
		return "", fmt.Errorf("no resources found in service offering")
	}

	// Use the first resource
	resourceEntity := offering.ResourceParameters[0]
	resourceKey := resourceEntity.UrlKey

	// Create default parameters with common sensible defaults
	// Using the same approach as instance/create.go but with more minimal defaults
	defaultParams := map[string]interface{}{
		
	}

	// Try to use default cloud provider and region (AWS us-east-1 as fallback)
	cloudProvider := "aws"
	region := "ap-south-1"

	// Create the instance request WITHOUT subscription ID (for service provider orgs)
	// Following the same pattern as instance/create.go
	request := openapiclientfleet.FleetCreateResourceInstanceRequest2{
		ProductTierVersion: &version,
		CloudProvider:      &cloudProvider,
		Region:             &region,
		RequestParams:      defaultParams,
		NetworkType:        nil,
		// Note: No SubscriptionId for service provider direct instance creation
	}

	// Create the instance
	instance, err := dataaccess.CreateResourceInstance(ctx, token,
		res.ConsumptionDescribeServiceOfferingResult.ServiceProviderId,
		res.ConsumptionDescribeServiceOfferingResult.ServiceURLKey,
		offering.ServiceAPIVersion,
		offering.ServiceEnvironmentURLKey,
		offering.ServiceModelURLKey,
		offering.ProductTierURLKey,
		resourceKey,
		request)
	if err != nil {
		return "", fmt.Errorf("failed to create resource instance: %w", err)
	}

	if instance == nil || instance.Id == nil {
		return "", fmt.Errorf("instance creation returned empty result")
	}

	return *instance.Id, nil
}

// findExistingInstance searches for existing instances in the subscription
func findExistingInstance(ctx context.Context, token, subscriptionID string) (string, error) {
	// Search for instances associated with this subscription
	searchQuery := fmt.Sprintf("resourceinstance subscription:%s", subscriptionID)
	searchRes, err := dataaccess.SearchInventory(ctx, token, searchQuery)
	if err != nil {
		return "", fmt.Errorf("failed to search for instances: %w", err)
	}

	// Return the first instance found
	if len(searchRes.ResourceInstanceResults) > 0 {
		return searchRes.ResourceInstanceResults[0].Id, nil
	}

	return "", nil // No instance found
}

// upgradeExistingInstance upgrades an existing instance to the latest version
func upgradeExistingInstance(ctx context.Context, token, instanceID, serviceID, productTierID string) error {
	// Get the latest version
	latestVersion, err := dataaccess.FindLatestVersion(ctx, token, serviceID, productTierID)
	if err != nil {
		return fmt.Errorf("failed to find latest version: %w", err)
	}

	// Get instance details to find environment ID
	searchRes, err := dataaccess.SearchInventory(ctx, token, fmt.Sprintf("resourceinstance:%s", instanceID))
	if err != nil {
		return fmt.Errorf("failed to find instance details: %w", err)
	}

	if len(searchRes.ResourceInstanceResults) == 0 {
		return fmt.Errorf("instance not found: %s", instanceID)
	}

	environmentID := searchRes.ResourceInstanceResults[0].ServiceEnvironmentId

	// Perform the upgrade with empty resource override config (use defaults)
	resourceOverrideConfig := make(map[string]openapiclientfleet.ResourceOneOffPatchConfigurationOverride)
	
	err = dataaccess.OneOffPatchResourceInstance(ctx, token,
		serviceID,
		environmentID,
		instanceID,
		resourceOverrideConfig,
		latestVersion,
	)
	if err != nil {
		return fmt.Errorf("failed to upgrade instance: %w", err)
	}

	return nil
}
