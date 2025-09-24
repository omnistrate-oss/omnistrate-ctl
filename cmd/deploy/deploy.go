package deploy

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/omnistrate-oss/omnistrate-ctl/cmd/build"
	"github.com/omnistrate-oss/omnistrate-ctl/cmd/common"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/config"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/dataaccess"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/utils"

	"github.com/chelnak/ysmrr"
	"github.com/compose-spec/compose-go/loader"
	"github.com/compose-spec/compose-go/types"
	openapiclientfleet "github.com/omnistrate-oss/omnistrate-sdk-go/fleet"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	instancecmd "github.com/omnistrate-oss/omnistrate-ctl/cmd/instance"
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
	DeployCmd.Flags().String("product-name", "", "Specify a custom service name. If not provided, directory name will be used.")
	DeployCmd.Flags().String("release-description", "", "Release description for the version")
	DeployCmd.Flags().String("subscription-name", "", "Subscription name for service subscription")
	DeployCmd.Flags().Bool("dry-run", false, "Perform validation checks without actually deploying")
	DeployCmd.Flags().String("aws-account-id", "", "AWS account ID for BYOA or hosted deployment")
	DeployCmd.Flags().String("aws-bootstrap-role-arn", "", "AWS bootstrap role ARN for BYOA or hosted deployment")
	DeployCmd.Flags().String("gcp-project-id", "", "GCP project ID for BYOA or hosted deployment. Must be used with --gcp-project-number")
	DeployCmd.Flags().String("gcp-project-number", "", "GCP project number for BYOA or hosted deployment. Must be used with --gcp-project-id")
	DeployCmd.Flags().String("gcp-service-account-email", "", "GCP service account email for BYOA or hosted deployment")
	DeployCmd.Flags().String("azure-subscription-id", "", "Azure subscription ID for BYOA or hosted deployment")
	DeployCmd.Flags().String("azure-tenant-id", "", "Azure tenant ID for BYOA or hosted deployment")
	DeployCmd.Flags().String("deployment-type", "", "Deployment type: hosted  or byoa")
	DeployCmd.Flags().String("service-plan-id", "", "Specify the service plan ID to use when multiple plans exist")
	DeployCmd.Flags().StringP("spec-type", "s", DockerComposeSpecType, "Spec type")
	DeployCmd.Flags().Bool("wait", false, "Wait for deployment to complete before returning.")
	DeployCmd.Flags().String("instance-id", "", "Specify the instance ID to use when multiple deployments exist.")
	// Additional flags from build command
	DeployCmd.Flags().StringArray("env-var", nil, "Specify environment variables required for running the image. Use the format: --env-var key1=var1 --env-var key2=var2. Only effective when no compose spec exists in the repo.")
	DeployCmd.Flags().Bool("skip-docker-build", false, "Skip building and pushing the Docker image")
	DeployCmd.Flags().Bool("skip-service-build", false, "Skip building the service from the compose spec")
	DeployCmd.Flags().Bool("skip-environment-promotion", false, "Skip creating and promoting to the production environment")
	DeployCmd.Flags().Bool("skip-saas-portal-init", false, "Skip initializing the SaaS Portal")
	DeployCmd.Flags().StringArray("platforms", []string{"linux/amd64"}, "Specify the platforms to build for. Use the format: --platforms linux/amd64 --platforms linux/arm64. Default is linux/amd64.")
	DeployCmd.Flags().Bool("reset-pat", false, "Reset the GitHub Personal Access Token (PAT) for the current user.")
}

var waitFlag bool

func runDeploy(cmd *cobra.Command, args []string) error {
	// Extract additional cloud provider flags for YAML creation
	awsBootstrapRoleARN, err := cmd.Flags().GetString("aws-bootstrap-role-arn")
	if err != nil {
		return err
	}
	gcpServiceAccountEmail, err := cmd.Flags().GetString("gcp-service-account-email")
	if err != nil {
		return err
	}
	// Get Azure account flags
	azureSubscriptionID, err := cmd.Flags().GetString("azure-subscription-id")
	if err != nil {
		return err
	}
	 azureTenantID, err := cmd.Flags().GetString("azure-tenant-id")
	   if err != nil {
		   return err
	   }
	// _, err = cmd.Flags().GetString("azure-tenant-id")
	defer config.CleanupArgsAndFlags(cmd, &args)

	// Step 0: Validate user is logged in first
	token, err := common.GetTokenWithLogin()
	if err != nil {
		return err
	}

	// Get cloud provider account flags
	deploymentType, err := cmd.Flags().GetString("deployment-type")
	if err != nil {
		return err
	}

	awsAccountID, err := cmd.Flags().GetString("aws-account-id")
	if err != nil {
		return err
	}

	gcpProjectID, err := cmd.Flags().GetString("gcp-project-id")
	if err != nil {
		return err
	}

	gcpProjectNumber, err := cmd.Flags().GetString("gcp-project-number")
	if err != nil {
		return err
	}

	// Get additional build flags
	envVars, err := cmd.Flags().GetStringArray("env-var")
	if err != nil {
		return err
	}

	skipDockerBuild, err := cmd.Flags().GetBool("skip-docker-build")
	if err != nil {
		return err
	}

	skipServiceBuild, err := cmd.Flags().GetBool("skip-service-build")
	if err != nil {
		return err
	}

	skipEnvironmentPromotion, err := cmd.Flags().GetBool("skip-environment-promotion")
	if err != nil {
		return err
	}

	skipSaasPortalInit, err := cmd.Flags().GetBool("skip-saas-portal-init")
	if err != nil {
		return err
	}

	platforms, err := cmd.Flags().GetStringArray("platforms")
	if err != nil {
		return err
	}

	resetPAT, err := cmd.Flags().GetBool("reset-pat")
	if err != nil {
		return err
	}

	
	if deploymentType == "byoa" || deploymentType == "hosted" {
		if awsAccountID == "" && gcpProjectID == "" && azureSubscriptionID == "" {
			return errors.New("BYOA deployment type requires either --aws-account-id, --gcp-project-id or --azure-subscription-id to be specified")
		}
		if gcpProjectID != "" && gcpProjectNumber == "" {
			return errors.New("GCP project number is required with GCP project ID")
		}
		if gcpProjectID == "" && gcpProjectNumber != "" {
			return errors.New("GCP project ID is required with GCP project number")
		}
		if azureSubscriptionID == "" && azureTenantID != "" {
			return errors.New("Azure subscription ID is required with Azure tenant ID")
		}
	}

	// Pre-checks: Validate environment and requirements
	fmt.Println("Running pre-deployment checks...")
	
	// Get the spec file path - follow the same flow as build_from_repo.go
	var specFile string
	var useRepo bool
	if len(args) > 0 {
		specFile = args[0]
	} else {
		// Look for compose.yaml in current directory first
		if _, err := os.Stat("compose.yaml"); err == nil {
			specFile = "compose.yaml"
		} else {
			// No spec file found - ask user if they want to auto-generate one
			fmt.Print("No spec file found, do you want to auto-generate one (Y/N): ")
			var response string
			fmt.Scanln(&response)
			
			response = strings.ToLower(strings.TrimSpace(response))
			if response == "y" || response == "yes" {
				// Check if we're in a git repository for auto-generation
				cwd, err := os.Getwd()
				if err != nil {
					return errors.Wrap(err, "failed to get current working directory")
				}
				
				if _, err := os.Stat(filepath.Join(cwd, ".git")); err == nil {
					// We're in a git repository - use repository-based build
					useRepo = true
					fmt.Println("Using repository-based build...")
				} else {
					return errors.New("auto-generation requires a git repository. Please initialize git repository or provide a spec file")
				}
			} else {
				return errors.New("Run deploy command with [--file=file] [--spec-type=spec-type] arguments")
			}
		}
	}

       // Check if spec-type was explicitly provided
       specTypeExplicit := cmd.Flags().Changed("spec-type")

       // Get instance-id flag value
       instanceID, err := cmd.Flags().GetString("instance-id")
       if err != nil {
	       return err
       }

	// Convert to absolute path if using spec file
	var absSpecFile string
	var processedData []byte
	var specType string
	if !useRepo {
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

		// Process template expressions recursively
		processedData, err = processTemplateExpressions(fileData, filepath.Dir(absSpecFile))
		if err != nil {
			return errors.Wrap(err, "failed to process template expressions")
		}

	       // Determine spec type
	       if !specTypeExplicit {
		       // If the file is named compose.yaml or docker-compose.yaml, default to DockerComposeSpecType
		       baseName := filepath.Base(absSpecFile)
		       if baseName == "compose.yaml" || baseName == "docker-compose.yaml" {
			       specType = build.DockerComposeSpecType // Use the correct constant value
				  
		       } else {
				
			       // If not, require the user to provide --spec-type flag
			       return errors.New("Please provide the --spec-type flag (docker-compose or service-plan) when using a custom spec file name.")
		       }
			  
	       } else {
		       specType, err = determineSpecType(processedData)
		       if err != nil {
			       return errors.Wrap(err, "failed to determine spec type")
		       }
	       }
				   var yamlMap map[string]interface{}
				   if awsAccountID != "" || gcpProjectID != "" || azureSubscriptionID != "" {
					   // Use createDeploymentYAML to generate a YAML map, then marshal to []byte for processedData
					   yamlMap = createDeploymentYAML(
						   deploymentType, // modelType
						   specType,       // creationMethod (may need to adjust if you have a separate creationMethod variable)
						   awsAccountID,
						   awsBootstrapRoleARN,
						   gcpProjectID,
						   gcpProjectNumber,
						   gcpServiceAccountEmail,
						   azureSubscriptionID,
						   azureTenantID,
					   )
					   yamlBytes, err := yaml.Marshal(yamlMap)
					   if err != nil {
						   return errors.Wrap(err, "failed to marshal deployment YAML")
					   }
					   // If processedData already has content, append yamlBytes to it with a separator
					   if len(processedData) > 0 {
						   processedData = append(processedData, []byte("\n---\n")...)
						   processedData = append(processedData, yamlBytes...)
					   } else {
						   processedData = yamlBytes
					   }
				   }
	}

       var cloudProviderAPI string = "aws"
       if awsAccountID != "" {
	       cloudProviderAPI = "aws"
       } else if gcpProjectID != "" {
	       cloudProviderAPI = "gcp"
       } else if azureSubscriptionID != "" {
	       cloudProviderAPI = "azure"
       }
	// Pre-check 1: Check for linked cloud provider accounts
	fmt.Print("Checking linked cloud provider accounts... ", cloudProviderAPI, gcpProjectID, azureSubscriptionID)
	accounts, err := dataaccess.ListAccounts(cmd.Context(), token, cloudProviderAPI)
	if err != nil {
		fmt.Println("❌")
		return fmt.Errorf("failed to check cloud provider accounts: %w", err)
	}

	if len(accounts.AccountConfigs) == 0 {
		fmt.Println("❌")
		return errors.New("no cloud provider accounts linked. Please link at least one cloud provider account using 'omctl account create' before deploying")
	}

	if len(accounts.AccountConfigs) == 1 {
		// Determine cloud provider from the account
		var cloudProvider string
		account := accounts.AccountConfigs[0]
		if account.AwsAccountID != nil {
			cloudProvider = "AWS"
		} else if account.GcpProjectID != nil {
			cloudProvider = "GCP"
		} else if account.AzureSubscriptionID != nil {
			cloudProvider = "Azure"
		} else {
			cloudProvider = "Unknown"
		}
		fmt.Printf("✅ (1 account found - assuming provider hosted: %s)\n", cloudProvider)
	} else {
		fmt.Printf("✅ (%d accounts found)\n", len(accounts.AccountConfigs))
	}

	// Validate cloud provider account flags for BYOA deployment
	if deploymentType == "byoa" || deploymentType == "hosted" && (awsAccountID != "" || gcpProjectID != "") {
		// Check if the specified account IDs match any linked accounts
		var foundMatchingAccount bool
		for _, account := range accounts.AccountConfigs {
			if awsAccountID != "" && account.AwsAccountID != nil && *account.AwsAccountID == awsAccountID {
				foundMatchingAccount = true
				break
			}
			if gcpProjectID != "" && account.GcpProjectID != nil && *account.GcpProjectID == gcpProjectID {
				if gcpProjectNumber != "" && account.GcpProjectNumber != nil && *account.GcpProjectNumber == gcpProjectNumber {
					foundMatchingAccount = true
					break
				}
			}
			if azureSubscriptionID != "" && account.AzureSubscriptionID != nil && *account.AzureSubscriptionID == azureSubscriptionID {
				if azureTenantID != "" && account.AzureTenantID != nil && *account.AzureTenantID == azureTenantID {
					foundMatchingAccount = true
					break
				}
			}
		}
		if !foundMatchingAccount {
			fmt.Println("❌")
			if awsAccountID != "" {
				return fmt.Errorf("AWS account ID %s is not linked to your organization. Please link it using 'omctl account create' first", awsAccountID)
			}
			if gcpProjectID != "" {
				return fmt.Errorf("GCP project %s/%s is not linked to your organization. Please link it using 'omctl account create' first", gcpProjectID, gcpProjectNumber)
			}
			if azureSubscriptionID != "" {
				return fmt.Errorf("Azure subscription %s/%s is not linked to your organization. Please link it using 'omctl account create' first", azureSubscriptionID, azureTenantID)
			}
		}
	}

	// Pre-check 2: Validate spec file configuration if using spec file
	if !useRepo {
		fmt.Print("Validating spec file configuration... ", processedData)
		tenantAwareResourceCount, err := validateSpecFileConfiguration(processedData, specType)
		if err != nil {
			fmt.Println("❌")
			return fmt.Errorf("spec file validation failed: %w", err)
		}

		if(deploymentType == "byoa" || deploymentType == "hosted"){
		// Check if account ID is required but missing
		// Account ID is considered available if it's in the spec file OR provided via flags
		hasAccountIDFromFlags := awsAccountID != "" || gcpProjectID != ""
		// Assume hosted deployment if we have a single cloud provider account
		assumeHosted := len(accounts.AccountConfigs) == 1
		if !assumeHosted && !hasAccountIDFromFlags {
			fmt.Println("❌")
			return errors.New("multiple cloud provider accounts found but no account ID specified in spec file or flags. Please specify account ID in spec file or use --aws-account-id/--gcp-project-id flags")
		}
	}

		// Check tenant-aware resource count
		if tenantAwareResourceCount == 0 {
			fmt.Println("❌")
			return errors.New("no tenant-aware resources found in spec file. At least one tenant-aware resource is required for deployment")
		} else if tenantAwareResourceCount > 1 {
			fmt.Println("❌")
			return errors.New("multiple tenant-aware resources found in spec file. Please specify exactly one tenant-aware resource or use --tenant-resource flag")
		}

		fmt.Println("✅")
	} else {
		// For repository-based builds, check if account ID is required for multiple accounts
		hasAccountIDFromFlags := awsAccountID != "" || gcpProjectID != ""
		assumeHosted := len(accounts.AccountConfigs) == 1


		if !assumeHosted && !hasAccountIDFromFlags && (deploymentType == "byoa" || deploymentType == "hosted") {
			fmt.Print("Validating repository configuration... ")
			fmt.Println("❌")
			return errors.New("multiple cloud provider accounts found but no account ID specified. Please use --aws-account-id/--gcp-project-id flags to specify which account to use")
		}
		
		fmt.Println("Validating repository configuration... ✅ (repository-based deployment)")
	}

	// Get service name for further validation
	productName, err := cmd.Flags().GetString("product-name")
	if err != nil {
		return err
	}

       var serviceNameToUse string
       serviceNameToUse = productName
       if serviceNameToUse == "" {
	       if !useRepo && len(processedData) > 0 {
		       // Try to extract 'name' from the YAML spec
		       nameRegex := regexp.MustCompile(`(?m)^name:\s*"?([a-zA-Z0-9_-]+)"?`)
		       matches := nameRegex.FindSubmatch(processedData)
		       if len(matches) > 1 {
			       serviceNameToUse = string(matches[1])
		       }
	       }
	       if serviceNameToUse == "" {
		       if useRepo {
			       // Use current directory name for repository-based builds
			       cwd, err := os.Getwd()
			       if err != nil {
				       return err
			       }
			       serviceNameToUse = filepath.Base(cwd)
		       } else {
			       // Use directory name from spec file path
			       serviceNameToUse = filepath.Base(filepath.Dir(absSpecFile))
		       }
		       if serviceNameToUse == "." || serviceNameToUse == "/" || serviceNameToUse == "" {
			       serviceNameToUse = "my-service"
		       }
	       }
       }

	// Pre-check 3: Check if service exists and validate service plan count
	fmt.Println("Checking existing service... ", serviceNameToUse)
	existingServiceID, envs, err := findExistingService(cmd.Context(), token, serviceNameToUse)
	if err != nil {
		fmt.Println("❌")
		return fmt.Errorf("failed to check existing service: %w", err)
	}

	if existingServiceID != "" {
		// Service exists - check service plan count
		servicePlans, err := getServicePlans(cmd.Context(), token, envs)
		if err != nil {
			fmt.Println("❌")
			return fmt.Errorf("failed to check service plans: %w", err)
		}
   		// Only validate --service-plan-id if provided, do not require it
		servicePlanID, _ := cmd.Flags().GetString("service-plan-id")
		
		if servicePlanID != "" {
			var found bool
			for _, plan := range servicePlans {
				// Extract ProductTierID using reflection since we don't know the exact type
				if planValue := fmt.Sprintf("%+v", plan); strings.Contains(planValue, servicePlanID) {
					found = true
					break
				}
			}
			if !found {
				fmt.Println("❌")
				return fmt.Errorf("service plan ID '%s' not found for service '%s'", servicePlanID, serviceNameToUse)
			}
		}else {
			if len(servicePlans) > 1 {
				fmt.Println("❌")
				return fmt.Errorf("service '%s' has %d service plans. Please specify --service-plan-id when multiple plans exist", serviceNameToUse, len(servicePlans))
			}
		}
		

		// Pre-check 4: Check deployment count if service exists
		fmt.Print("Checking existing deployments... ")
		deploymentCount, err := getDeploymentCount(cmd.Context(), token, existingServiceID, envs, instanceID)
		if err != nil {
			fmt.Println("❌")
			return fmt.Errorf("failed to check deployments: %w", err)
		}

		if deploymentCount > 1 {
			fmt.Println("❌")
			return fmt.Errorf("service '%s' has %d deployments. Please specify --instance-id when multiple deployments exist", serviceNameToUse, deploymentCount)
		}

		fmt.Printf("✅ (%d deployment)\n", deploymentCount)
	} else {
		fmt.Println("✅ (new service)")
	}

	fmt.Println("✅ All pre-checks passed! Proceeding with deployment...")


       // Get dry-run and wait flags
       dryRun, err := cmd.Flags().GetBool("dry-run")
       if err != nil {
	       return err
       }
       waitFlag, err = cmd.Flags().GetBool("wait")
       if err != nil {
	       return err
       }

	// Additional pre-checks: x-omnistrate-mode-internal tag and required parameters
	if !useRepo {
		// // Pre-check 5: Check for x-omnistrate-mode-internal tag
		// fmt.Print("Checking for x-omnistrate-mode-internal tag... ")
		// hasInternalTag, err := checkForInternalTag(processedData, specType)
		// if err != nil {
		// 	fmt.Println("❌")
		// 	return fmt.Errorf("failed to check x-omnistrate-mode-internal tag: %w", err)
		// }

		// if !hasInternalTag {
		// 	fmt.Println("❌")
		// 	return errors.New("at least one resource must have the x-omnistrate-mode-internal tag defined")
		// }
		// fmt.Println("✅")

		// Pre-check 6: Validate required parameters and prompt for missing values
		fmt.Print("Validating required parameters... ")
		// parameterErrors, err := validateAndPromptForRequiredParameters(processedData, specType)
		// if err != nil {
		// 	fmt.Println("❌")
		// 	return fmt.Errorf("failed to validate parameters: %w", err)
		// }

		// if len(parameterErrors) > 0 {
		// 	fmt.Println("❌")
		// 	fmt.Println("Missing required parameters:")
		// 	for _, paramErr := range parameterErrors {
		// 		fmt.Printf("  - %s\n", paramErr)
		// 	}
		// 	return errors.New("required parameters are missing. Please provide values for all required parameters")
		// }
		// fmt.Println("✅")
	}

	// Dry-run exit point
	if dryRun {
		fmt.Println("🔍 Dry-run completed successfully! All validations passed.")
		fmt.Println("To proceed with actual deployment, run the command without --dry-run flag.")
		return nil
	}
	fmt.Println()

	// Get flags
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

	// Step 1: Read and parse the spec file or prepare for repo build (if not already done)
	spinner := sm.AddSpinner("Reading and parsing spec file")
	
	if useRepo {
		// Repository-based build - no spec file to read
		spinner.UpdateMessage("Reading and parsing spec file: Using repository-based build")
		specType = "repository" // We'll handle this as a special case
	} else {
		// We already processed the data during pre-checks
		spinner.UpdateMessage(fmt.Sprintf("Reading and parsing spec file: %s (%s)", filepath.Base(absSpecFile), specType))
	}
	spinner.Complete()

	// Step 2: Determine service name
	spinner = sm.AddSpinner("Determining service name")
	if serviceNameToUse == "" {
		if useRepo {
			// Use current directory name for repository-based builds
			cwd, err := os.Getwd()
			if err != nil {
				utils.HandleSpinnerError(spinner, sm, err)
				return err
			}
			serviceNameToUse = filepath.Base(cwd)
		} else {
			// Use directory name from spec file path
			serviceNameToUse = filepath.Base(filepath.Dir(absSpecFile))
		}
		
		if serviceNameToUse == "." || serviceNameToUse == "/" || serviceNameToUse == "" {
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


	var serviceID, devEnvironmentID, devPlanID string
	var undefinedResources map[string]string


	if useRepo {
		serviceID, devEnvironmentID, devPlanID, undefinedResources, err = buildServiceFromRepo(
			cmd.Context(),
			token,
			serviceNameToUse,
			releaseDescriptionPtr,
			deploymentType,
			awsAccountID,
			gcpProjectID,
			gcpProjectNumber,
			azureSubscriptionID,
			azureTenantID,
			envVars,
			skipDockerBuild,
			skipServiceBuild,
			skipEnvironmentPromotion,
			skipSaasPortalInit,
			dryRun,
			platforms,
			resetPAT,
		)
	
	} else {
		
		serviceID, devEnvironmentID, devPlanID, undefinedResources, err = buildServiceSpec(
			cmd.Context(),
			processedData,
			token,
			serviceNameToUse,
			specType,
			releaseDescriptionPtr,
		)
		
	}
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

	// Execute post-service-build deployment workflow
	err = executeDeploymentWorkflow(cmd, sm, token, serviceID, devEnvironmentID, devPlanID, serviceNameToUse, instanceID)
	if err != nil {
		return err
	}

	return nil
}

// executeDeploymentWorkflow handles the complete post-service-build deployment workflow
// This function is reusable for both deploy and build_simple commands
func executeDeploymentWorkflow(cmd *cobra.Command, sm ysmrr.SpinnerManager, token, serviceID, devEnvironmentID, devPlanID, serviceName, instanceID string) error {
	const defaultProdEnvName = "Production"

	// Step 4: Check if production environment exists
	spinner := sm.AddSpinner("Checking if the production environment is set up")
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
	
	// Check if user provided a specific service plan ID
	userProvidedPlanID, _ := cmd.Flags().GetString("service-plan-id")
	
	for _, env := range service.ServiceEnvironments {
		if env.Id == prodEnvironmentID {
			if len(env.ServicePlans) > 0 {
				hasProductionPlans = true
				
				if userProvidedPlanID != "" {
					// Use the user-provided service plan ID
					for _, plan := range env.ServicePlans {
						if plan.ProductTierID == userProvidedPlanID {
							prodPlanID = plan.ProductTierID
							break
						}
					}
					if prodPlanID == "" {
						spinner.UpdateMessage("Setting service plan as preferred in production: Skipped (provided plan ID not found in production)")
						spinner.Complete()
						fmt.Printf("Warning: Provided service plan ID '%s' not found in production environment.\n\n", userProvidedPlanID)
						break
					}
				} else {
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
		subscriptionName = fmt.Sprintf("%s-subscription", serviceName)
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
				spinner.UpdateMessage("Creating subscription to the production service: Skipped (service provider org - will create instance directly)")
				spinner.Complete()
			} else if subscriptionResp == nil || subscriptionResp.Id == nil {
				// If error is due to missing subscription ID, skip subscription flow entirely
				spinner.UpdateMessage("Creating subscription to the production service: Skipped (no subscription ID required - will create instance directly)")
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
	var finalInstanceID string
	var instanceActionType string = "create"

	var existingInstanceID string

	var useSubscription = subscriptionID != ""
	spinnerMsg := "Checking for existing instances"
	if !useSubscription {
			spinnerMsg = "Checking for existing instances without subscription"
		}
	spinner = sm.AddSpinner(spinnerMsg)

	existingInstanceID, err = listInstances(cmd.Context(), token, serviceID, prodEnvironmentID, prodPlanID, instanceID, subscriptionID)
	if err != nil {
		spinner.UpdateMessage(spinnerMsg + ": Failed")
		spinner.Complete()
		fmt.Printf("Warning: Failed to check for existing instances: %s\n", err.Error())
		existingInstanceID = "" // Reset to create new instance
	}

	fmt.Printf("Note: Instance creation/upgrade is automatic.\n", existingInstanceID)

	if existingInstanceID != "" {
		foundMsg := spinnerMsg + ": Found existing instance"
		spinner.UpdateMessage(foundMsg)
		spinner.Complete()

		spinner = sm.AddSpinner(fmt.Sprintf("Upgrading existing instance: %s", existingInstanceID))
		upgradeErr := upgradeExistingInstance(cmd.Context(), token, existingInstanceID, serviceID, prodPlanID)
		instanceActionType = "upgrade"
		if upgradeErr != nil {
			spinner.UpdateMessage(fmt.Sprintf("Upgrading existing instance: Failed (%s)", upgradeErr.Error()))
			spinner.Complete()
			fmt.Printf("Warning: Instance upgrade failed: %s\n", upgradeErr.Error())
		} else {
			finalInstanceID = existingInstanceID
			spinner.UpdateMessage(fmt.Sprintf("Upgrading existing instance: Success (ID: %s)", finalInstanceID))
			spinner.Complete()
		}
		
	} else {
		noFoundMsg := spinnerMsg + ": No existing instances found"
		spinner.UpdateMessage(noFoundMsg)
		spinner.Complete()

		createMsg := "Creating new instance deployment"
		if !useSubscription {
			createMsg = "Creating instance without subscription"
		}
		spinner = sm.AddSpinner(createMsg)
		createdInstanceID, err := "", error(nil)
		createdInstanceID, err = createInstanceUnified(cmd.Context(), token, serviceID, prodEnvironmentID, prodPlanID, utils.ToPtr(subscriptionID))
		finalInstanceID = createdInstanceID  
		instanceActionType = "create"
		if err != nil {
			spinner.UpdateMessage(fmt.Sprintf("%s: Failed (%s)", createMsg, err.Error()))
			spinner.Complete()
			if useSubscription {
			fmt.Printf("Error: Failed to create instance: %s\n", err.Error())
			}else{
				fmt.Printf("Error: Failed to create instance without subscription: %s\n", err.Error())
			}

		} else {
			spinner.UpdateMessage(fmt.Sprintf("%s: Success (ID: %s)", createMsg, finalInstanceID))
			spinner.Complete()
		}
	}



       // Step 10: Success message - completed deployment
       spinner = sm.AddSpinner("Deployment workflow completed")
       spinner.UpdateMessage("Deployment workflow completed: Service built, promoted, and ready for instances")
       spinner.Complete()

       sm.Stop()

	   // Success message
	   fmt.Println()
	   fmt.Println("🎉 Deployment completed successfully!")
	   fmt.Printf("   Service: %s (ID: %s)\n", serviceName, serviceID)
	   fmt.Printf("   Production Environment: %s (ID: %s)\n", defaultProdEnvName, prodEnvironmentID)
	   if subscriptionID != "" {
		   fmt.Printf("   Subscription: %s (ID: %s)\n", subscriptionName, subscriptionID)
	   } else if isServiceProvider {
		   fmt.Printf("   Organization: Service Provider (direct instance deployment)\n")
	   }
	   if finalInstanceID != "" {
		   fmt.Printf("   Instance: %s (ID: %s)\n", instanceActionType, finalInstanceID)
	   }
	   fmt.Println()
	   
	   // Optionally display workflow progress if desired (if you want to keep this logic, pass cmd/context as needed)
	   if waitFlag && finalInstanceID != "" {
		   fmt.Println("🔄 Deployment progress...")
		   err = instancecmd.DisplayWorkflowResourceDataWithSpinners(cmd.Context(), token, finalInstanceID, instanceActionType)
		   if err != nil {
			   fmt.Printf("❌ Deployment failed-- %s\n", err)
		   } else {
			   fmt.Println("✅ Deployment successful")
		   }
	   }
	   return nil
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



// createInstanceUnified creates an instance with or without subscription, removing duplicate code
// waitFlag is set from the deploy command flag
func createInstanceUnified(ctx context.Context, token, serviceID, environmentID, productTierID string, subscriptionID *string) (string, error) {
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
	defaultParams := map[string]interface{}{}

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
	}
	if subscriptionID != nil && *subscriptionID != "" {
		request.SubscriptionId = subscriptionID
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





// listInstances is a helper function for backward compatibility
func listInstances(ctx context.Context, token, serviceID, environmentID, servicePlanID, instanceID, subscriptionID string) (string, error) {
	res, err := dataaccess.ListResourceInstance(ctx, token, serviceID, environmentID)
	if err != nil {
		return "", fmt.Errorf("failed to search for instances: %w", err)
	}

	fmt.Printf("Total instances found: %d\n", len(res.ResourceInstances))
	exitInstanceID := ""
	err = nil
	for _, instance := range res.ResourceInstances {
		var idStr string
		if instance.ConsumptionResourceInstanceResult.Id != nil {
			idStr = *instance.ConsumptionResourceInstanceResult.Id
		} else {
			idStr = "<nil>"
		}
		
		instanceCount := 0


		// Match based on provided filters
		// Priority: instanceID > subscriptionID > servicePlanID
		if exitInstanceID == "" && instanceID != "" && idStr != "" && idStr == instanceID {
			exitInstanceID = idStr
			instanceCount++
		} else if exitInstanceID == "" && subscriptionID != "" && instance.SubscriptionId != "" && instance.SubscriptionId == subscriptionID {
			if idStr!= "" {
				exitInstanceID = idStr
			}
			instanceCount++
			fmt.Println("instance.SubscriptionId :",instance.SubscriptionId, subscriptionID,idStr)
		} else if exitInstanceID == "" && servicePlanID != "" && instance.ProductTierId == servicePlanID {
			if idStr!= "" {
				exitInstanceID = idStr
			}
			fmt.Println("instance.SubscriptionId :",instance.ProductTierId,servicePlanID,idStr)
			instanceCount++
			
	}
	// If multiple instances match servicePlanID, return error to specify instanceID
		if instanceCount > 1 {
			err = fmt.Errorf("multiple instances found for service plan ID %s - please specify instance ID to select the correct one", servicePlanID)
		}
	}
	return exitInstanceID, err
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

// validateSpecFileConfiguration validates the spec file for  tenant-aware resources
func validateSpecFileConfiguration(data []byte, specType string) (tenantAwareResourceCount int, err error) {
	content := string(data)
	// For Docker Compose specs, look for tenant-aware resources
	if specType == build.DockerComposeSpecType {
		// Only count x-omnistrate-mode-internal if not explicitly set to false
		// Match lines like: x-omnistrate-mode-internal: true OR just x-omnistrate-mode-internal (without : false)
		tenantAwareRegex := regexp.MustCompile(`(?m)^\s*x-omnistrate-mode-internal\s*:\s*(false|False|FALSE)\s*$`)
		matches := tenantAwareRegex.FindAllString(content, -1)
		tenantAwareResourceCount = len(matches)

	} else {
		// For Service Plan specs, assume tenant-aware if not specified otherwise
		tenantAwareResourceCount = 1
	}
	
	return  tenantAwareResourceCount, nil
}

// findExistingService searches for an existing service by name
func findExistingService(ctx context.Context, token, serviceName string) (string, map[string]interface{}, error) {
	services, err := dataaccess.ListServices(ctx, token)
	if err != nil {
		return "", nil, fmt.Errorf("failed to list services: %w", err)
	}

	for _, service := range services.Services {
		if service.Name == serviceName {
			// Build map with keys like "DEV", "PROD", etc. mapping to the full ServiceEnvironment object
			envs := make(map[string]interface{})
			for _, env := range service.ServiceEnvironments {
				if env.Type != nil && *env.Type != "" {
					envs[*env.Type] = env
				}
			}
			return service.Id, envs, nil
		}
	}

	return "", nil, nil // Service not found
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

// getDeploymentCount counts the number of deployments for a service
func getDeploymentCount(ctx context.Context, token, serviceID string, envs map[string]interface{}, instanceID string) (int, error) {
	var envObj interface{}
	var ok bool

	envObj, ok = envs["PROD"]
	if !ok {
		envObj, ok = envs["prod"]
	}
	if !ok {
		return 0, fmt.Errorf("no PROD environment found for service")
	}

	// Extract environment ID
	var environmentID string
	switch env := envObj.(type) {
	case map[string]interface{}:
		if id, exists := env["Id"]; exists {
			if idStr, ok := id.(string); ok {
				environmentID = idStr
			} else {
				return 0, fmt.Errorf("environment ID is not a string")
			}
		} else {
			return 0, fmt.Errorf("environment does not contain 'Id' field")
		}
	default:
		// Try reflection for struct types
		val := reflect.ValueOf(envObj)
		if val.Kind() == reflect.Ptr {
			val = val.Elem()
		}
		if val.Kind() == reflect.Struct {
			field := val.FieldByName("Id")
			if field.IsValid() && field.CanInterface() {
				if idStr, ok := field.Interface().(string); ok {
					environmentID = idStr
				} else {
					return 0, fmt.Errorf("environment ID is not a string")
				}
			} else {
				return 0, fmt.Errorf("environment struct does not contain 'Id' field")
			}
		} else {
			return 0, fmt.Errorf("unsupported environment type for extracting ID")
		}
	}

	// Search for instances associated with this service
	res, err := dataaccess.ListResourceInstance(ctx, token, serviceID, environmentID)
	if err != nil {
		return 0, fmt.Errorf("failed to search for deployments: %w", err)
	}

	if instanceID != ""{
		for _, instance := range res.ResourceInstances {
			if instance.ConsumptionResourceInstanceResult.Id != nil && *instance.ConsumptionResourceInstanceResult.Id == instanceID {
				return 1, nil
			}
		}
		return 0, fmt.Errorf("Instance Id not match for prod environment: %w", err)
	}
	
	return len(res.ResourceInstances), nil
}



// checkForInternalTag checks if at least one resource has the x-omnistrate-mode-internal tag
func checkForInternalTag(data []byte, specType string) (bool, error) {
	content := string(data)
	
	// Check for x-omnistrate-mode-internal tag in various formats
	internalTagPatterns := []string{
		"x-omnistrate-mode-internal",
		"x-omnistrate-mode: internal",
		"x-omnistrate-mode: \"internal\"",
		"x-omnistrate-mode: 'internal'",
	}
	
	for _, pattern := range internalTagPatterns {
		if strings.Contains(content, pattern) {
			return true, nil
		}
	}
	
	return false, nil
}

// validateAndPromptForRequiredParameters validates required parameters and prompts for missing values
func validateAndPromptForRequiredParameters(data []byte, specType string) ([]string, error) {
	var parameterErrors []string
	
	if specType == build.DockerComposeSpecType {
		// Parse the compose spec to check for environment variables without defaults
		var composeProject *types.Project
		var err error
		
		// Load the compose project from data
		composeProject, err = loader.LoadWithContext(context.Background(), types.ConfigDetails{
			WorkingDir: ".",
			ConfigFiles: []types.ConfigFile{
				{
					Content: data,
				},
			},
		})
		if err != nil {
			return nil, fmt.Errorf("failed to parse compose spec: %w", err)
		}
		
		// Check all services for environment variables without defaults
		for serviceName, service := range composeProject.Services {
			// Check environment variables
			for _, envVar := range service.Environment {
				envVarStr := ""
				if envVar != nil {
					envVarStr = *envVar
				}
				
				if strings.Contains(envVarStr, "${") && !strings.Contains(envVarStr, ":-") {
					// This is a required environment variable without a default
					varName := extractEnvVarName(envVarStr)
					if varName != "" {
						// Check if it's set in the environment
						if os.Getenv(varName) == "" {
							// Prompt user for the value
							fmt.Printf("Service '%s' requires environment variable '%s'. Please enter a value: ", serviceName, varName)
							var value string
							fmt.Scanln(&value)
							if value == "" {
								parameterErrors = append(parameterErrors, fmt.Sprintf("service '%s' requires environment variable '%s'", serviceName, varName))
							} else {
								// Set the environment variable for this session
								os.Setenv(varName, value)
							}
						}
					}
				}
			}
			
			// Check command and args for template variables
			if service.Command != nil {
				for _, cmd := range service.Command {
					if strings.Contains(cmd, "${") && !strings.Contains(cmd, ":-") {
						varName := extractEnvVarName(cmd)
						if varName != "" && os.Getenv(varName) == "" {
							parameterErrors = append(parameterErrors, fmt.Sprintf("service '%s' command requires variable '%s'", serviceName, varName))
						}
					}
				}
			}
		}
	} else {
		// For service plan specs, check for template variables in the YAML
		content := string(data)
		
		// Look for template variables like ${VAR} without defaults ${VAR:-default}
		templateVarRegex := regexp.MustCompile(`\$\{([^}]+)\}`)
		matches := templateVarRegex.FindAllStringSubmatch(content, -1)
		
		for _, match := range matches {
			if len(match) > 1 {
				varExpression := match[1]
				// Skip if it has a default value (contains :-)
				if strings.Contains(varExpression, ":-") {
					continue
				}
				
				varName := strings.TrimSpace(varExpression)
				if os.Getenv(varName) == "" {
					// Prompt user for the value
					fmt.Printf("Spec file requires variable '%s'. Please enter a value: ", varName)
					var value string
					fmt.Scanln(&value)
					if value == "" {
						parameterErrors = append(parameterErrors, fmt.Sprintf("spec file requires variable '%s'", varName))
					} else {
						// Set the environment variable for this session
						os.Setenv(varName, value)
					}
				}
			}
		}
	}
	
	return parameterErrors, nil
}

// extractEnvVarName extracts the environment variable name from a template expression
func extractEnvVarName(envVar string) string {
	// Handle ${VAR} format
	if strings.Contains(envVar, "${") && strings.Contains(envVar, "}") {
		start := strings.Index(envVar, "${") + 2
		end := strings.Index(envVar[start:], "}")
		if end > 0 {
			varExpression := envVar[start : start+end]
			// Remove default value if present (${VAR:-default})
			if colonIndex := strings.Index(varExpression, ":"); colonIndex > 0 {
				return varExpression[:colonIndex]
			}
			return varExpression
		}
	}
	
	return ""
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
       yamlDoc := make(map[string]interface{})

	if awsBootstrapRoleARN == "" && awsAccountID != "" {
		// Default role ARN if not provided
		awsBootstrapRoleARN = fmt.Sprintf("arn:aws:iam::%s:role/OmnistrateBootstrapRole", awsAccountID)
	}

	if gcpServiceAccountEmail == "" && gcpProjectID != "" {
		// Default service account email if not provided
		gcpServiceAccountEmail = fmt.Sprintf("omnistrate-bootstrap@%s.iam.gserviceaccount.com", gcpProjectID)
	}

	// Clear out any existing deployment or account sections to avoid duplication
	delete(yamlDoc, "deployment")
	delete(yamlDoc, "x-omnistrate-byoa")
	delete(yamlDoc, "x-omnistrate-my-account")

	       // Build the deployment section based on deploymentType and specType

	       if deploymentType == "byoa" {
		       if specType != "DockerCompose"  {
			       yamlDoc["deployment"] = map[string]interface{}{
				       "byoaDeployment": map[string]interface{}{
					       "AwsAccountId": awsAccountID,
					       "AwsBootstrapRoleAccountArn": awsBootstrapRoleARN,
				       },
			       }
		       } else {
			       yamlDoc["x-omnistrate-byoa"] = map[string]interface{}{
				       "AwsAccountId": awsAccountID,
				       "AwsBootstrapRoleAccountArn": awsBootstrapRoleARN,
			       }
		       }
	       } else if deploymentType == "hosted" {
		       if specType != "DockerCompose" {
			       hostedDeployment := make(map[string]interface{})
			       if awsAccountID != "" {
				       hostedDeployment["AwsAccountId"] = awsAccountID
				       if awsBootstrapRoleARN != "" {
					       hostedDeployment["AwsBootstrapRoleAccountArn"] = awsBootstrapRoleARN
				       }
			       }
			       if gcpProjectID != "" {
				       hostedDeployment["GcpProjectId"] = gcpProjectID
				       if gcpProjectNumber != "" {
					       hostedDeployment["GcpProjectNumber"] = gcpProjectNumber
				       }
				       if gcpServiceAccountEmail != "" {
					       hostedDeployment["GcpServiceAccountEmail"] = gcpServiceAccountEmail
				       }
			       }
			       if azureSubscriptionID != "" {
				       hostedDeployment["AzureSubscriptionID"] = azureSubscriptionID
				       if azureTenantID != "" {
					       hostedDeployment["AzureTenantID"] = azureTenantID
				       }
			       }
			       yamlDoc["deployment"] = map[string]interface{}{
				       "hostedDeployment": hostedDeployment,
			       }
		       } else {
			       myAccount := make(map[string]interface{})
			       if awsAccountID != "" {
				       myAccount["AwsAccountId"] = awsAccountID
				       if awsBootstrapRoleARN != "" {
					       myAccount["AwsBootstrapRoleAccountArn"] = awsBootstrapRoleARN
				       }
			       }
			       if gcpProjectID != "" {
				       myAccount["GcpProjectId"] = gcpProjectID
				       if gcpProjectNumber != "" {
					       myAccount["GcpProjectNumber"] = gcpProjectNumber
				       }
				       if gcpServiceAccountEmail != "" {
					       myAccount["GcpServiceAccountEmail"] = gcpServiceAccountEmail
				       }
			       }
			       if azureSubscriptionID != "" {
				       myAccount["AzureSubscriptionID"] = azureSubscriptionID
				       if azureTenantID != "" {
					       myAccount["AzureTenantID"] = azureTenantID
				       }
			       }
			       yamlDoc["x-omnistrate-my-account"] = myAccount
		       }
			}
	   return yamlDoc
}

