package deploy

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/chelnak/ysmrr"
	"github.com/omnistrate-oss/omnistrate-ctl/cmd/build"
	"github.com/omnistrate-oss/omnistrate-ctl/cmd/common"
	"github.com/omnistrate-oss/omnistrate-ctl/cmd/instance" // Import the correct package for instancecmd
	"github.com/omnistrate-oss/omnistrate-ctl/internal/config"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/dataaccess"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/utils"
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

# Deploy with AWS account ID for BYOA deployment
omctl deploy spec.yaml --aws-account-id "123456789012"

# Deploy with GCP project for BYOA deployment
omctl deploy spec.yaml --gcp-project-id "my-project" --gcp-project-number "123456789012"

# Perform a dry-run to validate configuration without deploying
omctl deploy spec.yaml --dry-run

# Auto-generate spec from repository and deploy
omctl deploy --product-name "My Repo Service"
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
	DeployCmd.Flags().StringP("file", "f", "", "Path to the docker compose file")
	DeployCmd.Flags().String("product-name", "", "Specify a custom service name. If not provided, directory name will be used.")
	DeployCmd.Flags().Bool("dry-run", false, "Perform validation checks without actually deploying")
	DeployCmd.Flags().String("resource-id", "", "Specify the resource ID to use when multiple resources exist.")
	DeployCmd.Flags().String("instance-id", "", "Specify the instance ID to use when multiple deployments exist.")

	
	DeployCmd.Flags().StringP("environment", "e", "Prod", "Name of the environment to build the service in (default: Prod)")
	DeployCmd.Flags().StringP("environment-type", "t", "prod", "Type of environment. Valid options include: 'dev', 'prod', 'qa', 'canary', 'staging', 'private' (default: prod)")

	DeployCmd.Flags().String("cloud-provider", "", "Cloud provider (aws|gcp|azure)")
	DeployCmd.Flags().String("region", "", "Region code (e.g. us-east-2, us-central1)")
	DeployCmd.Flags().String("param", "", "Parameters for the instance deployment")
	DeployCmd.Flags().String("param-file", "", "Json file containing parameters for the instance deployment")
	// Additional flags from build command
	DeployCmd.Flags().Bool("skip-docker-build", false, "Skip building and pushing the Docker image")
	DeployCmd.Flags().StringArray("platforms", []string{"linux/amd64"}, "Specify the platforms to build for. Use the format: --platforms linux/amd64 --platforms linux/arm64. Default is linux/amd64.")
	
	if err := DeployCmd.MarkFlagFilename("param-file"); err != nil {
		return
	}

	err := DeployCmd.MarkFlagFilename("file")
	if err != nil {
		return
	}

}

var waitFlag bool

func runDeploy(cmd *cobra.Command, args []string) error {
	defer config.CleanupArgsAndFlags(cmd, &args)

	// Step 0: Validate user is logged in first
	token, err := common.GetTokenWithLogin()
	if err != nil {
		utils.PrintError(fmt.Errorf("Not logged in. Please run 'omctl login' to authenticate."))
		return err
	}

	// Retrieve flags
	file, err := cmd.Flags().GetString("file")
	if err != nil {
		return err
	}


	// Check if file was explicitly provided
	fileExplicit := cmd.Flags().Changed("file")

	// Get service name for further validation
	productName, err := cmd.Flags().GetString("product-name")
	if err != nil {
		utils.PrintError(fmt.Errorf("failed to check existing service: %v", err))
		return err
	}
	
	

	// Get cloud provider account flags
	cloudProvider, err := cmd.Flags().GetString("cloud-provider"); 
	if err != nil {
		return err
	}
	region, err := cmd.Flags().GetString("region"); 
	if err != nil {
		return err
	}


	skipDockerBuild, err := cmd.Flags().GetBool("skip-docker-build")
	if err != nil {
		return err
	}

	
	platforms, err := cmd.Flags().GetStringArray("platforms")
	if err != nil {
		return err
	}


	// Get dry-run flags
	dryRun, err := cmd.Flags().GetBool("dry-run")
	if err != nil {
		return err
	}




	// Get instance-id flag value
	instanceID, err := cmd.Flags().GetString("instance-id")
	if err != nil {
		return err
	}


	// Get resource-id flag value
	resourceID, err := cmd.Flags().GetString("resource-id")
	if err != nil {
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


	// Get env type and name flag value
	environmentType, err := cmd.Flags().GetString("environment-type")
	if err != nil {
		return err
	}

	environment, err := cmd.Flags().GetString("environment")
	if err != nil {
		utils.PrintError(err)
		return err
	}



	// Initialize spinner manager
	sm := ysmrr.NewSpinnerManager()
	sm.Start()
	defer sm.Stop()

	// Inform user of deployment start
	spinner := sm.AddSpinner("Starting deployment process...")


	// Improved spec file detection: prefer service plan, then docker compose, else repo
	var specFile string
	var specType string =  build.DockerComposeSpecType
	var deploymentType string = "hosted" // Default to hosted deployment


	
	// 1. If user provided a file via --file or arg, use it
	if fileExplicit && file != "" {
		specFile = file
	} else if len(args) > 0 && args[0] != "" {
		specFile = args[0]
	} else {
		// 2. Check for service plan spec (plan.yaml or serviceplan.yaml)
		for _, fname := range []string{"plan.yaml", "serviceplan.yaml"} {
			if _, err := os.Stat(fname); err == nil {
				specFile = fname
				break
			}
		}
		// 3. If not found, check for docker-compose.yaml
		if specFile == "" {
			if _, err := os.Stat("docker-compose.yaml"); err == nil {
				specFile = "docker-compose.yaml"
			}
		}
	}


	// Convert to absolute path if using spec file
	var absSpecFile string
	var processedData []byte
	if specFile != "" {
		absSpecFile, err = filepath.Abs(specFile)
		if err != nil {
			return errors.Wrap(err, "failed to get absolute path for spec file")
		}

		// Check if spec file exists
		if _, err := os.Stat(absSpecFile); os.IsNotExist(err) {
			return errors.Errorf("spec file does not exist: %s", absSpecFile)
		}

		// Read and process spec file for pre-checks
		fileData, err := os.ReadFile(absSpecFile)
		if err != nil {
			return errors.Wrap(err, "failed to read spec file")
		}

		
		// Check for plan spec indicators in processed YAML
		var planCheck map[string]interface{}
		if err := yaml.Unmarshal(fileData, &planCheck); err == nil {
			if _, ok := planCheck["helm"]; ok {
				specType = build.ServicePlanSpecType
			} else if _, ok := planCheck["operator"]; ok {
				specType = build.ServicePlanSpecType
			} else if _, ok := planCheck["terraform"]; ok {
				specType = build.ServicePlanSpecType
			} else if _, ok := planCheck["kustomize"]; ok {
				specType = build.ServicePlanSpecType
			}
		}
	}

	spinner.UpdateMessage("Checking cloud provider accounts...\n")

	awsAccountID, gcpProjectID, gcpProjectNumber, azureSubscriptionID, azureTenantID := extractCloudAccountsFromProcessedData(processedData)
		
		
		

     if cloudProvider == "" {
       if awsAccountID != ""  {
	       cloudProvider = "aws"
       } else if gcpProjectID != "" {
	       cloudProvider = "gcp"
       } else if azureSubscriptionID != "" {
	       cloudProvider = "azure"
       }
	}
	  
		   // If no cloud provider is set, assume all providers are available
	  allCloudProviders := []string{"aws", "gcp", "azure"}
	 

		allAccounts := []*openapiclient.DescribeAccountConfigResult{}
		// Filter for READY accounts and collect status information
		readyAccounts := []*openapiclient.DescribeAccountConfigResult{}
		accountStatusSummary := make(map[string]int)
		var foundMatchingAccount bool
		var accountStatus string

	   for _, cp := range allCloudProviders {
		   // Pre-check 1: Check for linked cloud provider accounts
		   accounts, err := dataaccess.ListAccounts(cmd.Context(), token, cp)
		   if err != nil {
			   spinner.UpdateMessage(fmt.Sprintf("failed to check cloud provider accounts: %v", err))
			   spinner.Error()
			   return err
		   }
		   for _, acc := range accounts.AccountConfigs {
			   allAccounts = append(allAccounts, &acc)
			   if acc.Status == "READY" {
				   readyAccounts = append(readyAccounts, &acc)
				   if awsAccountID == "" && acc.AwsAccountID != nil {
						   awsAccountID = *acc.AwsAccountID
				   }
				   if gcpProjectID == "" && acc.GcpProjectID != nil{
					gcpProjectID = *acc.GcpProjectID
					gcpProjectNumber = *acc.GcpProjectNumber
				   }
				     if azureSubscriptionID == "" && acc.AzureSubscriptionID != nil {
					azureSubscriptionID = *acc.AzureSubscriptionID
					azureTenantID = *acc.AzureTenantID
				   }
			   }
			   accountStatusSummary[acc.Status]++
			   if awsAccountID != "" && acc.AwsAccountID != nil {
				   if *acc.AwsAccountID == awsAccountID {
					   foundMatchingAccount = true
					   accountStatus = acc.Status
					   break
				   }
			   }
			   if gcpProjectID != "" && acc.GcpProjectID != nil {
				   if *acc.GcpProjectID == gcpProjectID && *acc.GcpProjectNumber == gcpProjectNumber {
					   foundMatchingAccount = true
					   accountStatus = acc.Status
					   break
				   }
			   }
			   if azureSubscriptionID != "" && acc.AzureSubscriptionID != nil {
				   if *acc.AzureSubscriptionID == azureSubscriptionID && *acc.AzureTenantID == azureTenantID {
					   foundMatchingAccount = true
					   accountStatus = acc.Status
					   break
				   }
			   }
		   }
	   }

		if !foundMatchingAccount {
				spinner.UpdateMessage("Error: Specified cloud account is not linked to your organization.")
				spinner.Error()
				if awsAccountID != "" {
					fmt.Printf("AWS account ID %s is not linked. Please link it using 'omctl account create'.\n", awsAccountID)
				}
				if gcpProjectID != "" {
					fmt.Printf("GCP project %s/%s is not linked. Please link it using 'omctl account create'.\n", gcpProjectID, gcpProjectNumber)
				}
				if azureSubscriptionID != "" {
					fmt.Printf("Azure subscription %s/%s is not linked. Please link it using 'omctl account create'.\n", azureSubscriptionID, azureTenantID)
				}
			} else if accountStatus != "READY" {
				spinner.UpdateMessage("Error: Specified cloud account is not READY.")
				spinner.Error()
				if awsAccountID != "" {
					fmt.Printf("AWS account ID %s is linked but has status '%s'. Complete onboarding if required.\n", awsAccountID, accountStatus)
				}
				if gcpProjectID != "" {
					fmt.Printf("GCP project %s/%s is linked but has status '%s'. Complete onboarding if required.\n", gcpProjectID, gcpProjectNumber, accountStatus)
				}
				if azureSubscriptionID != "" {
					fmt.Printf("Azure subscription %s/%s is linked but has status '%s'. Complete onboarding if required.\n", azureSubscriptionID, azureTenantID, accountStatus)
				}
			}

	if awsAccountID == "" && gcpProjectID == "" && azureSubscriptionID == "" {
	
	// Ensure at least one READY account is available
	if len(readyAccounts) == 0 {
		if len(allAccounts) > 0 {
			utils.PrintError(fmt.Errorf(
				"\nNo READY accounts found. Account setup required:\n"+
					"   Your organization has %d accounts, but none are in READY status.\n"+
					"   Non-READY accounts may need to complete onboarding or have configuration issues.\n"+
					"\n💡 Next steps:\n"+
					"   1. Check existing account status: omctl account list\n"+
					"   2. Complete onboarding for existing accounts, or\n"+
					"   3. Create a new READY account: omctl account create\n",
				len(allAccounts),
			))
			   spinner.UpdateMessage(" deployment requires at least one READY cloud provider account")
			   spinner.Error()
			   return nil
		} else {
			utils.PrintError(fmt.Errorf(
				"\nNo cloud provider accounts found.\n"+
					"💡 Create your first account: omctl account create\n",
			))
	spinner.UpdateMessage(" no cloud provider accounts linked. Please link at least one account using 'omctl account create' before deploying")
	spinner.Error()
	return nil
    }
}

}
	spinner.UpdateMessage("Specified account is linked and READY")
	spinner.Complete()


	spinner = sm.AddSpinner("Determining service name")

       var serviceNameToUse string
       serviceNameToUse = productName
       if serviceNameToUse == "" {
			if  specType != "" {
				// Use current directory name for repository-based builds
				cwd, err := os.Getwd()
				if err != nil {
					return err
				}
				serviceNameToUse = sanitizeServiceName(filepath.Base(cwd))
			} else {
				// Use directory name from spec file path
				serviceNameToUse = sanitizeServiceName(filepath.Base(filepath.Dir(absSpecFile)))
			}
			if serviceNameToUse == "." || serviceNameToUse == "/" || serviceNameToUse == "" {
				serviceNameToUse = "ctl"
			}
			
       }

	spinner.UpdateMessage(fmt.Sprintf("Determining service name: %s", serviceNameToUse))
	spinner.Complete()

	// Pre-check 3: Check if service exists and validate service plan count
	spinner.UpdateMessage(fmt.Sprintf("Checking existing service... %s", serviceNameToUse))
	spinner.Complete()
	existingServiceID,  err := findExistingService(cmd.Context(), token, serviceNameToUse)
	if err != nil {
		spinner.UpdateMessage(fmt.Sprintf("Error: failed to check existing service: %w", err))
		spinner.Error()
		return nil
	}

	if existingServiceID != "" {
		spinner.UpdateMessage(fmt.Sprintf("Checking existing service (ID: %s)\n", existingServiceID))
		spinner.Complete()
	} else {
		spinner.UpdateMessage("(new service)")
		spinner.Complete()
	}



	// Step 3: Build service in DEV environment with release-as-preferred
	spinner = sm.AddSpinner("Building service")


	var serviceID, environmentID, planID string
	var undefinedResources map[string]string

	if specType == build.DockerComposeSpecType && !skipDockerBuild {
		serviceID, environmentID, planID, undefinedResources, err = build.BuildServiceFromRepository(
		cmd,
		cmd.Context(),
		token,
		serviceNameToUse,
		"",
		false,
		dryRun,
		skipDockerBuild,
		false,
		false,
		false,
		deploymentType,
		awsAccountID,
		gcpProjectID,
		gcpProjectNumber,
		sm,
		file,
		[]string{},
		platforms,
	)
	} else {

		serviceID, environmentID, planID, undefinedResources, err = build.BuildService(
			cmd.Context(),
			processedData,
			token,
			serviceNameToUse,
			specType,
			nil,
			nil,
			&environment,
			 &environmentType,
			true,
			true,
			nil,
			dryRun,
		)

	}
	if err != nil {
			utils.HandleSpinnerError(spinner, sm, err)
			return err
	}
	



	// Dry-run exit point
	if dryRun {
		spinner.UpdateMessage("Simulated service build completed successfully (dry run)")
		spinner.Complete()
		fmt.Println("🔍 Dry-run mode: Validation checks only. No deployment will be performed.")
		if token == "" {
			fmt.Println("❌ Not logged in. Please run 'omctl login' to authenticate.")
			return fmt.Errorf("user not logged in")
		}
		fmt.Println("✅ Login check passed.")
		fmt.Println("✅ All other validations passed.")
		fmt.Println("To proceed with actual deployment, run the command without --dry-run flag.")
		return nil
	}
	fmt.Println()

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

	// Execute post-service-build deployment workflow
	err = executeDeploymentWorkflow(cmd, sm, token, serviceID, environmentID, planID, serviceNameToUse, environment, environmentType, instanceID, cloudProvider, region, param, paramFile, resourceID)
	if err != nil {
		return err
	}

	return nil
}


// executeDeploymentWorkflow handles the complete post-service-build deployment workflow
// This function is reusable for both deploy and build_simple commands
func executeDeploymentWorkflow(cmd *cobra.Command, sm ysmrr.SpinnerManager, token, serviceID, environmentID, planID, serviceName, environment, environmentType, instanceID, cloudProvider, region, param, paramFile, resourceID string) error {


	// Step 7: Set service plan as preferred in environment
	spinner := sm.AddSpinner(fmt.Sprintf("Setting service plan as preferred in %s", environment))
	
	// Find the latest version of the environment plan
	targetVersion, err := dataaccess.FindLatestVersion(cmd.Context(), token, serviceID, planID)
	if err != nil {
		utils.HandleSpinnerError(spinner, sm, err)
		return err
	}

	// Set as preferred
	_, err = dataaccess.SetDefaultServicePlan(cmd.Context(), token, serviceID, planID, targetVersion)
	if err != nil {
		utils.HandleSpinnerError(spinner, sm, err)
		return err
	}
	spinner.UpdateMessage(fmt.Sprintf("Setting service plan as preferred in %s: Success",environment))
	spinner.Complete()

	


	// Step 9: Create or upgrade instance deployment automatically
	var finalInstanceID string
	var instanceActionType string = "create"


	spinnerMsg := "Checking for existing instances"
	spinner = sm.AddSpinner(spinnerMsg)

	var existingInstanceIDs []string
	existingInstanceIDs, err = listInstances(cmd.Context(), token, serviceID, environmentID, planID, instanceID)
	if err != nil {
		spinner.UpdateMessage(spinnerMsg + ": Failed (" + err.Error() + ")")
		spinner.Error()
		existingInstanceIDs = []string{} // Reset to create new instance
		sm.Stop()
	}

	// Display automatic instance handling message
	if len(existingInstanceIDs) > 1 {
		spinner.UpdateMessage(fmt.Sprintf("%s: Found %d existing instances", spinnerMsg, len(existingInstanceIDs)))
		spinner.Complete()
		
		// Stop spinner manager temporarily to show the note
		sm.Stop()
		fmt.Printf("📝 Note: Instance upgrade is automatic.\n")
		fmt.Printf("   Existing Instances: %v\n", existingInstanceIDs)
		
		// Restart spinner manager
		sm = ysmrr.NewSpinnerManager()
		sm.Start()
	} else if len(existingInstanceIDs) == 1 {
		spinner.UpdateMessage(fmt.Sprintf("%s: Found 1 existing instance", spinnerMsg))
		spinner.Complete()
		
		// Stop spinner manager temporarily to show the note
		sm.Stop()
		fmt.Printf("📝 Note: Instance upgrade is automatic.\n")
		fmt.Printf("   Existing Instance: %v\n", existingInstanceIDs)
		
		// Restart spinner manager
		sm = ysmrr.NewSpinnerManager()
		sm.Start()
	} else {
		spinner.UpdateMessage(fmt.Sprintf("%s: No existing instances found", spinnerMsg))
		spinner.Complete()
		
		// Stop spinner manager temporarily to show the note
		sm.Stop()
		fmt.Printf("📝 Note: Instance creation is automatic.\n")
		
		// Restart spinner manager
		sm = ysmrr.NewSpinnerManager()
		sm.Start()
	}

	if len(existingInstanceIDs) > 0 {
		foundMsg := spinnerMsg + ": Found existing instance"
		spinner.UpdateMessage(foundMsg)
		spinner.Complete()

		spinner = sm.AddSpinner(fmt.Sprintf("Upgrading existing instance: %s", existingInstanceIDs))
		upgradeErr := upgradeExistingInstance(cmd.Context(), token, existingInstanceIDs, serviceID, planID)
		instanceActionType = "upgrade"
		if upgradeErr != nil {
			spinner.UpdateMessage(fmt.Sprintf("Upgrading existing instance: Failed (%s)", upgradeErr.Error()))
			spinner.Error()
			sm.Stop()
		} else {
			finalInstanceID = existingInstanceIDs[0]
			spinner.UpdateMessage(fmt.Sprintf("Upgrading existing instance: Success (ID: %s)", finalInstanceID))
			spinner.Complete()
		}
		
	} else {
		noFoundMsg := spinnerMsg + ": No existing instances found"
		spinner.UpdateMessage(noFoundMsg)
		spinner.Complete()

		createMsg := "Creating new instance deployment"
		
		spinner = sm.AddSpinner(createMsg)
		createdInstanceID, err := "", error(nil)
		createdInstanceID, err = createInstanceUnified(cmd.Context(), token, serviceID, environmentID, planID, cloudProvider, region, param, paramFile, resourceID)
		finalInstanceID = createdInstanceID  
		instanceActionType = "create"
		if err != nil {
			spinner.UpdateMessage(fmt.Sprintf("%s: Failed (%s)", createMsg, err.Error()))
			spinner.Error()
			sm.Stop()

		} else {
			spinner.UpdateMessage(fmt.Sprintf("%s: Success (ID: %s)", createMsg, finalInstanceID))
			spinner.Complete()
		}
	}



       // Step 10: Success message - completed deployment
       spinner = sm.AddSpinner("Deployment workflow completed")
       spinner.Complete()

       sm.Stop()

	   // Success message
	   fmt.Println()
	   fmt.Printf("   Service: %s (ID: %s)\n", serviceName, serviceID)
	   fmt.Printf("   Environment: %s, Environment Type: %s (ID: %s)\n", environment, environmentType , environmentID)
	   if finalInstanceID != "" {
		   fmt.Printf("   Instance: %s (ID: %s)\n", instanceActionType, finalInstanceID)
	   }
	   fmt.Println()
	   

	    // Optionally display workflow progress if desired (if you want to keep this logic, pass cmd/context as needed)
       if finalInstanceID != "" {
		   err = instance.DisplayWorkflowResourceDataWithSpinners(cmd.Context(), token, finalInstanceID, instanceActionType) // Use the correct package alias
	       if err != nil {
		       fmt.Printf("❌ Deployment failed-- %s\n", err)
	       } else {
		       fmt.Println("✅ Deployment successful")
	       }
       }
	   
	   return nil
}






// createInstanceUnified creates an instance with or without subscription, removing duplicate code
func createInstanceUnified(ctx context.Context, token, serviceID, environmentID, productTierID, cloudProvider, region, param, paramFile, resourceID string) (string, error) {
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

	   // Get list of resources in the target tier version
		resources, err := dataaccess.ListResources(ctx, token, serviceID, productTierID, &version)
		if err != nil {
			return "", fmt.Errorf("no resources found in service plan: %w", err)
		}

		if len(resources.Resources) == 0 {
			return "", fmt.Errorf("no resources found in service plan")
		}


		resourceKey := ""

		if resourceID != "" {
			for _, resource := range resources.Resources {
				if resource.Id == resourceID {
					resourceKey = resource.Key
					resourceID = resource.Id
					break
				}
			}
			if resourceKey == "" {
				fmt.Printf("⚠️ Warning: resource ID : %s not found in service plan", resourceID)
			}
		}

		if resourceKey == "" {
			if len(resources.Resources) == 1 {
				resourceKey = resources.Resources[0].Key
				resourceID = resources.Resources[0].Id
			}
			if len(resources.Resources) > 1 {
				fmt.Println("Multiple resources found in service plan. Please select one:")
				for idx, resource := range resources.Resources {
					fmt.Printf("  %d. Name: %s, ID: %s\n", idx+1, resource.Name, resource.Id)
				}
				var choice int
				for {
					fmt.Print("Enter the number of the resource to use: ")
					_, err := fmt.Scanln(&choice)
					if err == nil && choice > 0 && choice <= len(resources.Resources) {
						break
					}
					fmt.Println("Invalid selection. Please enter a valid number.")
				}
				selected := resources.Resources[choice-1]
				resourceKey = selected.Key
				resourceID = selected.Id
			}
		}



		// Format parameters
		formattedParams, err := common.FormatParams(param, paramFile)
		if err != nil {
			return "", err
		}


	   
	   if resourceID == "" || resourceKey == "" {
		   return "", fmt.Errorf("invalid resource in service plan")
	   }



	   // Select default cloudProvider and region from offering.CloudProviders if available


	   if len(offering.CloudProviders) > 0 {
		   found := false
		   for _, cp := range offering.CloudProviders {
			   if cp == cloudProvider {
				   found = true
				   break
			   }
		   }
		   if !found {
			   // fallback to first available provider
			   return "", fmt.Errorf("cloud provider '%s' is not supported for this service plan. Supported providers: %v", cloudProvider, offering.CloudProviders)
		   }
	   }

		
	   if cloudProvider != "" {
		   var regions []string
		   switch cloudProvider {
		   case "gcp":
			   regions = offering.GcpRegions
		   case "aws":
			   regions = offering.AwsRegions
		   case "azure":
			   regions = offering.AzureRegions
		   }
		   found := false
		   for _, rk := range regions {
			   if rk == region {
				   found = true
				   break
			   }
		   }
		   if !found && len(regions) > 0 {
			   return "", fmt.Errorf("region '%s' is not supported for cloud provider '%s'. Supported regions: %v", region, cloudProvider, regions)
		   }
	   }

	   if region == "" {
		  switch cloudProvider {
		   case "gcp":
			   region = "us-central1"
		   case "aws":
			region = "ap-south-1"
			//    region = "us-east-1"
		   case "azure":
			   region = "eastus2"
			// region = "eastus"
		   
		   }
	   }


		 // Create default parameters with common sensible defaults
        defaultParams := map[string]interface{}{}
		// Try to describe service offering resource - this is optional for parameter validation
		resApiParams, err := dataaccess.DescribeServiceOfferingResource(ctx, token, serviceID, resourceID, "none", productTierID, version)

		if err != nil {
			return "", fmt.Errorf("failed to describe service offering resource: %w", err)
		}
		
		// Extract CREATE verb parameters and set defaults
		if len(resApiParams.ConsumptionDescribeServiceOfferingResourceResult.Apis) > 0 {
			for _, apiSpec := range resApiParams.ConsumptionDescribeServiceOfferingResourceResult.Apis {
				if apiSpec.Verb == "CREATE" {
					
					for _, inputParam := range apiSpec.InputParameters {
						// Handle special system parameters
						switch inputParam.Key {
						case "subscriptionId","cloud_provider","region":
							continue
						default:
							// Handle custom parameters
							if inputParam.Required {
								if formattedParams[inputParam.Key] != nil {
									defaultParams[inputParam.Key] = formattedParams[inputParam.Key] 
								}else{
									defaultParams[inputParam.Key] = *inputParam.DefaultValue
									
								}
						}
						if inputParam.Required == false{
								if formattedParams[inputParam.Key] != nil {
									defaultParams[inputParam.Key] = formattedParams[inputParam.Key] 
								} else if inputParam.DefaultValue != nil {
									defaultParams[inputParam.Key] = *inputParam.DefaultValue
								}
						}
							
						}
					}
					break // Found CREATE verb, no need to continue
				}
			}
		}

		// Check for missing required parameters
		var defaultRequiredParams []string
		for k, v := range defaultParams {
			if v == nil || (reflect.TypeOf(v).Kind() == reflect.String && v == "") {
				defaultRequiredParams = append(defaultRequiredParams, k)
			}
		}

		
		// Validate that all required parameters have values
		if len(defaultRequiredParams) > 0 {
			return "", fmt.Errorf("missing required parameters for instance creation: %v. Please provide values using --param or --param-file flags", defaultRequiredParams)
		}

		// Check for unused parameters from formattedParams
		var unusedParams []string
		for paramKey := range formattedParams {
			if _, exists := defaultParams[paramKey]; !exists {
				unusedParams = append(unusedParams, paramKey)
			}
		}
		
		// Warn user about unused parameters
		if len(unusedParams) > 0 {
			fmt.Printf("⚠️  Warning: The following parameters were provided but are not supported by this service and won't be used:\n")
			for _, param := range unusedParams {
				fmt.Printf("   - %s\n", param)
			}
			
		}

		
		
	  
	   request := openapiclientfleet.FleetCreateResourceInstanceRequest2{
		   ProductTierVersion: &version,
		   CloudProvider:      &cloudProvider,
		   Region:             &region,
		   RequestParams:      defaultParams,
		   NetworkType:        nil,
	   }
      

	   fmt.Printf("🌐 Creating instance in %s ...\n", request)
	   return "", nil

       // Create the instance
    //    instance, err := dataaccess.CreateResourceInstance(ctx, token,
	//        res.ConsumptionDescribeServiceOfferingResult.ServiceProviderId,
	//        res.ConsumptionDescribeServiceOfferingResult.ServiceURLKey,
	//        offering.ServiceAPIVersion,
	//        offering.ServiceEnvironmentURLKey,
	//        offering.ServiceModelURLKey,
	//        offering.ProductTierURLKey,
	//        resourceKey,
	//        request)
    //    if err != nil {
	//        return "", fmt.Errorf("failed to create resource instance: %w", err)
    //    }

    //    if instance == nil || instance.Id == nil {
	//        return "", fmt.Errorf("instance creation returned empty result")
    //    }

    //    return *instance.Id, nil
}





// listInstances is a helper function for backward compatibility
func listInstances(ctx context.Context, token, serviceID, environmentID, servicePlanID, instanceID string) ([]string, error) {

	res, err := dataaccess.ListResourceInstance(ctx, token, serviceID, environmentID)
	if err != nil {
		return []string{}, fmt.Errorf("failed to search for instances: %w", err)
	}

	exitInstanceIDs := make([]string, 0)
	seenIDs := make(map[string]bool)
	if len(res.ResourceInstances) == 0 {
		return []string{}, nil
	}
	for _, instance := range res.ResourceInstances {
		var idStr string
		if instance.ConsumptionResourceInstanceResult.Id != nil {
			idStr = *instance.ConsumptionResourceInstanceResult.Id
		} else {
			idStr = "<nil>"
		}

		// Priority: instanceID  > servicePlanID
		if instanceID != "" && idStr == instanceID {
			if !seenIDs[idStr] {
				exitInstanceIDs = append(exitInstanceIDs, idStr)
				seenIDs[idStr] = true
			}
		} else if instanceID == "" && servicePlanID != "" && instance.ProductTierId == servicePlanID {
			if idStr != "" && !seenIDs[idStr] {
				exitInstanceIDs = append(exitInstanceIDs, idStr)
				seenIDs[idStr] = true
			}
		}
	}
	
	return exitInstanceIDs, nil
}


// upgradeExistingInstance upgrades an existing instance to the latest version
func upgradeExistingInstance(ctx context.Context, token string, instanceIDs []string, serviceID, productTierID string) error {
	// Get the latest version
	latestVersion, err := dataaccess.FindLatestVersion(ctx, token, serviceID, productTierID)
	if err != nil {
		return fmt.Errorf("failed to find latest version: %w", err)
	}

	// Get instance details to find environment ID
	for _, id := range instanceIDs {
		searchRes, err := dataaccess.SearchInventory(ctx, token, fmt.Sprintf("resourceinstance:%s", id))
		if err != nil {
			return fmt.Errorf("failed to find instance details for %s: %w", id, err)
		}
		if len(searchRes.ResourceInstanceResults) == 0 {
			return fmt.Errorf("instance not found: %s", id)
		}
		
		environmentID := searchRes.ResourceInstanceResults[0].ServiceEnvironmentId
		resourceOverrideConfig := make(map[string]openapiclientfleet.ResourceOneOffPatchConfigurationOverride)
		err = dataaccess.OneOffPatchResourceInstance(ctx, token,
			serviceID,
			environmentID,
			id,
			resourceOverrideConfig,
			latestVersion,
		)
		if err != nil {
			return fmt.Errorf("failed to upgrade instance %s: %w", id, err)
		}
	}

	return nil
}



// validateSpecFileConfiguration validates the spec file for  tenant-aware resources
func validateSpecFileConfiguration(data []byte, specType string) (tenantAwareResourceCount int, err error) {
	
	var spec map[string]interface{}
	if err := yaml.Unmarshal(data, &spec); err != nil {
		return 0, fmt.Errorf("failed to parse YAML: %w", err)
	}

	tenantAwareResourceCount = 0
	// If services section exists, use it
	if servicesRaw, ok := spec["services"]; ok {
		if services, ok := servicesRaw.(map[string]interface{}); ok {
			for _, svcRaw := range services {
				svc, ok := svcRaw.(map[string]interface{})
				if !ok {
					continue
				}
				val, hasKey := svc["x-omnistrate-mode-internal"]
				if hasKey {
					switch v := val.(type) {
					case bool:
						if v {
							continue
						}
					case string:
						if strings.EqualFold(v, "false") {
							continue
						}
					}
					tenantAwareResourceCount++
				} else {
					tenantAwareResourceCount++
				}
			}
			fmt.Println("tenantAwareResourceCount spec:", tenantAwareResourceCount)
			return tenantAwareResourceCount, nil
		}
	}
	for _, val := range spec {
		
		// Each top-level key is a resource
		res, ok := val.(map[string]interface{})
		if !ok {
			continue
		}
		tag, hasTag := res["x-omnistrate-mode-internal"]
		fmt.Println("Resource:", res)
		fmt.Println("Tag:", tag, "HasTag:", hasTag)
		if hasTag {
			switch v := tag.(type) {
			case bool:
				if v {
					continue
				}
			case string:
				if strings.EqualFold(v, "true") {
					continue
				}
			}
			tenantAwareResourceCount++
		} else {
			tenantAwareResourceCount++
		}
	}
	fmt.Println("tenantAwareResourceCount spec:", tenantAwareResourceCount)
	return tenantAwareResourceCount, nil
	}


// findExistingService searches for an existing service by name
func findExistingService(ctx context.Context, token, serviceName string) (string, error) {
	services, err := dataaccess.ListServices(ctx, token)
	if err != nil {
		return "", fmt.Errorf("failed to list services: %w", err)
	}

	for _, service := range services.Services {
		if service.Name == serviceName {
			
			return service.Id, nil
		}
	}

	return "", nil // Service not found
}

// getServicePlans retrieves service plans for a given service
func getServicePlans(ctx context.Context, token string, envs map[string]interface{}) ([]interface{}, error) {
	env := envs["DEV"]
	if env == nil {
		env = envs["dev"]
	}
	if env == nil {
		return nil, fmt.Errorf("no DEV environment found for service")
	}

	// Use reflection to handle both struct and pointer-to-struct
	val := reflect.ValueOf(env)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}
	if val.Kind() == reflect.Struct {
		field := val.FieldByName("ServicePlans")
		if field.IsValid() && field.CanInterface() {
			plansVal := field
			if plansVal.Kind() == reflect.Ptr {
				plansVal = plansVal.Elem()
			}
			if plansVal.Kind() == reflect.Slice {
				var allPlans []interface{}
				for i := 0; i < plansVal.Len(); i++ {
					allPlans = append(allPlans, plansVal.Index(i).Interface())
				}
				return allPlans, nil
			}
		}
	} else if val.Kind() == reflect.Map {
		// If ServicePlans is present as a key in a map
		plansVal := val.MapIndex(reflect.ValueOf("ServicePlans"))
		if plansVal.IsValid() && plansVal.Kind() == reflect.Slice {
			var allPlans []interface{}
			for i := 0; i < plansVal.Len(); i++ {
				allPlans = append(allPlans, plansVal.Index(i).Interface())
			}
			return allPlans, nil
		}
	}
	return nil, fmt.Errorf("could not extract ServicePlans from environment")
}


// createDeploymentYAML generates a YAML document for deployment based on modelType, creationMethod, and cloud account flags
// Returns a map[string]interface{} representing the YAML structure
func createDeploymentYAML(
	deploymentType string,
	specType string,
	awsAccountID string,
	awsBootstrapRoleARN string,
	gcpProjectID string,
	gcpProjectNumber string,
	gcpServiceAccountEmail string,
	azureSubscriptionID string,
	azureTenantID string,
) map[string]interface{} {
	if deploymentType == "" {
		deploymentType = "hosted"
	}
	
       yamlDoc := make(map[string]interface{})

	if awsBootstrapRoleARN == "" && awsAccountID != "" {
		// Default role ARN if not provided
		awsBootstrapRoleARN = fmt.Sprintf("arn:aws:iam::%s:role/OmnistrateBootstrapRole", awsAccountID)
	}

	if gcpServiceAccountEmail == "" && gcpProjectID != "" {
		// Default service account email if not provided
		gcpServiceAccountEmail = fmt.Sprintf("omnistrate-bootstrap@%s.iam.gserviceaccount.com", gcpProjectID)
	}

	

	// Build the deployment section based on deploymentType and specType

	if deploymentType == "byoa" {
		if specType != "DockerCompose"  {
			yamlDoc["deployment"] = map[string]interface{}{
				"byoaDeployment": map[string]interface{}{
					"awsAccountID": awsAccountID,
					"awsBootstrapRoleAccountArn": awsBootstrapRoleARN,
				},
			}
		} else {
			yamlDoc["x-omnistrate-byoa"] = map[string]interface{}{
				"awsAccountID": awsAccountID,
				"awsBootstrapRoleAccountArn": awsBootstrapRoleARN,
			}
		}
	} else if deploymentType == "hosted" {
		if specType != "DockerCompose" {
			hostedDeployment := make(map[string]interface{})
			if awsAccountID != "" {
				hostedDeployment["awsAccountID"] = awsAccountID
				if awsBootstrapRoleARN != "" {
					hostedDeployment["awsBootstrapRoleAccountArn"] = awsBootstrapRoleARN
				}
			}
			if gcpProjectID != "" {
				hostedDeployment["gcpProjectID"] = gcpProjectID
				if gcpProjectNumber != "" {
					hostedDeployment["gcpProjectNumber"] = gcpProjectNumber
				}
				if gcpServiceAccountEmail != "" {
					hostedDeployment["gcpServiceAccountEmail"] = gcpServiceAccountEmail
				}
			}
			if azureSubscriptionID != "" {
				hostedDeployment["azureSubscriptionID"] = azureSubscriptionID
				if azureTenantID != "" {
					hostedDeployment["azureTenantID"] = azureTenantID
				}
			}
			yamlDoc["deployment"] = map[string]interface{}{
				"hostedDeployment": hostedDeployment,
			}
		} else {
			myAccount := make(map[string]interface{})
			if awsAccountID != "" {
				myAccount["awsAccountID"] = awsAccountID
				if awsBootstrapRoleARN != "" {
					myAccount["awsBootstrapRoleAccountArn"] = awsBootstrapRoleARN
				}
			}
			if gcpProjectID != "" {
				myAccount["gcpProjectID"] = gcpProjectID
				if gcpProjectNumber != "" {
					myAccount["gcpProjectNumber"] = gcpProjectNumber
				}
				if gcpServiceAccountEmail != "" {
					myAccount["gcpServiceAccountEmail"] = gcpServiceAccountEmail
				}
			}
			if azureSubscriptionID != "" {
				myAccount["azureSubscriptionID"] = azureSubscriptionID
				if azureTenantID != "" {
					myAccount["azureTenantID"] = azureTenantID
				}
			}
			yamlDoc["x-omnistrate-my-account"] = myAccount
		}
	}
	return yamlDoc
}


// extractCloudAccountsFromProcessedData extracts cloud provider account information from the YAML content
func extractCloudAccountsFromProcessedData(processedData []byte) (awsAccountID, gcpProjectID, gcpProjectNumber, azureSubscriptionID, azureTenantID string) {
	if len(processedData) == 0 {
		return "", "", "", "", ""
	}

	// Split YAML into multiple documents and parse each one
	yamlDocs := strings.Split(string(processedData), "---")
	
	for _, docStr := range yamlDocs {
		docStr = strings.TrimSpace(docStr)
		if docStr == "" {
			continue
		}
		
		var yamlContent map[string]interface{}
		if err := yaml.Unmarshal([]byte(docStr), &yamlContent); err != nil {
			continue // Skip invalid YAML documents
		}

		// Check for deployment section (ServicePlan spec format)
		if deployment, exists := yamlContent["deployment"]; exists {
			if deploymentMap, ok := deployment.(map[string]interface{}); ok {
				// Check byoaDeployment
				if byoa, exists := deploymentMap["byoaDeployment"]; exists {
					if byoaMap, ok := byoa.(map[string]interface{}); ok {
						if aws, exists := byoaMap["awsAccountID"]; exists {
							if awsStr, ok := aws.(string); ok && awsAccountID == "" {
								awsAccountID = awsStr
							}
						}
					}
				}
				
				// Check hostedDeployment
				if hosted, exists := deploymentMap["hostedDeployment"]; exists {
					if hostedMap, ok := hosted.(map[string]interface{}); ok {
						if aws, exists := hostedMap["awsAccountID"]; exists {
							if awsStr, ok := aws.(string); ok && awsAccountID == "" {
								awsAccountID = awsStr
							}
						}
						if gcp, exists := hostedMap["gcpProjectID"]; exists {
							if gcpStr, ok := gcp.(string); ok && gcpProjectID == "" {
								gcpProjectID = gcpStr
							}
						}
						if gcpNum, exists := hostedMap["gcpProjectNumber"]; exists {
							if gcpNumStr, ok := gcpNum.(string); ok && gcpProjectNumber == "" {
								gcpProjectNumber = gcpNumStr
							}
						}
						if azure, exists := hostedMap["azureSubscriptionID"]; exists {
							if azureStr, ok := azure.(string); ok && azureSubscriptionID == "" {
								azureSubscriptionID = azureStr
							}
						}
						if azureTenant, exists := hostedMap["azureTenantID"]; exists {
							if azureTenantStr, ok := azureTenant.(string); ok && azureTenantID == "" {
								azureTenantID = azureTenantStr
							}
						}
					}
				}
			}
		}

		// Check for Docker Compose format (x-omnistrate-* sections)
		if byoa, exists := yamlContent["x-omnistrate-byoa"]; exists {
			if byoaMap, ok := byoa.(map[string]interface{}); ok {
				if aws, exists := byoaMap["awsAccountID"]; exists {
					if awsStr, ok := aws.(string); ok && awsAccountID == "" {
						awsAccountID = awsStr
					}
				}
			}
		}

		if myAccount, exists := yamlContent["x-omnistrate-my-account"]; exists {
			if myAccountMap, ok := myAccount.(map[string]interface{}); ok {
				if aws, exists := myAccountMap["awsAccountID"]; exists {
					if awsStr, ok := aws.(string); ok && awsAccountID == "" {
						awsAccountID = awsStr
					}
				}
				if gcp, exists := myAccountMap["gcpProjectID"]; exists {
					if gcpStr, ok := gcp.(string); ok && gcpProjectID == "" {
						gcpProjectID = gcpStr
					}
				}
				if gcpNum, exists := myAccountMap["gcpProjectNumber"]; exists {
					if gcpNumStr, ok := gcpNum.(string); ok && gcpProjectNumber == "" {
						gcpProjectNumber = gcpNumStr
					}
				}
				if azure, exists := myAccountMap["azureSubscriptionID"]; exists {
					if azureStr, ok := azure.(string); ok && azureSubscriptionID == "" {
						azureSubscriptionID = azureStr
					}
				}
				if azureTenant, exists := myAccountMap["azureTenantID"]; exists {
					if azureTenantStr, ok := azureTenant.(string); ok && azureTenantID == "" {
						azureTenantID = azureTenantStr
					}
				}
			}
		}
	}

	return awsAccountID, gcpProjectID, gcpProjectNumber, azureSubscriptionID, azureTenantID
}

// sanitizeServiceName converts a service name to be API-compatible (lowercase, valid characters)
func sanitizeServiceName(name string) string {
	if name == "" {
		return ""
	}
	
	// Convert to lowercase to match API pattern ^[a-z0-9][a-z0-9_-]*$
	name = strings.ToLower(name)
	
	// Replace any invalid characters with hyphens
	re := regexp.MustCompile(`[^a-z0-9_-]`)
	name = re.ReplaceAllString(name, "-")
	
	// Remove leading hyphens/underscores if any
	name = regexp.MustCompile(`^[-_]+`).ReplaceAllString(name, "")
	
	// Ensure it starts with alphanumeric if it doesn't already
	if len(name) > 0 && !regexp.MustCompile(`^[a-z0-9]`).MatchString(name) {
		name = "svc-" + name
	}
	
	// Remove trailing hyphens/underscores
	name = regexp.MustCompile(`[-_]+$`).ReplaceAllString(name, "")
	
	return name
}

