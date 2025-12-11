package deploy

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"time"

	"github.com/omnistrate-oss/omnistrate-ctl/internal/utils"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"
	"gopkg.in/yaml.v3"

	"github.com/chelnak/ysmrr"
	"github.com/omnistrate-oss/omnistrate-ctl/cmd/account"
	"github.com/omnistrate-oss/omnistrate-ctl/cmd/build"
	"github.com/omnistrate-oss/omnistrate-ctl/cmd/common"
	"github.com/omnistrate-oss/omnistrate-ctl/cmd/instance"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/config"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/dataaccess"
	openapiclientfleet "github.com/omnistrate-oss/omnistrate-sdk-go/fleet"
	openapiclient "github.com/omnistrate-oss/omnistrate-sdk-go/v1"
	pkgerrors "github.com/pkg/errors"
	"github.com/spf13/cobra"
)

const (
	deployExample = `
# Build and deploy using the default spec in the current directory
# Looks for omnistrate-compose.yaml, if no spec file is found, deploy falls back to build-from-repo.
omctl deploy

# Deploy using a specific Omnistrate spec
omctl deploy --file omnistrate-compose.yaml

# Build and deploy with a specific product name
omctl deploy --product-name "My Service"

# Build and deploy to a specific cloud and region
omctl deploy --cloud-provider aws --region us-east-1

# Build and deploy using BYOA (Bring Your Own Account)
omctl deploy --deployment-type byoa

# Build and deploy with instance parameters supplied inline
omctl deploy --param '{"disk_size":"20Gi", "nodes":3}'

# Build and deploy with parameters loaded from a file
omctl deploy --param-file params.json

# Build and upgrade an existing instance
omctl deploy --instance-id inst-12345

# Build from repository but skip Docker build (use pre-built image) and then deploy
omctl deploy --skip-docker-build --product-name "My Service"

# Multi-arch build from repo and deploy
omctl deploy --platforms "linux/amd64,linux/arm64"
`

	deployLong = `Deploy command is the unified entry point to build (or update) a service and then
deploy or upgrade an instance of that service.

It automatically handles:
  - Building from repository when no spec file is found
  - Building from an Omnistrate spec (such as omnistrate-compose.yaml)
  - Creating or updating the service version
  - Determining deployment type (hosted or BYOA)
  - Selecting cloud and region
  - Selecting or onboarding cloud accounts for BYOA deployments
  - Collecting instance parameters
  - Launching a new instance or upgrading an existing instance

Main modes of operation:

  - Build from repository and deploy
      Triggered when no spec file is provided and no supported spec is found in
      the current directory. The command detects a Dockerfile, builds an image,
      creates the service, generates the Omnistrate spec, and deploys an instance.

  - Build from Omnistrate spec and deploy
      Triggered when a supported spec (with x-omnistrate metadata) is provided or
      discovered. The command creates or updates the service version and deploys.

  - Upgrade an existing instance
      If --instance-id is provided, deploy builds the service version and upgrades
      the specified instance directly.

Instance selection and deployment:

  - If instances already exist in the target environment, the command can prompt
    to upgrade an existing instance or create a new one.

  - If no instances exist, a new one will be created automatically.

  - When creating a new instance, deploy determines the cloud, region, resource
    (if applicable), BYOA account (if applicable), and any required parameters.

Dry run:

  - With --dry-run, deploy performs full validation and build steps but stops
    before launching or upgrading an instance.`
)

// DeployCmd represents the deploy command
var DeployCmd = &cobra.Command{
	Use:          "deploy [--file=file] [--product-name=service-name] [--dry-run] [--deployment-type=deployment-type] [--spec-type=spec-type] [--cloud-provider=cloud] [--region=region] [--env-type=type] [--env-name=name] [--skip-docker-build] [--platforms=platforms] [--param key=value] [--param-file=file] [--instance-id=id] [--resource-id=id] [--github-user-name=username]",
	Short:        "Build or update a service and deploy or upgrade an instance",
	Long:         deployLong,
	Example:      deployExample,
	Args:         cobra.MaximumNArgs(1),
	RunE:         runDeploy,
	SilenceUsage: true,
}

func init() {
	DeployCmd.Flags().StringP("file", "f", "", fmt.Sprintf("Path to the Omnistrate spec or compose file (defaults to %s)", build.OmnistrateComposeFileName))
	DeployCmd.Flags().String("product-name", "", "Specify a custom service name. If not provided, the directory name will be used.")
	DeployCmd.Flags().Bool("dry-run", false, "Perform validation checks without actually building or deploying")
	DeployCmd.Flags().String("resource-id", "", "Specify the resource ID to use when multiple resources exist.")
	DeployCmd.Flags().String("instance-id", "", "Specify the instance ID to use when multiple deployments exist.")

	DeployCmd.Flags().StringP("environment", "e", "Prod", "Name of the environment to build the service in (default: Prod)")
	DeployCmd.Flags().StringP("environment-type", "t", "prod", "Type of environment. Valid options: dev, prod, qa, canary, staging, private (default: prod)")

	DeployCmd.Flags().String("cloud-provider", "", "Cloud provider (aws|gcp|azure)")
	DeployCmd.Flags().String("region", "", "Region code (e.g. us-east-2, us-central1)")
	DeployCmd.Flags().String("param", "", "JSON parameters for the instance deployment")
	DeployCmd.Flags().String("param-file", "", "JSON file containing parameters for the instance deployment")

	// Additional flags from build command
	DeployCmd.Flags().Bool("skip-docker-build", false, "Skip building and pushing the Docker image")
	DeployCmd.Flags().StringArray("platforms", []string{"linux/amd64"}, "Specify the platforms to build for. Example: --platforms linux/amd64 --platforms linux/arm64")
	DeployCmd.Flags().String("deployment-type", "hosted", "Type of deployment. Valid values: hosted, byoa (default \"hosted\" i.e. deployments are hosted in the service provider account)")
	DeployCmd.Flags().String("github-username", "", "GitHub username to use if GitHub API fails to retrieve it automatically")

	if err := DeployCmd.MarkFlagFilename("param-file"); err != nil {
		return
	}

	err := DeployCmd.MarkFlagFilename("file")
	if err != nil {
		return
	}
	DeployCmd.MarkFlagsRequiredTogether("environment", "environment-type")

}

func runDeploy(cmd *cobra.Command, args []string) error {
	defer config.CleanupArgsAndFlags(cmd, &args)

	fmt.Println("🚀 omctl deploy")
	fmt.Println()
	fmt.Println("Step 1/2: Service creation")
	fmt.Println("Step 2/2: Instance deployment")
	fmt.Println()

	// Step 0: Validate user is logged in first (before any spinners)
	token, err := common.GetTokenWithLogin()
	if err != nil {
		printAuthError()
		return fmt.Errorf("authentication error: %w", err)
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
		utils.PrintError(fmt.Errorf("failed to read product-name: %v", err))
		return err
	}

	// Get cloud provider account flags
	cloudProvider, err := cmd.Flags().GetString("cloud-provider")
	if err != nil {
		return err
	}
	region, err := cmd.Flags().GetString("region")
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

	environmentTypeUpper := strings.ToUpper(environmentType)

	environment, err := cmd.Flags().GetString("environment")
	if err != nil {
		utils.PrintError(err)
		return err
	}

	deploymentType, err := cmd.Flags().GetString("deployment-type")
	if err != nil {
		utils.PrintError(err)
		return err
	}

	// Validate deployment-type
	if deploymentType != build.DeploymentTypeHosted && deploymentType != build.DeploymentTypeByoa {
		err := fmt.Errorf("invalid deployment-type '%s'. Valid values are: hosted, byoa", deploymentType)
		utils.PrintError(err)
		return err
	}

	// Initialize spinner manager (only after we know we're logged in)
	sm := ysmrr.NewSpinnerManager()
	sm.Start()
	defer sm.Stop()

	// Inform user of deployment start
	spinner := sm.AddSpinner("Step 1/2: Starting service creation...")

	// Improved spec file detection: prefer service plan, then docker compose, else repo
	var specFile string
	var specType = build.DockerComposeSpecType
	var buildFromRepo = false

	// 1. If user provided a file via --file or arg, use it
	if fileExplicit && file != "" {
		specFile = file
	} else if len(args) > 0 && args[0] != "" {
		specFile = args[0]
	} else if specFile == "" {
		// Check for omnistrate-compose.yaml first (preferred)
		if _, err := os.Stat(build.OmnistrateComposeFileName); err == nil {
			specFile = build.OmnistrateComposeFileName
		} else {
			// If omnistrate-compose.yaml not found, check for docker-compose.yaml and error out
			if _, err := os.Stat(build.DockerComposeFileName); err == nil {
				spinner.Error()
				errMsg := fmt.Sprintf(
					"❌ Deployment failed\n\n"+
						"  Required file missing: %s\n\n"+
						"  Found:    %s\n"+
						"  Expected: %s\n\n"+
						"Tip:\n"+
						"  You can convert your docker-compose.yaml into Omnistrate's native format using\n"+
						"  the omnistrate-fde skill via the Omnistrate MCP Server (usable from AI agents\n"+
						"  like Claude or Gemini).\n",
					build.OmnistrateComposeFileName, build.DockerComposeFileName, build.OmnistrateComposeFileName,
				)
				utils.PrintError(fmt.Errorf("%s", errMsg))
				return pkgerrors.Wrap(err, errMsg)
			}
			// No spec files found – proceed with repo-based build
			buildFromRepo = true
		}
	}

	// Convert to absolute path if using spec file
	var absSpecFile string
	var processedData []byte
	if specFile != "" {
		absSpecFile, err = filepath.Abs(specFile)
		if err != nil {
			return pkgerrors.Wrap(err, "failed to get absolute path for spec file")
		}

		// Check if spec file exists
		if _, err := os.Stat(absSpecFile); os.IsNotExist(err) {
			err := fmt.Errorf("spec file does not exist: %s", absSpecFile)
			utils.PrintError(err)
			return err
		}

		// Read and process spec file for pre-checks
		fileData, err := os.ReadFile(absSpecFile)
		if err != nil {
			return pkgerrors.Wrap(err, "failed to read spec file")
		}

		// Process template expressions recursively
		processedData, err = processTemplateExpressions(fileData, filepath.Dir(absSpecFile))
		if err != nil {
			return pkgerrors.Wrap(err, "failed to process template expressions")
		}

		// Check for omnistrate-specific configurations
		var planCheck map[string]interface{}
		if err := yaml.Unmarshal(processedData, &planCheck); err == nil {
			// Check if this is an omnistrate spec file
			isOmnistrate := build.ContainsOmnistrateKey(planCheck)
			if !isOmnistrate {
				err := fmt.Errorf(
					"spec file '%s' is missing Omnistrate configuration (x-omnistrate-* keys).\n"+
						"This looks like a plain docker-compose or non-Omnistrate YAML file.\n\n"+
						"Next steps:\n"+
						"  - Add x-omnistrate-* keys to your spec, or\n"+
						"  - Convert your compose file using Omnistrate tools as described in the docs",
					specFile,
				)
				utils.PrintError(err)
				spinner.Error()
				return err
			}
			// Use the common function to detect spec type
			specType = build.DetectSpecType(planCheck)
		} else {
			// Fallback to file extension based detection
			fileToRead := filepath.Base(absSpecFile)
			if fileToRead == build.PlanSpecFileName {
				specType = build.ServicePlanSpecType
			} else {
				specType = build.DockerComposeSpecType
			}
		}
	}

	spinner.UpdateMessage("Step 1/2: Checking cloud provider accounts...")

	isAccountId := false
	awsAccountID, awsBootstrapRoleARN, gcpProjectID, gcpProjectNumber, gcpServiceAccountEmail, azureSubscriptionID, azureTenantID, extractDeploymentType :=
		extractCloudAccountsFromProcessedData(processedData)

	if awsAccountID != "" || gcpProjectID != "" || azureSubscriptionID != "" {
		isAccountId = true
	}

	// Explain precedence between spec deploymentType and CLI deploymentType
	if extractDeploymentType != "" && extractDeploymentType != deploymentType {
		utils.PrintWarning(
			fmt.Sprintf(
				"⚠️ deployment-type override:\n  Spec file: %s\n  CLI flag: %s\n  Using:    %s (CLI value has precedence)",
				extractDeploymentType, deploymentType, deploymentType,
			),
		)
	}

	// If no cloud provider is set, we will figure out based on accounts / offering later
	var cloudProvidersToCheck []string

	if awsAccountID != "" {
		cloudProvidersToCheck = append(cloudProvidersToCheck, "aws")
	}
	if gcpProjectID != "" {
		cloudProvidersToCheck = append(cloudProvidersToCheck, "gcp")
	}
	if azureSubscriptionID != "" {
		cloudProvidersToCheck = append(cloudProvidersToCheck, "azure")
	}

	// If spec does not constrain providers, check all
	if len(cloudProvidersToCheck) == 0 {
		cloudProvidersToCheck = []string{"aws", "gcp", "azure"}
	}

	allAccounts := []*openapiclient.DescribeAccountConfigResult{}
	// Filter for READY accounts and collect status information
	readyAccounts := []*openapiclient.DescribeAccountConfigResult{}
	accountStatusSummary := make(map[string]int)
	var foundMatchingAccount bool
	var accountStatus string

	for _, cp := range cloudProvidersToCheck {
		// Pre-check 1: Check for linked cloud provider accounts
		accounts, err := dataaccess.ListAccounts(cmd.Context(), token, cp)
		if err != nil {
			spinner.Error()
			printBackendError("cloud provider account lookup", err)
			return err
		}
		for _, acc := range accounts.AccountConfigs {
			allAccounts = append(allAccounts, &acc)
			if acc.Status == "READY" {
				readyAccounts = append(readyAccounts, &acc)
				if awsAccountID == "" && acc.AwsAccountID != nil {
					awsAccountID = *acc.AwsAccountID
					if acc.AwsBootstrapRoleARN != nil {
						awsBootstrapRoleARN = *acc.AwsBootstrapRoleARN
					}
					foundMatchingAccount = true
					accountStatus = acc.Status
				}

				if gcpProjectID == "" && acc.GcpProjectID != nil {
					gcpProjectID = *acc.GcpProjectID
					gcpProjectNumber = *acc.GcpProjectNumber
					if acc.GcpServiceAccountEmail != nil {
						gcpServiceAccountEmail = *acc.GcpServiceAccountEmail
					}
					foundMatchingAccount = true
					accountStatus = acc.Status

				}
				if azureSubscriptionID == "" && acc.AzureSubscriptionID != nil {
					azureSubscriptionID = *acc.AzureSubscriptionID
					azureTenantID = *acc.AzureTenantID
					foundMatchingAccount = true
					accountStatus = acc.Status
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

	if !foundMatchingAccount && (awsAccountID != "" || gcpProjectID != "" || azureSubscriptionID != "") {

		var errorMessage string
		if awsAccountID != "" {
			errorMessage += fmt.Sprintf("AWS account ID %s is not linked. Please link it using 'omctl account create'.\n", awsAccountID)
		}
		if gcpProjectID != "" {
			errorMessage += fmt.Sprintf("GCP project %s/%s is not linked. Please link it using 'omctl account create'.\n", gcpProjectID, gcpProjectNumber)
		}
		if azureSubscriptionID != "" {
			errorMessage += fmt.Sprintf("Azure subscription %s/%s is not linked. Please link it using 'omctl account create'.", azureSubscriptionID, azureTenantID)
		}

		spinner.Error()
		utils.PrintError(fmt.Errorf("cloud account mismatch:\n%s", errorMessage))
		return fmt.Errorf("cloud account mismatch: %s", errorMessage)
	} else if accountStatus != "READY" && (awsAccountID != "" || gcpProjectID != "" || azureSubscriptionID != "") {

		var errorMessage string
		if awsAccountID != "" {
			errorMessage += fmt.Sprintf("AWS account ID %s is linked but has status '%s'. Complete onboarding if required.\n", awsAccountID, accountStatus)
		}
		if gcpProjectID != "" {
			errorMessage += fmt.Sprintf("GCP project %s/%s is linked but has status '%s'. Complete onboarding if required.\n", gcpProjectID, gcpProjectNumber, accountStatus)
		}
		if azureSubscriptionID != "" {
			errorMessage += fmt.Sprintf("Azure subscription %s/%s is linked but has status '%s'. Complete onboarding if required.", azureSubscriptionID, azureTenantID, accountStatus)
		}
		spinner.Error()
		utils.PrintError(fmt.Errorf("cloud account not ready:\n%s", errorMessage))
		return fmt.Errorf("cloud account not ready: %s", errorMessage)
	}

	if awsAccountID == "" && gcpProjectID == "" && azureSubscriptionID == "" {

		// Ensure at least one READY account is available
		if len(readyAccounts) == 0 {
			if len(allAccounts) > 0 {
				utils.PrintError(fmt.Errorf(
					"❌ No READY cloud provider accounts found\n\n"+
						"  Your organization has %d cloud account(s), but none are in READY status.\n"+
						"  Non-READY accounts may need to complete onboarding or have configuration issues.\n\n"+
						"Next steps:\n"+
						"  1. Check existing account status: omctl account list\n"+
						"  2. Complete onboarding for existing accounts, or\n"+
						"  3. Create a new READY account: omctl account create",
					len(allAccounts),
				))
				utils.HandleSpinnerError(spinner, sm, err)
				spinner.UpdateMessage("Step 1/2: Service creation requires at least one READY cloud provider account")
				spinner.Error()
				return fmt.Errorf("deployment requires at least one READY cloud provider account")
			} else {
				// No accounts at all: start interactive account creation flow
				sm.Stop()
				fmt.Println("No cloud provider accounts found. Starting cloud account creation flow...")

				// Determine which cloud provider to use and get credentials
				if cloudProvider == "" {
					cloudProvider = promptForCloudProvider()
				}

				// Get cloud-specific credentials
				paramsJSON, err := promptForCloudCredentials(cloudProvider)
				if err != nil {
					return fmt.Errorf("failed to get cloud credentials: %w", err)
				}

				// Parse the JSON to extract credentials
				var paramsMap map[string]interface{}
				if err := json.Unmarshal([]byte(paramsJSON), &paramsMap); err != nil {
					return fmt.Errorf("failed to parse credentials: %w", err)
				}

				// Create account params based on cloud provider
				accountParams := account.CloudAccountParams{
					Name: fmt.Sprintf("%s-account-%d", cloudProvider, time.Now().Unix()),
				}

				switch cloudProvider {
				case "aws":
					if awsAccountID, ok := paramsMap["aws_account_id"].(string); ok {
						accountParams.AwsAccountID = awsAccountID
					}
				case "gcp":
					if gcpProjectID, ok := paramsMap["gcp_project_id"].(string); ok {
						accountParams.GcpProjectID = gcpProjectID
					}
					if gcpProjectNumber, ok := paramsMap["gcp_project_number"].(string); ok {
						accountParams.GcpProjectNumber = gcpProjectNumber
					}
				case "azure":
					if azureSubscriptionID, ok := paramsMap["azure_subscription_id"].(string); ok {
						accountParams.AzureSubscriptionID = azureSubscriptionID
					}
					if azureTenantID, ok := paramsMap["azure_tenant_id"].(string); ok {
						accountParams.AzureTenantID = azureTenantID
					}
				}

				// Create the cloud provider account
				sm.Start()
				spinner = sm.AddSpinner("Creating cloud provider account...")
				accountData, err := account.CreateCloudAccount(cmd.Context(), token, accountParams, spinner, sm)
				if err != nil || accountData == nil {
					utils.PrintError(fmt.Errorf("failed to create cloud provider account: %v", err))
					spinner.Error()
					return err
				}
				dataaccess.PrintNextStepVerifyAccountMsg(accountData)
				// Wait for account to become READY (poll up to 10 min)
				err = waitForAccountReady(cmd.Context(), token, accountData.Id)
				if err != nil {
					utils.PrintError(fmt.Errorf("account did not become READY: %v", err))
					spinner.Error()
					return err
				}
				spinner.Complete()
			}
		}

	}
	if awsAccountID != "" || gcpProjectID != "" || azureSubscriptionID != "" {
		spinner.Complete()
		spinner = sm.AddSpinner("Cloud account(s) linked and READY")
		spinner.Complete()

		if awsAccountID != "" {
			spinner = sm.AddSpinner(fmt.Sprintf("  - Using AWS Account ID: %s", awsAccountID))
			spinner.Complete()

		}
		if gcpProjectID != "" {
			spinner = sm.AddSpinner(fmt.Sprintf("  - Using GCP Project ID: %s and Project Number: %s", gcpProjectID, gcpProjectNumber))
			spinner.Complete()

		}
		if azureSubscriptionID != "" {
			spinner = sm.AddSpinner(fmt.Sprintf("  - Using Azure Subscription ID: %s and Tenant ID: %s", azureSubscriptionID, azureTenantID))
			spinner.Complete()

		}

		spinner = sm.AddSpinner("Step 1/2: Cloud provider account check complete")
		spinner.Complete()

	}

	// Pre-check 2: Determine service name
	spinner = sm.AddSpinner("Step 1/2: Determining service name...")

	var serviceNameToUse string
	serviceNameToUse = productName
	if serviceNameToUse == "" {
		if specFile == "" {
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

	spinner.UpdateMessage(fmt.Sprintf("Step 1/2: Service name resolved: %s", serviceNameToUse))
	spinner.Complete()

	// Pre-check 3: Check if service exists and validate service plan count
	spinner = sm.AddSpinner(fmt.Sprintf("Step 1/2: Checking for existing service '%s'...", serviceNameToUse))
	existingServiceID, err := findExistingService(cmd.Context(), token, serviceNameToUse)
	if err != nil {
		utils.HandleSpinnerError(spinner, sm, err)
		utils.PrintError(fmt.Errorf("failed to check existing service: %v", err))
		return err
	}

	if existingServiceID != "" {
		spinner.UpdateMessage(fmt.Sprintf("Step 1/2: Existing service detected: %s (ID: %s)", serviceNameToUse, existingServiceID))
		spinner.Complete()
	} else {
		spinner.UpdateMessage(fmt.Sprintf("Step 1/2: New service will be created: %s", serviceNameToUse))
		spinner.Complete()
	}

	// Step 3: Build service in target environment with release-as-preferred
	spinner = sm.AddSpinner(fmt.Sprintf("Step 1/2: Building service '%s'...", serviceNameToUse))

	var serviceID, environmentID, planID string
	var undefinedResources map[string]string

	if specType == build.DockerComposeSpecType && buildFromRepo {
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
			deploymentType,
			awsAccountID,
			gcpProjectID,
			gcpProjectNumber,
			azureSubscriptionID,
			azureTenantID,
			sm,
			file,
			[]string{},
			platforms,
			false,
		)
		if err != nil {
			utils.HandleSpinnerError(spinner, sm, err)
			wrapAndPrintServiceBuildError(err)
			return err
		}

	} else {

		if !isAccountId {
			// Use createDeploymentYAML to generate the deployment section
			deploymentSection := createDeploymentYAML(
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
			// Marshal the deployment section to YAML
			deploymentYAML, err := yaml.Marshal(deploymentSection)
			if err != nil {
				utils.PrintError(fmt.Errorf("failed to marshal deployment section: %w", err))
				return err
			}
			if deploymentYAML != nil {
				composeMap := map[string]interface{}{}
				if err := yaml.Unmarshal(processedData, &composeMap); err != nil {
					return pkgerrors.Wrap(err, "failed to parse compose YAML for injection")
				}
				depMap := map[string]interface{}{}
				if err := yaml.Unmarshal(deploymentYAML, &depMap); err == nil {
					if specType != build.DockerComposeSpecType {
						// Inject deployment info under each service
						if services, ok := composeMap["services"].(map[string]interface{}); ok {
							for svcName, svcVal := range services {
								svcMap, ok := svcVal.(map[string]interface{})
								if !ok {
									continue
								}
								for k, v := range depMap {
									svcMap[k] = v
								}
								services[svcName] = svcMap
							}
							composeMap["services"] = services
						}else {
						// Inject deployment info at root level
						for k, v := range depMap {
							composeMap[k] = v
						}
					}
					} else {
						// Inject deployment info at root level
						for k, v := range depMap {
							composeMap[k] = v
						}
					}
				}
				finalYAML, err := yaml.Marshal(composeMap)
				if err != nil {
					return pkgerrors.Wrap(err, "failed to marshal final compose YAML")
				}
				processedData = finalYAML
			}
		}

		serviceID, environmentID, planID, undefinedResources, _, err = build.BuildService(
			cmd.Context(),
			processedData,
			token,
			serviceNameToUse,
			specType,
			nil,
			nil,
			&environment,
			&environmentTypeUpper,
			true,
			true,
			nil,
			dryRun,
			false,
		)
		if err != nil {
			utils.HandleSpinnerError(spinner, sm, err)
			wrapAndPrintServiceBuildError(err)
			return err
		}

	}

	// Dry-run exit point
	if dryRun {
		spinner.UpdateMessage("Step 1/2: Simulated service build completed successfully (dry run)")
		spinner.Complete()
		fmt.Println()
		fmt.Println("🔍 Dry-run mode: Validation checks only. No service or instance was created.")
		fmt.Println("✅ Authentication check passed.")
		fmt.Println("✅ Service spec and deployment configuration validated.")
		fmt.Println("To proceed with actual deployment, run the command without the --dry-run flag.")
		return nil
	}
	spinner.UpdateMessage(fmt.Sprintf("Step 1/2: Built service '%s' in environment %s (%s), Service ID: %s", serviceNameToUse, environment, environmentTypeUpper, serviceID))
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
	err = executeDeploymentWorkflow(cmd, sm, token, serviceID, environmentID, planID, serviceNameToUse, environment, environmentTypeUpper, instanceID, cloudProvider, region, param, paramFile, resourceID, deploymentType)
	if err != nil {
		return err
	}

	return nil
}

// executeDeploymentWorkflow handles the complete post-service-build deployment workflow
// This function is reusable for both deploy and build_simple commands
func executeDeploymentWorkflow(cmd *cobra.Command, sm ysmrr.SpinnerManager, token, serviceID, environmentID, planID, serviceName, environment, environmentTypeUpper, instanceID, cloudProvider, region, param, paramFile, resourceID, deploymentType string) error {

	// Step 7: Set service plan as preferred in environment
	spinner := sm.AddSpinner(fmt.Sprintf("Step 1/2: Setting service plan as preferred in %s...", environment))

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
	spinner.UpdateMessage(fmt.Sprintf("Step 1/2: Service plan set as preferred in %s (version %s)", environment, targetVersion))
	spinner.Complete()

	// Step 9: Create or upgrade instance deployment automatically

	var finalInstanceID string
	instanceActionType := "create"

	spinnerMsg := "Step 2/2: Preparing instance deployment"
	spinner = sm.AddSpinner(spinnerMsg)

	if instanceID != "" {
		spinnerMsg = "Step 2/2: Checking for existing instance"
		spinner.UpdateMessage(spinnerMsg)

		var existingInstanceIDs []string
		existingInstanceIDs, _, err = listInstances(cmd.Context(), token, serviceID, environmentID, planID, instanceID, "excludeCloudAccounts")
		if err != nil {
			utils.HandleSpinnerError(spinner, sm, err)
			spinner.UpdateMessage(spinnerMsg + ": Failed (" + err.Error() + ")")
			spinner.Error()
			return err
		}

		if instanceID != "" && len(existingInstanceIDs) == 0 {
			spinner.UpdateMessage(fmt.Sprintf("%s: No existing instance found for instance ID: %s (provided instance does not match)", spinnerMsg, instanceID))
			spinner.Error()
			fmt.Println()
			fmt.Println("❌ No instance found with the given --instance-id.")
			fmt.Printf("   Instance ID: %s\n\n", instanceID)
			fmt.Println("Next steps:")
			fmt.Println("  - Check the instance ID value.")
			fmt.Println("  - List instances for this service: omctl instance list --service", serviceName)
			return nil
		}

		// Display automatic instance handling message
		if len(existingInstanceIDs) > 0 {
			finalInstanceID = existingInstanceIDs[0]
			spinner.UpdateMessage(fmt.Sprintf("Step 2/2: %s: Found %d existing instance(s)", spinnerMsg, len(existingInstanceIDs)))
			spinner.Complete()

			// Show the note directly without stopping spinner manager
			spinner = sm.AddSpinner(fmt.Sprintf("Step 2/2: 📝 Note: Existing instance found. An upgrade will be performed. (Instance ID: %s)", finalInstanceID))
			spinner.Complete()

		} else {

			spinner.UpdateMessage(fmt.Sprintf("Step 2/2: %s: No existing instance found (provider instance does not match)", spinnerMsg))
			spinner.Complete()

		}
	} else {
		spinner.Complete()
		// Show the note directly
		spinner = sm.AddSpinner("Step 2/2: 📝 Note: No existing instance specified. A new instance will be created automatically.")
		spinner.Complete()

	}

	if finalInstanceID != "" {

		spinner = sm.AddSpinner(fmt.Sprintf("Step 2/2: Upgrading existing instance %s to latest version...", finalInstanceID))
		spinner.Complete()
		spinner = sm.AddSpinner("Step 2/2: Upgrading existing instance")
		upgradeErr := upgradeExistingInstance(cmd.Context(), token, []string{finalInstanceID}, serviceID, planID)
		instanceActionType = "upgrade"
		if upgradeErr != nil {
			utils.HandleSpinnerError(spinner, sm, upgradeErr)
			spinner.UpdateMessage(fmt.Sprintf("Step 2/2: Upgrading existing instance: Failed (%s)", upgradeErr.Error()))
			spinner.Error()
			return upgradeErr
		} else {

			spinner.UpdateMessage(fmt.Sprintf("Step 2/2: Upgrading existing instance: Success (ID: %s)", finalInstanceID))
			spinner.Complete()
		}

	} else {

		// Format parameters
		formattedParams, err := common.FormatParams(param, paramFile)
		if err != nil {
			return err
		}

		// If deployment type is BYOA, create cloud account instances first
		if deploymentType == build.DeploymentTypeByoa && formattedParams["cloud_provider_account_config_id"] == nil {
			// Initialize formattedParams if it's nil
			if formattedParams == nil {
				formattedParams = make(map[string]any)
			}

			fmt.Printf("BYOA deployment detected. Creating cloud account instance...\n")
			cloudAccountInstanceID, targetCloudProvider, err := createCloudAccountInstances(cmd.Context(), token, serviceID, environmentID, planID, cloudProvider, sm)
			if err != nil {
				fmt.Printf("Warning: Failed to create cloud account instances: %v\n", err)
			}
			cloudProvider = targetCloudProvider
			fmt.Printf("Cloud account instance created: provider=%s, id=%s\n", targetCloudProvider, cloudAccountInstanceID)
			formattedParams["cloud_provider_account_config_id"] = cloudAccountInstanceID

		}

		createMsg := "Step 2/2: Deploying a new instance"
		spinner = sm.AddSpinner(createMsg)
		createdInstanceID, err := "", error(nil)
		createdInstanceID, err = createInstanceUnified(cmd.Context(), token, serviceID, environmentID, planID, cloudProvider, region, resourceID, "resourceInstance", formattedParams, sm)
		finalInstanceID = createdInstanceID
		// instanceActionType is already "create" from initialization
		if err != nil {
			utils.HandleSpinnerError(spinner, sm, err)
			spinner.UpdateMessage(fmt.Sprintf("%s: Failed (%s)", createMsg, err.Error()))
			if isMissingParamsError(err) {
				printMissingParamsGuidance(err)
			}
			spinner.Error()
			return err
		}

		spinner.UpdateMessage(fmt.Sprintf("%s: Success (ID: %s)", createMsg, finalInstanceID))
		spinner.Complete()
	}
	spinner = sm.AddSpinner("Step 2/2: Instance deployment preparation complete")
	spinner.Complete()

	// Stop spinner manager before printing summary
	sm.Stop()

	// Success summary
	fmt.Println()
	fmt.Println("✅ Deployment summary")
	fmt.Println()
	fmt.Printf("  Service:\n")
	fmt.Printf("    Name:        %s\n", serviceName)
	fmt.Printf("    ID:          %s\n", serviceID)
	fmt.Printf("    Environment: %s (%s)\n", environment, environmentTypeUpper)
	fmt.Printf("    Plan ID:     %s\n", planID)
	if finalInstanceID != "" {
		fmt.Println()
		fmt.Printf("  Instance:\n")
		fmt.Printf("    Action:      %s\n", instanceActionType)
		fmt.Printf("    ID:          %s\n", finalInstanceID)
	}
	fmt.Println()
	fmt.Println("🔄 Deployment progress...")

	// Optionally display workflow progress if desired
	if finalInstanceID != "" {
		err = instance.DisplayWorkflowResourceDataWithSpinners(cmd.Context(), token, finalInstanceID, instanceActionType)
		if err != nil {
			fmt.Printf("❌ Deployment workflow failed: %s\n", err)
			return err
		} else {
			fmt.Println("✅ Deployment successful")
		}
	}

	return nil
}

// createInstanceUnified creates an instance with or without subscription, removing duplicate code
func createInstanceUnified(ctx context.Context, token, serviceID, environmentID, productTierID, cloudProvider, region, resourceID, instanceType string, formattedParams map[string]interface{}, sm ysmrr.SpinnerManager) (string, error) {

	// Get the latest version
	version, err := dataaccess.FindLatestVersion(ctx, token, serviceID, productTierID)
	if err != nil {
		return "", fmt.Errorf("failed to find latest version: %w", err)
	}

	// Describe service offering
	res, err := dataaccess.ExternalDescribeServiceOffering(ctx, token, serviceID, environmentID, productTierID)
	if err != nil {
		return "", fmt.Errorf("failed to describe service offering: %w", err)
	}

	if len(res.Offerings) == 0 {
		return "", fmt.Errorf("no service offerings found")
	}

	offering := res.Offerings[0]

	// Create default parameters with common sensible defaults
	defaultParams := map[string]interface{}{}
	resourceKey := ""

	if instanceType == "cloudAccount" {
		defaultParams = formattedParams

		// For cloud account instances, find the injected account config resource
		var accountConfigResource *openapiclient.ResourceEntity
		for _, param := range offering.ResourceParameters {
			if strings.HasPrefix(param.ResourceId, "r-injectedaccountconfig") {
				accountConfigResource = &param
				// Use the resource ID as the key for cloud account resources
				resourceKey = param.UrlKey
				resourceID = param.ResourceId
				break
			}
		}

		if accountConfigResource == nil {
			return "", fmt.Errorf("no injected account config resource found for BYOA deployment")
		}

		if offering.ServiceModelType != "BYOA" {
			return "", fmt.Errorf("cloud account instances are only supported for BYOA service model, got: %s", offering.ServiceModelType)
		}

		fmt.Printf("Found cloud account resource: ID=%s, Key=%s\n", resourceID, resourceKey)
	} else {
		// Get list of resources in the target tier version
		resources, err := dataaccess.ListResources(ctx, token, serviceID, productTierID, &version)
		if err != nil {
			return "", fmt.Errorf("no resources found in service plan: %w", err)
		}

		// Remove resources with internal:true (fix type error)
		filteredResources := make([]openapiclient.DescribeResourceResult, 0, len(resources.Resources))
		for _, r := range resources.Resources {
			// Defensive: check for Internal field via reflection if not present
			hasInternal := false
			v := reflect.ValueOf(r)
			field := v.FieldByName("Internal")
			if field.IsValid() && field.Kind() == reflect.Bool {
				hasInternal = field.Bool()
			}
			if hasInternal {
				continue
			}
			filteredResources = append(filteredResources, r)
		}
		resources.Resources = filteredResources
		if len(resources.Resources) == 0 {
			return "", fmt.Errorf("no resources found in service plan (after filtering internal resources)")
		}

		// First: determine resourceKey and resourceID (multi-resource UX)
		if resourceID != "" {
			for _, resource := range resources.Resources {
				if resource.Id == resourceID {
					resourceKey = resource.Key
					resourceID = resource.Id
					break
				}
			}
			if resourceKey == "" {
				return "", fmt.Errorf("resource ID '%s' not found in service plan", resourceID)
			}
		}

		if resourceKey == "" {
			if len(resources.Resources) == 1 {
				resourceKey = resources.Resources[0].Key
				resourceID = resources.Resources[0].Id
			}
			if len(resources.Resources) > 1 {
				// Stop spinner before prompting user
				sm.Stop()

				fmt.Println("Multiple resources found in service plan. Please select one:")
				for idx, resource := range resources.Resources {
					fmt.Printf("  %d. Name: %s, Key: %s, ID: %s\n", idx+1, resource.Name, resource.Key, resource.Id)
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

				// Restart spinner after user input
				sm.Start()
			}
		}

		if resourceID == "" || resourceKey == "" {
			return "", fmt.Errorf("invalid resource in service plan: missing ID or key")
		}

		// Select default cloudProvider and region from offering.CloudProviders if available

		if len(offering.CloudProviders) > 0 && cloudProvider != "" {
			found := false
			for _, cp := range offering.CloudProviders {
				if cp == cloudProvider {
					found = true
					break
				}
			}
			if !found {
				// fallback to first available provider, but explain
				return "", fmt.Errorf("cloud provider '%s' is not supported for this service plan. Supported providers: %v", cloudProvider, offering.CloudProviders)
			}
		}

		if cloudProvider == "" && region == "" {
			if len(offering.CloudProviders) > 0 {
				cloudProvider = offering.CloudProviders[0]
			} else {
				return "", fmt.Errorf("no cloud providers available for this service plan")
			}

		}

		if cloudProvider == "" && region != "" {
			// If region is specified but not cloud provider, try to infer cloud provider from region

			gcpRegions := offering.GcpRegions
			awsRegions := offering.AwsRegions
			azureRegions := offering.AzureRegions

			// Check GCP regions first
			for _, gcpRegion := range gcpRegions {
				if gcpRegion == region {
					cloudProvider = "gcp"
					break
				}
			}

			// check AWS regions
			if cloudProvider == "" {
				for _, awsRegion := range awsRegions {
					if awsRegion == region {
						cloudProvider = "aws"
						break
					}
				}
			}

			// check Azure regions
			if cloudProvider == "" {
				for _, azureRegion := range azureRegions {
					if azureRegion == region {
						cloudProvider = "azure"
						break
					}
				}
			}

			// If not found in any provider, return error
			if cloudProvider == "" {
				return "", fmt.Errorf("unknown region '%s'. Please specify a valid cloud provider", region)
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
			if region == "" && len(regions) > 0 {
				found = true // skip check if region is not specified
				region = regions[0]
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
			case "azure":
				region = "eastus2"
			}
		}

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
						case "subscriptionId", "cloud_provider", "region":
							continue
						default:
							// Handle custom parameters
							if inputParam.Required {
								if formattedParams[inputParam.Key] != nil {
									defaultParams[inputParam.Key] = formattedParams[inputParam.Key]
								} else if inputParam.DefaultValue != nil {
									defaultParams[inputParam.Key] = *inputParam.DefaultValue
								} else {
									defaultParams[inputParam.Key] = nil
								}

							}
							if !inputParam.Required {
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
				continue
			}
			if reflect.TypeOf(v).Kind() == reflect.String && v == "" {
				defaultRequiredParams = append(defaultRequiredParams, k)
			}
		}

		// Validate that all required parameters have values
		if len(defaultRequiredParams) > 0 {
			return "", fmt.Errorf("missing required parameters for instance creation: %v", defaultRequiredParams)
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
			fmt.Printf("⚠️  Warning: The following parameters were provided but are not supported by this service and will be ignored:\n")
			for _, param := range unusedParams {
				fmt.Printf("   - %s\n", param)
			}
			fmt.Println()
		}
	}

	request := openapiclientfleet.FleetCreateResourceInstanceRequest2{
		CloudProvider: &cloudProvider,
		RequestParams: defaultParams,
		NetworkType:   nil,
	}
	if instanceType == "cloudAccount" {
		networkType := "INTERNAL"
		request.NetworkType = &networkType

	}
	if instanceType == "resourceInstance" {

		request.Region = &region
		request.ProductTierVersion = &version
	}

	//    Create the instance
	instance, err := dataaccess.CreateResourceInstance(ctx, token,
		res.ServiceProviderId,
		res.ServiceURLKey,
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
func listInstances(ctx context.Context, token, serviceID, environmentID, servicePlanID, instanceID, filter string) ([]string, []struct {
	cloudProvider string
	instanceID    string
	status        string
}, error) {

	res, err := dataaccess.ListResourceInstance(ctx, token, serviceID, environmentID,
		&dataaccess.ListResourceInstanceOptions{
			ProductTierId: &servicePlanID,
			Filter:        &filter,
		},
	)
	if err != nil {
		return []string{}, []struct {
			cloudProvider string
			instanceID    string
			status        string
		}{}, fmt.Errorf("failed to search for instances: %w", err)
	}

	exitInstanceIDs := make([]string, 0)
	seenIDs := make(map[string]bool)
	instances := make([]struct {
		cloudProvider string
		instanceID    string
		status        string
	}, 0)

	if len(res.ResourceInstances) == 0 {
		return []string{}, instances, nil
	}
	for _, instance := range res.ResourceInstances {
		var idStr string
		if instance.ConsumptionResourceInstanceResult.Id != nil {
			idStr = *instance.ConsumptionResourceInstanceResult.Id
		} else {
			idStr = "<nil>"
		}

		instances = append(instances, struct {
			cloudProvider string
			instanceID    string
			status        string
		}{
			cloudProvider: instance.CloudProvider,
			instanceID:    idStr,
			status:        *instance.ConsumptionResourceInstanceResult.Status,
		})

		// Prioritize adding instanceID if specified

		// Priority: instanceID  > servicePlanID
		if instanceID != "" && idStr == instanceID {
			if !seenIDs[idStr] {
				exitInstanceIDs = append(exitInstanceIDs, idStr)
				seenIDs[idStr] = true
			}
		} else if instanceID == "" {
			if idStr != "" && !seenIDs[idStr] {
				exitInstanceIDs = append(exitInstanceIDs, idStr)
				seenIDs[idStr] = true
			}
		}
	}

	return exitInstanceIDs, instances, nil
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

// findAllOmnistrateServicePlanBlocks recursively finds all x-omnistrate-service-plan blocks in any YAML document
func findAllOmnistrateServicePlanBlocks(yamlContent interface{}) []map[string]interface{} {
	var results []map[string]interface{}
	var search func(node interface{})
	search = func(node interface{}) {
		switch v := node.(type) {
		case map[string]interface{}:
			for k, val := range v {
				if k == "x-omnistrate-service-plan" {
					if block, ok := val.(map[string]interface{}); ok {
						results = append(results, block)
					}
				}
				// Recurse into all values
				search(val)
			}
		case []interface{}:
			for _, item := range v {
				search(item)
			}
		}
	}
	search(yamlContent)
	return results
}

// extractCloudAccountsFromProcessedData extracts cloud provider account information from the YAML content
func extractCloudAccountsFromProcessedData(processedData []byte) (awsAccountID, awsBootstrapRoleARN, gcpProjectID, gcpProjectNumber, gcpServiceAccountEmail, azureSubscriptionID, azureTenantID, extractDeploymentType string) {
	if len(processedData) == 0 {
		return "", "", "", "", "", "", "", ""
	}

	// Helper to validate and set deployment type
	setDeploymentType := func(deployType string) {
		if deployType == build.DeploymentTypeHosted || deployType == build.DeploymentTypeByoa {
			extractDeploymentType = deployType
		}
		// Invalid deployment types are ignored, keeping the previous valid value or empty string
	}

	// Simple helper to get string value with multiple key variations
	getFirstString := func(m map[string]interface{}, keys ...string) string {
		for _, key := range keys {
			if v, ok := m[key].(string); ok && v != "" {
				return v
			}
			if v, ok := m[key].(int); ok {
				return fmt.Sprintf("%d", v)
			}
		}
		return ""
	}

	// Helper to extract account info from any map
	extractAccountDetails := func(m map[string]interface{}) {
		if awsAccountID == "" {
			awsAccountID = getFirstString(m, "awsAccountId", "awsAccountID", "AwsAccountID", "AwsAccountId")
		}
		if awsBootstrapRoleARN == "" {
			awsBootstrapRoleARN = getFirstString(m, "awsBootstrapRoleAccountArn", "awsBootstrapRoleARN", "AwsBootstrapRoleARN", "awsBootstrapRoleArn", "AwsBootstrapRoleArn")
		}
		if gcpProjectID == "" {
			gcpProjectID = getFirstString(m, "gcpProjectId", "gcpProjectID", "GcpProjectID", "GcpProjectId")
		}
		if gcpProjectNumber == "" {
			gcpProjectNumber = getFirstString(m, "gcpProjectNumber", "GcpProjectNumber")
		}
		if gcpServiceAccountEmail == "" {
			gcpServiceAccountEmail = getFirstString(m, "gcpServiceAccountEmail", "GcpServiceAccountEmail")
		}
		if azureSubscriptionID == "" {
			azureSubscriptionID = getFirstString(m, "azureSubscriptionId", "azureSubscriptionID", "AzureSubscriptionID", "AzureSubscriptionId")
		}
		if azureTenantID == "" {
			azureTenantID = getFirstString(m, "azureTenantId", "azureTenantID", "AzureTenantID", "AzureTenantId")
		}
	}

	// Helper to process deployment sections (hosted/byoa)
	processDeploymentMap := func(depMap map[string]interface{}) {
		if hosted, exists := depMap["hostedDeployment"]; exists {
			if hostedMap, ok := hosted.(map[string]interface{}); ok {
				setDeploymentType("hosted")
				extractAccountDetails(hostedMap)
			}
		}
		if byoa, exists := depMap["byoaDeployment"]; exists {
			if byoaMap, ok := byoa.(map[string]interface{}); ok {
				setDeploymentType(build.DeploymentTypeByoa)
				extractAccountDetails(byoaMap)
			}
		}
	}

	// Parse YAML content directly
	var yamlContent map[string]interface{}
	if err := yaml.Unmarshal(processedData, &yamlContent); err != nil {
		return "", "", "", "", "", "", "", "" // Return empty values if YAML is invalid
	}

	// Check direct x-omnistrate-byoa/hosted keys
	if byoa, exists := yamlContent["x-omnistrate-byoa"]; exists {
		if byoaMap, ok := byoa.(map[string]interface{}); ok {
			setDeploymentType(build.DeploymentTypeByoa)
			extractAccountDetails(byoaMap)
		}
	}
	if hosted, exists := yamlContent["x-omnistrate-hosted"]; exists {
		if hostedMap, ok := hosted.(map[string]interface{}); ok {
			setDeploymentType("hosted")
			extractAccountDetails(hostedMap)
		}
	}

	// Check x-omnistrate-service-plan
	if sp, exists := yamlContent["x-omnistrate-service-plan"]; exists {
		if spMap, ok := sp.(map[string]interface{}); ok {
			extractAccountDetails(spMap)
			if deployment, exists := spMap["deployment"]; exists {
				if depMap, ok := deployment.(map[string]interface{}); ok {
					processDeploymentMap(depMap)
				}
			}
		}
	}

	// Check top-level deployment
	if deployment, exists := yamlContent["deployment"]; exists {
		if depMap, ok := deployment.(map[string]interface{}); ok {
			processDeploymentMap(depMap)
		}
	}

	// Check nested service plan blocks
	spBlocks := findAllOmnistrateServicePlanBlocks(yamlContent)
	for _, spMap := range spBlocks {
		extractAccountDetails(spMap)
		if deployment, exists := spMap["deployment"]; exists {
			if depMap, ok := deployment.(map[string]interface{}); ok {
				processDeploymentMap(depMap)
			}
		}
	}

	return awsAccountID, awsBootstrapRoleARN, gcpProjectID, gcpProjectNumber, gcpServiceAccountEmail, azureSubscriptionID, azureTenantID, extractDeploymentType
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

// processTemplateExpressions processes template expressions like {{ $file:path }} recursively
func processTemplateExpressions(data []byte, baseDir string) ([]byte, error) {
	content := string(data)

	// Pattern to match {{ $file:path }}
	re := regexp.MustCompile(`(?m)^(?P<indent>[ \t]*)?(?P<key>[\S\t ]*)?{{\s*\$file:(?P<filepath>[^\s}]+)\s*}}`)

	for re.MatchString(content) {
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



	// Validate deployment type
	if deploymentType != build.DeploymentTypeHosted && deploymentType != build.DeploymentTypeByoa {
		fmt.Printf("Warning: Invalid deployment type '%s'. Using default 'hosted'. Valid values are: hosted, byoa\n", deploymentType)
		deploymentType = "hosted"
	}

	yamlDoc := make(map[string]interface{}) // Initialize yamlDoc as an empty map

	if awsAccountID != "" && awsBootstrapRoleARN == "" {
		// Default role ARN if not provided
		awsBootstrapRoleARN = fmt.Sprintf("arn:aws:iam::%s:role/omnistrate-bootstrap-role", awsAccountID)
	}

	if gcpServiceAccountEmail == "" && gcpProjectID != "" {
		// Default service account email if not provided
		gcpServiceAccountEmail = fmt.Sprintf("omnistrate-bootstrap@%s.iam.gserviceaccount.com", gcpProjectID)
	}

	getServicePlan := func() map[string]interface{} {
		if sp, ok := yamlDoc["x-omnistrate-service-plan"].(map[string]interface{}); ok {
			return sp
		}
		return make(map[string]interface{})
	}

	// Build the deployment section based on deploymentType and specType

	switch deploymentType {
	case build.DeploymentTypeByoa:
		if specType == build.ServicePlanSpecType {
			yamlDoc["deployment"] = map[string]interface{}{
				"byoaDeployment": map[string]interface{}{
					"awsAccountId":               awsAccountID,
					"awsBootstrapRoleAccountArn": awsBootstrapRoleARN,
				},
			}
		} else {
			sp := getServicePlan()
			sp["deployment"] = map[string]interface{}{
				"byoaDeployment": map[string]interface{}{
					"awsAccountId":               awsAccountID,
					"awsBootstrapRoleAccountArn": awsBootstrapRoleARN,
				},
			}
			yamlDoc["x-omnistrate-service-plan"] = sp
		}

	case build.DeploymentTypeHosted:
		if specType == build.ServicePlanSpecType {
			yamlDoc["deployment"] = map[string]interface{}{
				"hostedDeployment": map[string]interface{}{},
			}
			hosted := make(map[string]interface{})
		if awsAccountID != "" {
				hosted["awsAccountId"] = awsAccountID
			if awsBootstrapRoleARN != "" {
					hosted["awsBootstrapRoleAccountArn"] = awsBootstrapRoleARN
			}
		}
		if gcpProjectID != "" {
				hosted["gcpProjectId"] = gcpProjectID
			if gcpProjectNumber != "" {
					hosted["gcpProjectNumber"] = gcpProjectNumber
			}
			if gcpServiceAccountEmail != "" {
					hosted["gcpServiceAccountEmail"] = gcpServiceAccountEmail
			}
		}
		if azureSubscriptionID != "" {
				hosted["azureSubscriptionId"] = azureSubscriptionID
			if azureTenantID != "" {
					hosted["azureTenantId"] = azureTenantID
			}
		}
			yamlDoc["deployment"].(map[string]interface{})["hostedDeployment"] = hosted
			} else {
				sp := getServicePlan()
		hosted := make(map[string]interface{})
		if awsAccountID != "" {
			hosted["awsAccountId"] = awsAccountID
			if awsBootstrapRoleARN != "" {
				hosted["awsBootstrapRoleAccountArn"] = awsBootstrapRoleARN
			}
		}
		if gcpProjectID != "" {
			hosted["gcpProjectId"] = gcpProjectID
			if gcpProjectNumber != "" {
				hosted["gcpProjectNumber"] = gcpProjectNumber
			}
			if gcpServiceAccountEmail != "" {
				hosted["gcpServiceAccountEmail"] = gcpServiceAccountEmail
			}
		}
		if azureSubscriptionID != "" {
			hosted["azureSubscriptionId"] = azureSubscriptionID
			if azureTenantID != "" {
				hosted["azureTenantId"] = azureTenantID
			}
		}
				sp["deployment"] = map[string]interface{}{
					"hostedDeployment": hosted,
				}
				yamlDoc["x-omnistrate-service-plan"] = sp
			}
		}
	return yamlDoc
}

// containsString checks if a slice of strings contains a specific string.
func containsString(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// CloudInstanceStatus holds cloud account instances grouped by status
type CloudInstanceStatus struct {
	Ready    []string
	NotReady []string
	Provider string
}

func createCloudAccountInstances(ctx context.Context, token, serviceID, environmentID, planID, cloudProvider string, sm ysmrr.SpinnerManager) (string, string, error) {

	spinnerMsg := "Step 2/2: Checking for existing cloud account instances"
	spinner := sm.AddSpinner(spinnerMsg)
	targetCloudProvider := cloudProvider

	// Get existing cloud account instances grouped by cloud provider and status
	cloudInstancesByProvider, err := listCloudAccountInstancesByProvider(ctx, token, serviceID, environmentID, planID)
	if err != nil {
		spinner.UpdateMessage(spinnerMsg + ": Failed (" + err.Error() + ")")
		spinner.Error()
		sm.Stop()
		return "", targetCloudProvider, fmt.Errorf("failed to list cloud account instances: %w", err)
	}

	// Check for READY instances by cloud provider
	readyInstances := make(map[string][]string)
	for provider, instances := range cloudInstancesByProvider {
		if len(instances.Ready) > 0 {
			readyInstances[provider] = instances.Ready
		}
	}

	spinner.Complete()

	// If we have READY instances for any cloud provider, show them and let user choose
	if len(readyInstances) > 0 {
		sm.Stop()
		fmt.Println("Available READY cloud account instances:")

		// Create a list of all available instances with their providers
		var instanceOptions []struct {
			provider   string
			instanceID string
		}

		for provider, instances := range readyInstances {
			for _, instanceID := range instances {
				instanceOptions = append(instanceOptions, struct {
					provider   string
					instanceID string
				}{provider, instanceID})
			}
		}

		// Display options
		for i, option := range instanceOptions {
			fmt.Printf("  %d. %s cloud account instance: %s\n", i+1, strings.ToUpper(option.provider), option.instanceID)
		}
		fmt.Println("  0. Create a new cloud account instance")

		var choice int
		for {
			fmt.Printf("Select cloud account instance (0-%d): ", len(instanceOptions))
			if _, err := fmt.Scanln(&choice); err == nil && choice >= 0 && choice <= len(instanceOptions) {
				break
			}
			fmt.Println("Invalid selection. Please enter a valid number.")
		}

		if choice == 0 {
			// User chose to create a new instance - continue with creation logic below
			sm.Start()
		} else {
			// User selected an existing instance
			selected := instanceOptions[choice-1]
			fmt.Printf("Using existing READY %s cloud account instance: %s\n", strings.ToUpper(selected.provider), selected.instanceID)
			return selected.instanceID, selected.provider, nil
		}
	}

	// No READY instances found, create a new one
	sm.Stop()
	fmt.Println("No READY cloud account instances found. Creating a new one.")

	// Determine which cloud provider to use and get credentials
	if targetCloudProvider == "" {
		targetCloudProvider = promptForCloudProvider()
	}

	// Get cloud-specific credentials
	params, err := promptForCloudCredentials(targetCloudProvider)
	if err != nil {
		return "", targetCloudProvider, fmt.Errorf("failed to get cloud credentials: %w", err)
	}

	// Format parameters
	formattedParams, err := common.FormatParams(params, "")
	if err != nil {
		return "", targetCloudProvider, err
	}

	// Restart spinner for instance creation
	sm.Start()
	spinner = sm.AddSpinner("Step 2/2: Creating new cloud account instance")
	sm.Stop()
	// Determine which cloud provider to use and get credentials
	if targetCloudProvider == "" {
		targetCloudProvider = promptForCloudProvider()
	}

	sm.Start()

	createdInstanceID, err := createInstanceUnified(ctx, token, serviceID, environmentID, planID, targetCloudProvider, "", "", "cloudAccount", formattedParams, sm)
	if err != nil {
		spinner.UpdateMessage("Step 2/2: Creating cloud account instance: Failed (" + err.Error() + ")")
		spinner.Error()
		return "", targetCloudProvider, err
	}

	spinner.UpdateMessage(fmt.Sprintf("Step 2/2: Creating cloud account instance: Success (ID: %s)", createdInstanceID))
	spinner.Complete()

	// Stop spinner to show instructions
	sm.Stop()

	// Start polling for account verification
	fmt.Println("\n🔄 Checking for account verification...")
	accountID, err := waitForAccountVerification(ctx, token, serviceID, environmentID, planID, createdInstanceID, targetCloudProvider)
	if err != nil {
		fmt.Printf("❌ Account verification failed: %v\n", err)
		return createdInstanceID, targetCloudProvider, err
	}

	fmt.Printf("✅ Account verified successfully (ID: %s)\n", accountID)
	return createdInstanceID, targetCloudProvider, nil
}

// listCloudAccountInstancesByProvider lists cloud account instances grouped by cloud provider and status
func listCloudAccountInstancesByProvider(ctx context.Context, token, serviceID, environmentID, planID string) (map[string]CloudInstanceStatus, error) {
	_, existingInstances, err := listInstances(ctx, token, serviceID, environmentID, planID, "", "onlyCloudAccounts")
	if err != nil {
		return nil, err
	}

	cloudProviderMap := make(map[string]CloudInstanceStatus)

	for _, instance := range existingInstances {
		cloudProvider := instance.cloudProvider
		status := instance.status
		instanceID := instance.instanceID

		if cloudProvider == "" {
			cloudProvider = "unknown"
		}

		if _, exists := cloudProviderMap[cloudProvider]; !exists {
			cloudProviderMap[cloudProvider] = CloudInstanceStatus{
				Provider: cloudProvider,
				Ready:    []string{},
				NotReady: []string{},
			}
		}

		instanceStatus := cloudProviderMap[cloudProvider]
		if status == "READY" {
			instanceStatus.Ready = append(instanceStatus.Ready, instanceID)
		} else {
			instanceStatus.NotReady = append(instanceStatus.NotReady, instanceID)
		}
		cloudProviderMap[cloudProvider] = instanceStatus
	}

	return cloudProviderMap, nil
}

// promptForCloudProvider prompts user to select a cloud provider
func promptForCloudProvider() string {
	fmt.Println("Available cloud providers:")
	fmt.Println("  1. AWS")
	fmt.Println("  2. GCP")
	fmt.Println("  3. Azure")

	var choice int
	for {
		fmt.Print("Select cloud provider (1-3): ")
		_, err := fmt.Scanln(&choice)
		if err == nil && choice >= 1 && choice <= 3 {
			break
		}
		fmt.Println("Invalid selection. Please enter 1, 2, or 3.")
	}

	switch choice {
	case 1:
		return "aws"
	case 2:
		return "gcp"
	case 3:
		return "azure"
	default:
		return "aws" // fallback
	}
}

// promptForCloudCredentials prompts user for cloud-specific credentials
func promptForCloudCredentials(cloudProvider string) (string, error) {
	var params map[string]interface{}

	switch cloudProvider {
	case "aws":
		fmt.Println("Enter AWS credentials:")
		var awsAccountID, awsBootstrapRoleArn string

		fmt.Print("AWS Account ID: ")
		if _, err := fmt.Scanln(&awsAccountID); err != nil {
			return "", fmt.Errorf("failed to read AWS Account ID: %w", err)
		}

		fmt.Print("AWS Bootstrap Role ARN (optional, press enter for default): ")
		if _, err := fmt.Scanln(&awsBootstrapRoleArn); err != nil {
			// Ignore error for optional field
			awsBootstrapRoleArn = ""
		}

		if awsBootstrapRoleArn == "" {
			awsBootstrapRoleArn = fmt.Sprintf("arn:aws:iam::%s:role/omnistrate-bootstrap-role", awsAccountID)
		}

		params = map[string]interface{}{
			"account_configuration_method": "CloudFormation",
			"aws_account_id":               awsAccountID,
			"aws_bootstrap_role_arn":       awsBootstrapRoleArn,
			"cloud_provider":               "aws",
		}

	case "gcp":
		fmt.Println("Enter GCP credentials:")
		var gcpProjectID, gcpProjectNumber string

		fmt.Print("GCP Project ID: ")
		if _, err := fmt.Scanln(&gcpProjectID); err != nil {
			return "", fmt.Errorf("failed to read GCP Project ID: %w", err)
		}

		fmt.Print("GCP Project Number: ")
		if _, err := fmt.Scanln(&gcpProjectNumber); err != nil {
			return "", fmt.Errorf("failed to read GCP Project Number: %w", err)
		}

		params = map[string]interface{}{
			"account_configuration_method": "GCPScript",
			"gcp_project_id":               gcpProjectID,
			"gcp_project_number":           gcpProjectNumber,
			"cloud_provider":               "gcp",
		}

	case "azure":
		fmt.Println("Enter Azure credentials:")
		var azureSubscriptionID, azureTenantID string

		fmt.Print("Azure Subscription ID: ")
		if _, err := fmt.Scanln(&azureSubscriptionID); err != nil {
			return "", fmt.Errorf("failed to read Azure Subscription ID: %w", err)
		}

		fmt.Print("Azure Tenant ID: ")
		if _, err := fmt.Scanln(&azureTenantID); err != nil {
			return "", fmt.Errorf("failed to read Azure Tenant ID: %w", err)
		}

		params = map[string]interface{}{
			"account_configuration_method": "AzureScript",
			"azure_subscription_id":        azureSubscriptionID,
			"azure_tenant_id":              azureTenantID,
			"cloud_provider":               "azure",
		}

	default:
		return "", fmt.Errorf("unsupported cloud provider: %s", cloudProvider)
	}

	// Convert to JSON string
	jsonBytes, err := json.Marshal(params)
	if err != nil {
		return "", fmt.Errorf("failed to marshal parameters: %w", err)
	}

	return string(jsonBytes), nil
}

// showCloudSetupInstructions displays cloud-specific setup instructions
func showCloudSetupInstructions(cloudProvider, instanceID string) {
	fmt.Printf("\n📋 Cloud Account Setup Instructions for %s:\n", strings.ToUpper(cloudProvider))
	fmt.Printf("Instance ID: %s\n", instanceID)

	switch cloudProvider {
	case "aws":
		fmt.Printf(dataaccess.NextStepVerifyAccountMsgTemplateAWS,
			"https://console.aws.amazon.com/cloudformation/",
			dataaccess.AwsCloudFormationGuideURL,
			dataaccess.AwsGcpTerraformScriptsURL,
			instanceID,
			dataaccess.AwsGcpTerraformGuideURL)
	case "gcp":
		fmt.Printf(dataaccess.NextStepVerifyAccountMsgTemplateGCP,
			"# Follow the setup commands for your GCP project")
	case "azure":
		fmt.Printf(dataaccess.NextStepVerifyAccountMsgTemplateAzure,
			"# Follow the setup commands for your Azure subscription")
	}

	fmt.Println("\n⏳ Please complete the setup steps above and wait for verification...")
}

// waitForAccountVerification polls for account status changes from NOT_READY to READY
func waitForAccountVerification(ctx context.Context, token, serviceID,
	environmentID, planID, instanceID, targetCloudProvider string) (string, error) {
	maxRetries := 60 // 10 minutes with 10-second intervals
	retryInterval := 10 * time.Second
	showCloudSetupInstruction := false

	for i := 0; i < maxRetries; i++ {
		// Get all accounts for the cloud provider
		_, existingInstances, err := listInstances(ctx, token, serviceID, environmentID, planID, "", "onlyCloudAccounts")
		if err != nil {
			fmt.Printf("⚠️  Failed to check account status: %v\n", err)
			time.Sleep(retryInterval)
			continue
		}

		for _, instance := range existingInstances {
			if instance.instanceID == instanceID && instance.cloudProvider == targetCloudProvider {
				switch instance.status {
				case "READY":
					return instance.instanceID, nil
				case "FAILED":
					return "", fmt.Errorf("account setup encountered an error %s", instance.status)
				}
			}
		}

		// Still not ready, continue polling

		if i%3 == 0 && i > 2 { // Show progress every 30 seconds
			fmt.Printf("⏳ Still waiting for account verification... (%d/%d)\n", i+1, maxRetries)
			// Show cloud-specific setup instructions
			if !showCloudSetupInstruction {
				showCloudSetupInstructions(targetCloudProvider, instanceID)
				showCloudSetupInstruction = true
			}
		}

		time.Sleep(retryInterval)
	}

	return "", fmt.Errorf("account verification timed out after %d attempts", maxRetries)
}

// waitForAccountReady polls for account status to become READY, up to 10 minutes
func waitForAccountReady(ctx context.Context, token, accountID string) error {
	timeout := time.After(10 * time.Minute)
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-timeout:
			return pkgerrors.New("timed out waiting for account to become READY")
		case <-ticker.C:
			account, err := dataaccess.DescribeAccount(ctx, token, accountID)
			if err != nil {
				return err
			}
			if account.Status == "READY" {
				return nil
			}
		}
	}
}

// --- helpers for nicer errors/messages ---

func printAuthError() {
	utils.PrintError(fmt.Errorf(
		"❌ Authentication error\n\n" +
			"  You are not logged in or your session has expired.\n\n" +
			"Next steps:\n" +
			"  1. Run:  omctl login\n" +
			"  2. Re-run your previous omctl deploy command",
	))
}

func printBackendError(context string, err error) {
	contextTitle := cases.Title(language.English).String(context)
	utils.PrintError(fmt.Errorf(
		"❌ %s failed\n\n  %v\n\n"+
			"Next steps:\n"+
			"  - Retry the command in a few minutes\n"+
			"  - If the problem persists, contact Omnistrate support and share this error",
		contextTitle, err,
	))
}

func wrapAndPrintServiceBuildError(err error) {
	msg := err.Error()
	if strings.Contains(msg, "public service environment already exists") {
		utils.PrintError(fmt.Errorf(
			"❌ Environment conflict during service creation\n\n" +
				"  The service already has a public environment in this account and a new conflicting\n" +
				"  environment cannot be created automatically\n\n" +
				"Next steps:\n" +
				"  - To update the existing service and environment, re-run with the same service name\n" +
				"  - To create a new service, use a different name with --product-name",
		))
		return
	}
	utils.PrintError(fmt.Errorf(
		"❌ Service creation failed\n\n  %v\n\n"+
			"Step 1/2 (service creation) failed. No instance was created",
		err,
	))
}

func isMissingParamsError(err error) bool {
	return strings.Contains(err.Error(), "missing required parameters for instance creation")
}

func printMissingParamsGuidance(err error) {
	utils.PrintError(fmt.Errorf(
		"❌ Missing required parameters for instance creation\n\n"+
			"  %s\n\n"+
			"Next steps:\n"+
			"  - Provide values using --param, for example:\n"+
			"      omctl deploy --param '{\"key\":\"value\",...}'\n"+
			"  - Or provide a JSON file with --param-file",
		err.Error(),
	))
}
