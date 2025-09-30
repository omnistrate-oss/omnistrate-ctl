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
	DeployCmd.Flags().StringP("file", "f", build.ComposeFileName, "Path to the docker compose file")
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
	DeployCmd.Flags().String("deployment-type", "hosted", "Deployment type: hosted  or byoa")
	DeployCmd.Flags().String("service-plan-id", "", "Specify the service plan ID to use when multiple plans exist")
	DeployCmd.Flags().StringP("spec-type", "s", build.DockerComposeSpecType, "Spec type")
	DeployCmd.Flags().Bool("wait", false, "Wait for deployment to complete before returning.")
	DeployCmd.Flags().String("instance-id", "", "Specify the instance ID to use when multiple deployments exist.")
	DeployCmd.Flags().String("cloud-provider", "", "Cloud provider (aws|gcp|azure)")
	DeployCmd.Flags().String("region", "", "Region code (e.g. us-east-2, us-central1)")
	DeployCmd.Flags().String("param", "", "Parameters for the instance deployment")
	DeployCmd.Flags().String("param-file", "", "Json file containing parameters for the instance deployment")
	// Additional flags from build command
	DeployCmd.Flags().StringArray("env-var", nil, "Specify environment variables required for running the image. Use the format: --env-var key1=var1 --env-var key2=var2. Only effective when no compose spec exists in the repo.")
	DeployCmd.Flags().Bool("skip-docker-build", false, "Skip building and pushing the Docker image")
	DeployCmd.Flags().Bool("skip-service-build", false, "Skip building the service from the compose spec")
	DeployCmd.Flags().Bool("skip-environment-promotion", false, "Skip creating and promoting to the production environment")
	DeployCmd.Flags().Bool("skip-saas-portal-init", false, "Skip initializing the SaaS Portal")
	DeployCmd.Flags().StringArray("platforms", []string{"linux/amd64"}, "Specify the platforms to build for. Use the format: --platforms linux/amd64 --platforms linux/arm64. Default is linux/amd64.")
	DeployCmd.Flags().Bool("reset-pat", false, "Reset the GitHub Personal Access Token (PAT) for the current user.")
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


	// Step 0: Validate user is logged in first
	token, err := common.GetTokenWithLogin()
	if err != nil {
		fmt.Println("❌ Not logged in. Please run 'omctl login' to authenticate.")
		return err
	}

	// Retrieve flags
	file, err := cmd.Flags().GetString("file")
	if err != nil {
		return err
	}
	// Check if file was explicitly provided
	fileExplicit := cmd.Flags().Changed("file")

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

	
	// Get deployment type flag
	deploymentType, err := cmd.Flags().GetString("deployment-type")
	if err != nil {
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


	// Get dry-run and wait flags
	dryRun, err := cmd.Flags().GetBool("dry-run")
	if err != nil {
		return err
	}
	waitFlag, err = cmd.Flags().GetBool("wait")
	if err != nil {
		return err
	}

	specType, err := cmd.Flags().GetString("spec-type")
	if err != nil {
		return err
	}

	// Validate spec-type - only support DockerCompose or ServicePlanSpec
	if specType != "" && specType != build.DockerComposeSpecType && specType != build.ServicePlanSpecType {
		return fmt.Errorf("❌ invalid spec-type '%s'. Supported values: '%s' or '%s'", 
			specType, build.DockerComposeSpecType, build.ServicePlanSpecType)
	}

	// Check if spec-type was explicitly provided
	specTypeExplicit := cmd.Flags().Changed("spec-type")
	if !specTypeExplicit {
		if file == build.ComposeFileName {
			specType = build.DockerComposeSpecType
		} else {
			specType = build.ServicePlanSpecType
		}
	}

	// Get instance-id flag value
	instanceID, err := cmd.Flags().GetString("instance-id")
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



	// Pre-checks: Validate environment and requirements
	fmt.Println("Running pre-deployment checks...")
	
	// Get the spec file path - follow the same flow as build_from_repo.go
	var specFile string
	var useRepo bool
	
	if len(args) > 0 {
		specFile = args[0]
	} else if file != "" {
		if _, err := os.Stat(file); os.IsNotExist(err) {
			if fileExplicit {
				utils.PrintError(err)
				return err
			} else {
				// If the file doesn't exist and wasn't explicitly provided, we check if there is a spec file
				file = "plan.yaml"
				if _, err := os.Stat(file); os.IsNotExist(err) {
					utils.PrintError(err)
					return err
				}
			}
		}

		specFile = file
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

      

	// Convert to absolute path if using spec file
	var absSpecFile string
	var processedData []byte
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
		       // If the file is named compose.yaml , default to build.DockerComposeSpecType
		       baseName := filepath.Base(absSpecFile)
		       if baseName == "compose.yaml" {
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
					   yamlMap = createDeploymentYAML(
						   deploymentType,
						   specType,
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
					   // If processedData already has content, remove deployment/account keys before appending
					   if len(processedData) > 0 {
						   var specMap map[string]interface{}
						   if err := yaml.Unmarshal(processedData, &specMap); err == nil {
							   delete(specMap, "deployment")
							   delete(specMap, "x-omnistrate-byoa")
							   delete(specMap, "x-omnistrate-my-account")
							   cleanedData, _ := yaml.Marshal(specMap)
							   processedData = cleanedData
						   }
						   processedData = append(processedData, yamlBytes...)
					   } else {
						   processedData = yamlBytes
					   }
					   
					   // Overwrite the original spec file with the new YAML
					   if len(yamlBytes) > 0 {
						   err = os.WriteFile(absSpecFile, processedData, 0644)
					   if err != nil {
						   return errors.Wrap(err, "failed to overwrite spec file with processed YAML")
					   }
					}
				   }
				   
				   // Extract cloud account information from the processed YAML if not provided via flags
				   // This should run regardless of which flags are provided, to extract any missing account info
				   extractedAWS, extractedGCP, extractedGCPNum, extractedAzure, extractedAzureTenant := extractCloudAccountsFromProcessedData(processedData)
				   
				   
				   // Use extracted values if not provided via command line flags
				   if awsAccountID == "" && extractedAWS != "" {
					   awsAccountID = extractedAWS
					   fmt.Printf("📋 Found AWS account ID in spec file: %s\n", awsAccountID)
				   }
				   if gcpProjectID == "" && extractedGCP != "" {
					   gcpProjectID = extractedGCP
					   fmt.Printf("📋 Found GCP project ID in spec file: %s\n", gcpProjectID)
				   }
				   if gcpProjectNumber == "" && extractedGCPNum != "" {
					   gcpProjectNumber = extractedGCPNum
					   fmt.Printf("📋 Found GCP project number in spec file: %s\n", gcpProjectNumber)
				   }
				   if azureSubscriptionID == "" && extractedAzure != "" {
					   azureSubscriptionID = extractedAzure
					   fmt.Printf("📋 Found Azure subscription ID in spec file: %s\n", azureSubscriptionID)
				   }
				   if azureTenantID == "" && extractedAzureTenant != "" {
					   azureTenantID = extractedAzureTenant
					   fmt.Printf("📋 Found Azure tenant ID in spec file: %s\n", azureTenantID)
				   }
	}

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
	 

	   allAccounts := []interface{}{}
	   // Filter for READY accounts and collect status information
	   readyAccounts := []interface{}{}
	   accountStatusSummary := make(map[string]int)
	   var foundMatchingAccount bool
		var accountStatus string

	   for _, cp := range allCloudProviders {
		   // Pre-check 1: Check for linked cloud provider accounts
		   accounts, err := dataaccess.ListAccounts(cmd.Context(), token, cp)
		   if err != nil {
			   return fmt.Errorf("❌ failed to check cloud provider accounts: %w", err)
		   }
		   for _, acc := range accounts.AccountConfigs {
			   allAccounts = append(allAccounts, acc)
			   if acc.Status == "READY" {
				   readyAccounts = append(readyAccounts, acc)
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
	fmt.Printf("Checking cloud provider accounts...\n")
	
	
	// Auto-configure cloud provider and account IDs from single account if available
	if awsAccountID == "" && gcpProjectID == "" && azureSubscriptionID == "" {

	// Display detailed account status information
	fmt.Printf("✅ Found %d total accounts:\n", len(allAccounts))
	for status, count := range accountStatusSummary {
		if status == "READY" {
			fmt.Printf("   - %d %s accounts ✅\n", count, status)
		} else {
			fmt.Printf("   - %d %s accounts ⚠️\n", count, status)
		}
	}

	// Ensure at least one READY account is available
	if len(readyAccounts) == 0 {
		if len(allAccounts) > 0 {
			fmt.Printf("\n❌ No READY accounts found. Account setup required:\n")
			fmt.Printf("   Your organization has %d accounts, but none are in READY status.\n", len(allAccounts))
			fmt.Printf("   Non-READY accounts may need to complete onboarding or have configuration issues.\n")
			fmt.Printf("\n💡 Next steps:\n")
			fmt.Printf("   1. Check existing account status: omctl account list\n")
			fmt.Printf("   2. Complete onboarding for existing accounts, or\n")
			fmt.Printf("   3. Create a new READY account: omctl account create\n")
			return errors.New("❌ deployment requires at least one READY cloud provider account")
		} else {
			fmt.Printf("\n❌ No cloud provider accounts found.\n")
			fmt.Printf("💡 Create your first account: omctl account create\n")
			return errors.New("❌ no cloud provider accounts linked. Please link at least one account using 'omctl account create' before deploying")
		}
	}

	if len(readyAccounts) == 1 {
		account := readyAccounts[0]
		if accMap, ok := account.(map[string]interface{}); ok {
			if accMap["awsAccountID"] != nil && awsAccountID == "" {
				if v, ok := accMap["awsAccountID"].(string); ok {
					awsAccountID = v
				}
				if cloudProvider == "" {
					cloudProvider = "aws"
				}
			} else if accMap["gcpProjectID"] != nil && gcpProjectID == "" && gcpProjectNumber == "" {
				if v, ok := accMap["gcpProjectID"].(string); ok {
					gcpProjectID = v
				}
				if v, ok := accMap["gcpProjectNumber"].(string); ok {
					gcpProjectNumber = v
				}
				if cloudProvider == "" {
					cloudProvider = "gcp"
				}
			} else if accMap["azureSubscriptionID"] != nil && azureSubscriptionID == "" && azureTenantID == "" {
				cloudProvider = "azure"
				if v, ok := accMap["azureSubscriptionID"].(string); ok {
					azureSubscriptionID = v
				}
				if v, ok := accMap["azureTenantID"].(string); ok {
					azureTenantID = v
				}
				if cloudProvider == "" {
					cloudProvider = "azure"
				}
			}
		}
		fmt.Printf("✅ (1 account found - assuming provider hosted: %s)\n", cloudProvider)
	} else if len(readyAccounts) > 1 && awsAccountID == "" && gcpProjectID == "" && azureSubscriptionID == "" {
		return fmt.Errorf("❌ multiple cloud provider accounts found but no account ID specified in spec file or flags. Please specify account ID in spec file or use --aws-account-id/--gcp-project-id/--azure-subscription-id flags")

	}  else {
		fmt.Printf("✅ (%d accounts found)\n", len(readyAccounts))
	}
}


	// Validate cloud provider account flags for BYOA deployment
	if (awsAccountID != "" || gcpProjectID != "" || azureSubscriptionID != "") {
		// Check if the specified account IDs match any linked accounts
	
		if !foundMatchingAccount {
			
			if awsAccountID != "" {
				return fmt.Errorf("❌ AWS account ID %s is not linked to your organization. Please link it using 'omctl account create' first", awsAccountID)
			}
			if gcpProjectID != "" {
				return fmt.Errorf("❌ GCP project %s/%s is not linked to your organization. Please link it using 'omctl account create' first", gcpProjectID, gcpProjectNumber)
			}
			if azureSubscriptionID != "" {
				return fmt.Errorf("❌ Azure subscription %s/%s is not linked to your organization. Please link it using 'omctl account create' first", azureSubscriptionID, azureTenantID)
			}
		} else if accountStatus != "READY" {
			if awsAccountID != "" {
				return fmt.Errorf("❌ AWS account ID %s is linked but has status '%s'. Please check the account status in your organization and complete onboarding if required.", awsAccountID, accountStatus)
			}
			if gcpProjectID != "" {
				return fmt.Errorf("❌ GCP project %s/%s is linked but has status '%s'. Please check the account status in your organization and complete onboarding if required.", gcpProjectID, gcpProjectNumber, accountStatus)
			}
			if azureSubscriptionID != "" {
				return fmt.Errorf("❌ Azure subscription %s/%s is linked but has status '%s'. Please check the account status in your organization and complete onboarding if required.", azureSubscriptionID, azureTenantID, accountStatus)
			}
		}
		fmt.Println("✅ (specified account is linked and READY)")
	}

	// Pre-check 2: Validate spec file configuration if using spec file
	if !useRepo {
		tenantAwareResourceCount, err := validateSpecFileConfiguration(processedData, specType)
		if err != nil {
			return fmt.Errorf("❌ spec file validation failed: %w", err)
		}

		fmt.Println("Validating spec file configuration... ")		// Check tenant-aware resource count
		if tenantAwareResourceCount == 0 {
			return errors.New("❌ no tenant-aware resources found in spec file. At least one tenant-aware resource is required for deployment")
		} else if tenantAwareResourceCount > 1 {
			return errors.New("❌ multiple tenant-aware resources found in spec file. Please specify exactly one tenant-aware resource or use --tenant-resource flag")
		}

		fmt.Printf("✅ ")
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
		       // Try to extract 'name' from the YAML spec using proper YAML parsing
		       var yamlContent map[string]interface{}
		       if err := yaml.Unmarshal(processedData, &yamlContent); err == nil {
			       if nameVal, exists := yamlContent["name"]; exists {
				       if nameStr, ok := nameVal.(string); ok && nameStr != "" {
					       serviceNameToUse = sanitizeServiceName(nameStr)
				       }
			       }
		       }
	       }
	       if serviceNameToUse == "" {
		       if useRepo {
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
			       serviceNameToUse = "my-service"
		       }
	       }
       }

	// Pre-check 3: Check if service exists and validate service plan count
	fmt.Println("Checking existing service... ", serviceNameToUse)
	existingServiceID, envs, err := findExistingService(cmd.Context(), token, serviceNameToUse)
	if err != nil {
		return fmt.Errorf("❌ failed to check existing service: %w", err)
	}

	if existingServiceID != "" {
		// Service exists - check service plan count
		servicePlans, err := getServicePlans(cmd.Context(), token, envs)
		if err != nil {
			return fmt.Errorf("❌ failed to check service plans: %w", err)
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
				return fmt.Errorf("❌ service plan ID '%s' not found for service '%s'", servicePlanID, serviceNameToUse)
			}
		} else {
			if len(servicePlans) > 1 {
				fmt.Printf("❌ Multiple service plans found for service '%s'\n", serviceNameToUse)
				fmt.Printf("Service '%s' has %d service plans.\n", serviceNameToUse, len(servicePlans))
				fmt.Println("Available service plans:")
				for idx, plan := range servicePlans {
					fmt.Printf("  %d. %v\n", idx+1, plan)
				}
				return fmt.Errorf("Please specify --service-plan-id when multiple plans exist.")
			}
		}
		

		// Pre-check 4: Check deployment count if service exists
		fmt.Print("Checking existing deployments... ")
		deploymentCount, err := getDeploymentCount(cmd.Context(), token, existingServiceID, envs, instanceID)
		if err != nil {
			return fmt.Errorf("❌ failed to check deployments: %w", err)
		}

		if deploymentCount > 1 {
			fmt.Printf("Service '%s' has %d deployments.\n", serviceNameToUse, deploymentCount)
			fmt.Println("You have multiple deployment instances. Please choose one of the following options:")
			fmt.Println("  1. Upgrade all instances")
			fmt.Println("  2. Specify --instance-id to upgrade a specific instance")
			fmt.Println("To upgrade all, rerun with --upgrade-all. To upgrade a specific instance, rerun with --instance-id <id>.")
			// Optionally, you can implement interactive selection here if desired
			return nil
		}

		fmt.Printf("✅ (%d deployment)\n", deploymentCount)
	} else {
		fmt.Println("✅ (new service)")
	}

	fmt.Println("✅ All pre-checks passed! Proceeding with deployment...")



	// Dry-run exit point
	if dryRun {
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

	// Get flags
	releaseDescription, err := cmd.Flags().GetString("release-description")
	if err != nil {
		return err
	}

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
			serviceNameToUse = sanitizeServiceName(filepath.Base(cwd))
		} else {
			// Use directory name from spec file path
			serviceNameToUse = sanitizeServiceName(filepath.Base(filepath.Dir(absSpecFile)))
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
	err = executeDeploymentWorkflow(cmd, sm, token, serviceID, devEnvironmentID, devPlanID, serviceNameToUse, instanceID, cloudProvider, region, param, paramFile)
	if err != nil {
		return err
	}

	return nil
}

// executeDeploymentWorkflow handles the complete post-service-build deployment workflow
// This function is reusable for both deploy and build_simple commands
func executeDeploymentWorkflow(cmd *cobra.Command, sm ysmrr.SpinnerManager, token, serviceID, devEnvironmentID, devPlanID, serviceName, instanceID, cloudProvider, region, param, paramFile string) error {
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

	


	// Step 9: Create or upgrade instance deployment automatically
	var finalInstanceID string
	var instanceActionType string = "create"


	spinnerMsg := "Checking for existing instances"
	spinner = sm.AddSpinner(spinnerMsg)

	var existingInstanceIDs []string
	existingInstanceIDs, err = listInstances(cmd.Context(), token, serviceID, prodEnvironmentID, prodPlanID, instanceID)
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
		upgradeErr := upgradeExistingInstance(cmd.Context(), token, existingInstanceIDs, serviceID, prodPlanID)
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
		createdInstanceID, err = createInstanceUnified(cmd.Context(), token, serviceID, prodEnvironmentID, prodPlanID, cloudProvider, region, param, paramFile)
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
       spinner.UpdateMessage("Deployment workflow completed: Service built, promoted, and ready for instances")
       spinner.Complete()

       sm.Stop()

	   // Success message
	   fmt.Println()
	   fmt.Printf("   Service: %s (ID: %s)\n", serviceName, serviceID)
	   fmt.Printf("   Production Environment: %s (ID: %s)\n", defaultProdEnvName, prodEnvironmentID)
	   if finalInstanceID != "" {
		   fmt.Printf("   Instance: %s (ID: %s)\n", instanceActionType, finalInstanceID)
	   }
	   fmt.Println()
	   
	   // Optionally display workflow progress if desired (if you want to keep this logic, pass cmd/context as needed)
	   if waitFlag && finalInstanceID != "" && len(existingInstanceIDs) <= 1 {
		   fmt.Println("🔄 Deployment progress...")
		   err = instancecmd.DisplayWorkflowResourceDataWithSpinners(cmd.Context(), token, finalInstanceID, instanceActionType)
		   if err != nil {
			   fmt.Printf("❌ Deployment failed-- %s\n", err)
		   } else {
			   fmt.Println("✅ Deployment successful")
		   }
	   }
	   if(waitFlag  && len(existingInstanceIDs) > 1){
		fmt.Println("🔄 Deployment progress...")
		err = displayMultipleInstancesProgress(cmd.Context(), token, existingInstanceIDs, instanceActionType)
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
func createInstanceUnified(ctx context.Context, token, serviceID, environmentID, productTierID, cloudProvider, region, param, paramFile string) (string, error) {
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


		// Format parameters
		formattedParams, err := common.FormatParams(param, paramFile)
		if err != nil {
			return "", err
		}

	   resourceEntity := offering.ResourceParameters[0]
	   resourceKey := resourceEntity.UrlKey
	   resourceID := resourceEntity.ResourceId
	   if resourceID == "" || resourceKey == "" {
		   return "", fmt.Errorf("invalid resource in service offering")
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
			if v == nil {
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
	if len(exitInstanceIDs) == 0 {
		return []string{}, fmt.Errorf("no matching instances found")
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
		fmt.Printf("✅ Upgraded instance: %s\n", id)
	}

	return nil
}

// displayMultipleInstancesProgress shows progress bars for multiple instances
func displayMultipleInstancesProgress(ctx context.Context, token string, instanceIDs []string, actionType string) error {
	if len(instanceIDs) == 0 {
		return nil
	}

	fmt.Printf("📊 Monitoring %d instances:\n", len(instanceIDs))
	
	// Display instance IDs being monitored
	for i, instanceID := range instanceIDs {
		fmt.Printf("  %d. Instance %s\n", i+1, instanceID)
	}
	fmt.Println()

	successCount := 0
	failureCount := 0
	var errors []error

	// Process each instance sequentially to avoid spinner conflicts
	for i, instanceID := range instanceIDs {
		fmt.Printf("🔄 Processing deployment instance %d/%d (%s)...\n", i+1, len(instanceIDs), instanceID)
		
		err := instancecmd.DisplayWorkflowResourceDataWithSpinners(ctx, token, instanceID, actionType)
		if err != nil {
			fmt.Printf("❌ Deployment instance %d (%s): Failed - %s\n", i+1, instanceID, err.Error())
			failureCount++
			errors = append(errors, err)
		} else {
			fmt.Printf("✅ Deployment instance %d (%s): Completed successfully\n", i+1, instanceID)
			successCount++
		}
		fmt.Println() // Add spacing between instances
	}
	
	// Final status summary
	fmt.Printf("📋 Final Summary:\n")
	fmt.Printf("🎯 Results: %d successful, %d failed out of %d total deployment instances\n", successCount, failureCount, len(instanceIDs))
	
	if len(errors) > 0 {
		fmt.Printf("❌ Failed deployment instances:\n")
		for i, err := range errors {
			fmt.Printf("  %d. %s\n", i+1, err.Error())
		}
		return fmt.Errorf("%d out of %d instances failed", failureCount, len(instanceIDs))
	}
	
	fmt.Println("✅ All deployment instances completed successfully!")
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
		return 0, nil
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
					"awsAccountId": awsAccountID,
					"awsBootstrapRoleAccountArn": awsBootstrapRoleARN,
				},
			}
		} else {
			yamlDoc["x-omnistrate-byoa"] = map[string]interface{}{
				"awsAccountId": awsAccountID,
				"awsBootstrapRoleAccountArn": awsBootstrapRoleARN,
			}
		}
	} else if deploymentType == "hosted" {
		if specType != "DockerCompose" {
			hostedDeployment := make(map[string]interface{})
			if awsAccountID != "" {
				hostedDeployment["awsAccountId"] = awsAccountID
				if awsBootstrapRoleARN != "" {
					hostedDeployment["awsBootstrapRoleAccountArn"] = awsBootstrapRoleARN
				}
			}
			if gcpProjectID != "" {
				hostedDeployment["gcpProjectId"] = gcpProjectID
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
				myAccount["awsAccountId"] = awsAccountID
				if awsBootstrapRoleARN != "" {
					myAccount["awsBootstrapRoleAccountArn"] = awsBootstrapRoleARN
				}
			}
			if gcpProjectID != "" {
				myAccount["gcpProjectId"] = gcpProjectID
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
						if aws, exists := byoaMap["awsAccountId"]; exists {
							if awsStr, ok := aws.(string); ok && awsAccountID == "" {
								awsAccountID = awsStr
							}
						}
					}
				}
				
				// Check hostedDeployment
				if hosted, exists := deploymentMap["hostedDeployment"]; exists {
					if hostedMap, ok := hosted.(map[string]interface{}); ok {
						if aws, exists := hostedMap["awsAccountId"]; exists {
							if awsStr, ok := aws.(string); ok && awsAccountID == "" {
								awsAccountID = awsStr
							}
						}
						if gcp, exists := hostedMap["gcpProjectId"]; exists {
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
				if aws, exists := byoaMap["awsAccountId"]; exists {
					if awsStr, ok := aws.(string); ok && awsAccountID == "" {
						awsAccountID = awsStr
					}
				}
			}
		}

		if myAccount, exists := yamlContent["x-omnistrate-my-account"]; exists {
			if myAccountMap, ok := myAccount.(map[string]interface{}); ok {
				if aws, exists := myAccountMap["awsAccountId"]; exists {
					if awsStr, ok := aws.(string); ok && awsAccountID == "" {
						awsAccountID = awsStr
					}
				}
				if gcp, exists := myAccountMap["gcpProjectId"]; exists {
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

