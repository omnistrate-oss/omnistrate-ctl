package build

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/omnistrate-oss/omnistrate-ctl/cmd/common"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/model"

	"github.com/chelnak/ysmrr"
	"github.com/compose-spec/compose-go/loader"
	"github.com/compose-spec/compose-go/types"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/config"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/dataaccess"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/utils"
	openapiclient "github.com/omnistrate-oss/omnistrate-sdk-go/v1"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

var (
	ServiceID     string
	EnvironmentID string
	ProductTierID string
)

const (
	buildExample = `
# Build service in dev environment
omctl build --product-name "My Service"

# Build service with compose spec in dev environment
omctl build --file compose.yaml --product-name "My Service"

# Build service with compose spec in prod environment
omctl build --file compose.yaml --product-name "My Service" --environment prod --environment-type prod

# Build service with compose spec and release the service with a release description
omctl build --file compose.yaml --product-name "My Service" --release --release-description "v1.0.0-alpha"

# Build service with compose spec and release the service as preferred with a release description
omctl build --file compose.yaml --product-name "My Service" --release-as-preferred --release-description "v1.0.0-alpha"

# Build service with compose spec interactively
omctl build --file compose.yaml --product-name "My Service" --interactive

# Build service with compose spec with service description and service logo
omctl build --file compose.yaml --product-name "My Service" --description "My Service Description" --service-logo-url "https://example.com/logo.png"

# Build service with service specification for Helm, Operator or Kustomize in dev environment
omctl build --spec-type ServicePlanSpec --file spec.yaml --product-name "My Service"

# Build service with service specification for Helm, Operator or Kustomize in prod environment
omctl build --spec-type ServicePlanSpec --file spec.yaml --product-name "My Service" --environment prod --environment-type prod

# Build service with service specification for Helm, Operator or Kustomize as preferred
omctl build --spec-type ServicePlanSpec --file spec.yaml --product-name "My Service" --release-as-preferred --release-description "v1.0.0-alpha"

# Build service from image in dev environment
omctl build --image docker.io/mysql:5.7 --product-name MySQL --env-var "MYSQL_ROOT_PASSWORD=password" --env-var "MYSQL_DATABASE=mydb"

# Build service with private image in dev environment
omctl build --image docker.io/namespace/my-image:v1.2 --product-name "My Service" --image-registry-auth-username username --image-registry-auth-password password --env-var KEY1:VALUE1 --env-var KEY2:VALUE2
`

	buildLong = `Build command can be used to build a service from image, docker compose, and service plan spec. 
It has two main modes of operation:
  - Create a new service plan
  - Update an existing service plan

Below info served as service plan identifiers:
  - service name (--product-name, required)
  - environment name (--environment, optional, default: Dev)
  - environment type (--environment-type, optional, default: dev)
  - service plan name (the name field of x-omnistrate-service-plan tag in compose spec file, required)
If the identifiers match an existing service plan, it will update that plan. Otherwise, it'll create a new service plan. 

This command has an interactive mode. In this mode, you can choose to promote the service plan to production by interacting with the prompts.`
)

// BuildCmd represents the build command
var BuildCmd = &cobra.Command{
	Use:          "build [--file=file] [--spec-type=spec-type] [--product-name=service-name] [--description=service-description] [--service-logo-url=service-logo-url] [--environment=environment-name] [--environment-type=environment-type] [--release] [--release-as-preferred] [--release-description=release-description][--interactive] [--image=image-url] [--image-registry-auth-username=username] [--image-registry-auth-password=password] [--env-var=\"key=var\"]",
	Short:        "Build Services from image, compose spec or service plan spec",
	Long:         buildLong,
	Example:      buildExample,
	RunE:         runBuild,
	SilenceUsage: true,
}

func init() {
	BuildCmd.Flags().StringP("file", "f", "", "Path to the docker compose file (defaults to compose.yaml or spec.yaml)")
	BuildCmd.Flags().StringP("name", "n", "", "Name of the service. A service can have multiple service plans. The build command will build a new or existing service plan inside the specified service.")
	BuildCmd.Flags().StringP("product-name", "", "", "Name of the service. A service can have multiple service plans. The build command will build a new or existing service plan inside the specified service.")
	BuildCmd.Flags().StringP("description", "", "", "A short description for the whole service. A service can have multiple service plans.")
	BuildCmd.Flags().StringP("service-logo-url", "", "", "URL to the service logo")
	BuildCmd.Flags().StringP("environment", "", "Dev", "Name of the environment to build the service in")
	BuildCmd.Flags().StringP("environment-type", "", "dev", "Type of environment. Valid options include: 'dev', 'prod', 'qa', 'canary', 'staging', 'private')")
	BuildCmd.Flags().BoolP("release", "", false, "Release the service after building it")
	BuildCmd.Flags().BoolP("release-as-preferred", "", false, "Release the service as preferred after building it")
	BuildCmd.Flags().StringP("release-name", "", "", "Custom description of the release version. Deprecated: use --release-description instead")
	BuildCmd.Flags().StringP("release-description", "", "", "Used together with --release or --release-as-preferred flag. Provide a description for the release version")
	BuildCmd.Flags().BoolP("interactive", "i", false, "Interactive mode")
	BuildCmd.Flags().StringP("spec-type", "s", "", "Spec type (will infer from file if not provided). Valid options include: 'DockerCompose', 'ServicePlanSpec'")
	BuildCmd.Flags().BoolP("dry-run", "d", false, "Simulate building the service without actually creating resources")

	BuildCmd.Flags().StringP("image", "", "", "Provide the complete image repository URL with the image name and tag (e.g., docker.io/namespace/my-image:v1.2)")
	BuildCmd.Flags().StringArrayP("env-var", "", nil, "Used together with --image flag. Provide environment variables in the format --env-var key1=var1 --env-var key2=var2")
	BuildCmd.Flags().StringP("image-registry-auth-username", "", "", "Used together with --image flag. Provide the username to authenticate with the image registry if it's a private registry")
	BuildCmd.Flags().StringP("image-registry-auth-password", "", "", "Used together with --image flag. Provide the password to authenticate with the image registry if it's a private registry")

	BuildCmd.MarkFlagsRequiredTogether("image-registry-auth-username", "image-registry-auth-password")
	// Mark one of them as required
	BuildCmd.MarkFlagsOneRequired("name", "product-name")
	// Deprecate the old --name flag
	if err := BuildCmd.Flags().MarkDeprecated("name", "use --product-name instead"); err != nil {
		utils.PrintError(err)
		return
	}
	// Hide the deprecated flag from help
	if err := BuildCmd.Flags().MarkHidden("name"); err != nil {
		utils.PrintError(err)
		return
	}

	err := BuildCmd.MarkFlagFilename("file")
	if err != nil {
		return
	}
	err = BuildCmd.Flags().MarkHidden("release-name")
	if err != nil {
		return
	}
	BuildCmd.MarkFlagsRequiredTogether("environment", "environment-type")
}

func runBuild(cmd *cobra.Command, args []string) error {
	defer config.CleanupArgsAndFlags(cmd, &args)

	// Retrieve flags
	file, err := cmd.Flags().GetString("file")
	if err != nil {
		return err
	}

	specType, err := cmd.Flags().GetString("spec-type")
	if err != nil {
		return err
	}
	// Check if spec-type was provided and default if not
	if specType == "" {
		if file == PlanSpecFileName {
			specType = ServicePlanSpecType
		} else {
			specType = DockerComposeSpecType
		}
	}

	name, err := cmd.Flags().GetString("name")
	if err != nil {
		return err
	}
	productName, err := cmd.Flags().GetString("product-name")
	if err != nil {
		return err
	}

	if name != "" && productName != "" {
		err = errors.New("only one of name or product-name can be provided")
		utils.PrintError(err)
		return err
	}

	// Use product-name if provided, otherwise use name
	// Since flags are mutually exclusive, only one will be set
	if productName != "" {
		name = productName
	}
	description, err := cmd.Flags().GetString("description")
	if err != nil {
		return err
	}
	serviceLogoURL, err := cmd.Flags().GetString("service-logo-url")
	if err != nil {
		return err
	}
	environment, err := cmd.Flags().GetString("environment")
	if err != nil {
		return err
	}
	environmentType, err := cmd.Flags().GetString("environment-type")
	if err != nil {
		return err
	}
	release, err := cmd.Flags().GetBool("release")
	if err != nil {
		return err
	}
	releaseAsPreferred, err := cmd.Flags().GetBool("release-as-preferred")
	if err != nil {
		return err
	}
	releaseName, err := cmd.Flags().GetString("release-name")
	if err != nil {
		return err
	}
	releaseDescription, err := cmd.Flags().GetString("release-description")
	if err != nil {
		return err
	}
	interactive, err := cmd.Flags().GetBool("interactive")
	if err != nil {
		return err
	}
	imageUrl, err := cmd.Flags().GetString("image")
	if err != nil {
		return err
	}
	envVars, err := cmd.Flags().GetStringArray("env-var")
	if err != nil {
		return err
	}
	imageRegistryAuthUsername, err := cmd.Flags().GetString("image-registry-auth-username")
	if err != nil {
		return err
	}
	imageRegistryAuthPassword, err := cmd.Flags().GetString("image-registry-auth-password")
	if err != nil {
		return err
	}
	output, err := cmd.Flags().GetString("output")
	if err != nil {
		return err
	}
	dryRun, err := cmd.Flags().GetBool("dry-run")
	if err != nil {
		return err
	}

	// Validate input arguments
	if file == "" && imageUrl == "" {
		err := errors.New("either file or image is required")
		utils.PrintError(err)
		return err
	}

	if file != "" && imageUrl != "" {
		err := errors.New("only one of file or image can be provided")
		utils.PrintError(err)
		return err
	}

	if interactive && output == "json" {
		err := errors.New("interactive mode is not supported with json output")
		utils.PrintError(err)
		return err
	}

	// Load the compose file
	var fileData []byte
	if file != "" && imageUrl == "" {
		if _, err := os.Stat(file); os.IsNotExist(err) {
			if file != "" {
				err = fmt.Errorf("file %s does not exist", file)
				utils.PrintError(err)
				return err
			} else {
				// check for compose file
				file = ComposeFileName
				specType = DockerComposeSpecType
				if _, err := os.Stat(file); os.IsNotExist(err) {
					// If the file doesn't exist and wasn't explicitly provided, we check if there is a spec file
					file = PlanSpecFileName
					specType = ServicePlanSpecType
					if _, err := os.Stat(file); os.IsNotExist(err) {
						err = errors.New("no compose or spec file found, please provide a valid file using --file flag")
						utils.PrintError(err)
						return err
					}
				}
			}
		}

		var err error
		fileData, err = os.ReadFile(filepath.Clean(file))
		if err != nil {
			return err
		}
	}

	// Validate user is currently logged in
	token, err := common.GetTokenWithLogin()
	if err != nil {
		utils.PrintError(err)
		return err
	}

	// Step 0: Generate compose spec from image if image is provided
	if imageUrl != "" {
		// Split the image url into registry and image
		imageParts := strings.Split(imageUrl, "/")
		if len(imageParts) < 2 {
			err = errors.New("invalid image format")
			utils.PrintError(err)
			return err
		}

		// Get the image registry and image
		imageRegistry := imageParts[0]
		image := strings.Join(imageParts[1:], "/")

		// Check if the image is accessible
		var userNamePtr, passwordPtr *string
		if imageRegistryAuthUsername != "" && imageRegistryAuthPassword != "" {
			userNamePtr = utils.ToPtr(imageRegistryAuthUsername)
			passwordPtr = utils.ToPtr(imageRegistryAuthPassword)
		}

		checkImageRes, err := dataaccess.CheckIfContainerImageAccessible(cmd.Context(), token, imageRegistry, image, userNamePtr, passwordPtr)
		if err != nil {
			utils.PrintError(err)
			return err
		}

		// Error out if image is not accessible
		if !checkImageRes.ImageAccessible {
			err = errors.New("image not accessible")
			if checkImageRes.ErrorMsg != nil {
				err = errors.New(*checkImageRes.ErrorMsg)
			}
			utils.PrintError(err)
			return err
		}

		// Parse the environment variables
		var formattedEnvVars []openapiclient.EnvironmentVariable
		for _, envVar := range envVars {
			if envVar == "[]" {
				continue
			}
			envVarParts := strings.Split(envVar, "=")
			if len(envVarParts) != 2 {
				err = errors.New("invalid environment variable format")
				utils.PrintError(err)
				return err
			}
			formattedEnvVars = append(formattedEnvVars, openapiclient.EnvironmentVariable{
				Key:   envVarParts[0],
				Value: envVarParts[1],
			})
		}

		// Generate compose spec from image
		generateComposeSpecRequest := openapiclient.GenerateComposeSpecFromContainerImageRequest2{
			ImageRegistry:        imageRegistry,
			Image:                image,
			EnvironmentVariables: formattedEnvVars,
			Username:             userNamePtr,
			Password:             passwordPtr,
		}

		generateComposeSpecRes, err := dataaccess.GenerateComposeSpecFromContainerImage(cmd.Context(), token, generateComposeSpecRequest)
		if err != nil {
			utils.PrintError(err)
			return err
		}

		// Decode the base64 encoded file content
		fileData, err = base64.StdEncoding.DecodeString(generateComposeSpecRes.FileContent)
		if err != nil {
			utils.PrintError(err)
			return err
		}
	}

	// Step 1: Build service
	serviceLogoURLPtr := &serviceLogoURL
	if serviceLogoURL == "" {
		serviceLogoURLPtr = nil
	}

	descriptionPtr := &description
	if description == "" {
		descriptionPtr = nil
	}

	if !isValidSpecType(specType) {
		err = errors.New(fmt.Sprintf("invalid spec type %s, valid options are: %s", specType, strings.Join(validSpecType, ", ")))
		utils.PrintError(err)
		return err
	}

	environmentPtr := &environment
	if environment == "" {
		environmentPtr = nil
	}

	environmentTypePtr := utils.ToPtr(strings.ToUpper(environmentType))
	if environmentType == "" {
		environmentTypePtr = nil
	}

	var releaseNamePtr *string
	if releaseName != "" {
		releaseNamePtr = &releaseName
	}
	if releaseDescription != "" {
		releaseNamePtr = &releaseDescription
	}

	var sm1 ysmrr.SpinnerManager
	var spinner1 *ysmrr.Spinner
	if output != "json" {
		sm1 = ysmrr.NewSpinnerManager()
		spinner1 = sm1.AddSpinner("Building service...")
		sm1.Start()
	}

	var cwd string
	cwd, err = os.Getwd()
	if err != nil {
		utils.HandleSpinnerError(spinner1, sm1, err)
		return err
	}

	// Render files
	fileData, err = RenderFile(fileData, cwd, file, sm1, spinner1)
	if err != nil {
		utils.HandleSpinnerError(spinner1, sm1, err)
		return err
	}

	var undefinedResources map[string]string
	ServiceID, EnvironmentID, ProductTierID, undefinedResources, err = buildService(
		cmd.Context(),
		fileData,
		token,
		name,
		specType,
		descriptionPtr,
		serviceLogoURLPtr,
		environmentPtr,
		environmentTypePtr,
		release,
		releaseAsPreferred,
		releaseNamePtr,
		dryRun,
	)
	if err != nil {
		utils.HandleSpinnerError(spinner1, sm1, err)
		return err
	}
	header := "Successfully built service"
	if dryRun {
		header = "Simulated service build completed successfully (dry run)"
	}
	utils.HandleSpinnerSuccess(spinner1, sm1, header)

	// Get product tier name
	productTier, err := dataaccess.DescribeProductTier(cmd.Context(), token, ServiceID, ProductTierID)
	if err != nil {
		utils.PrintError(err)
		return err
	}

	// Print the service plan details
	servicePlanDetails := model.ServicePlanVersion{
		PlanID:      ProductTierID,
		PlanName:    productTier.Name,
		ServiceID:   ServiceID,
		ServiceName: name,
		Environment: environment,
	}

	if !dryRun && (release || releaseAsPreferred) {
		versionDetails, err := dataaccess.DescribeLatestVersion(cmd.Context(), token, ServiceID, ProductTierID)
		if err != nil {
			err = errors.Wrap(err, "failed to get the latest version")
			return err
		}
		servicePlanDetails.Version = versionDetails.Version
		if versionDetails.Name != nil {
			servicePlanDetails.ReleaseDescription = *versionDetails.Name
		}
		servicePlanDetails.VersionSetStatus = versionDetails.Status
	}

	if err = utils.PrintTextTableJsonOutput(output, servicePlanDetails); err != nil {
		return err
	}

	// Return early if output is json
	if output == "json" {
		return nil
	}

	// Print warning if there are any undefined resources
	if len(undefinedResources) > 0 {
		utils.PrintWarning("The following resources appear in the service plan but were not defined in the spec:")
		for resourceName, resourceID := range undefinedResources {
			utils.PrintWarning(fmt.Sprintf("  %s: %s", resourceName, resourceID))
		}
		utils.PrintWarning("These resources were not processed during the build. If you no longer need them, please deprecate and remove them from the service plan manually in UI or using the API.")
	}

	utils.PrintURL("Check the service plan result at", fmt.Sprintf("https://%s/product-tier?serviceId=%s&environmentId=%s", config.GetRootDomain(), ServiceID, EnvironmentID))

	// Ask user to verify account if there are any unverified accounts
	dataaccess.AskVerifyAccountIfAny(cmd.Context())

	serviceEnvironment, err := dataaccess.DescribeServiceEnvironment(cmd.Context(), token, ServiceID, EnvironmentID)
	if err != nil {
		utils.PrintError(err)
		return err
	}

	// Step 2: Display SaaS portal URL
	if checkIfSaaSPortalReady(serviceEnvironment) {
		utils.PrintURL("Access your SaaS offer at", getSaaSPortalURL(serviceEnvironment, ServiceID, EnvironmentID))
	} else if interactive {
		// Ask the user if they want to wait for the SaaS portal URL
		fmt.Print("Do you want to wait to access the SaaS portal? [Y/n] It may take a few minutes: ")
		var userInput string
		_, err = fmt.Scanln(&userInput)
		if err != nil {
			utils.PrintError(err)
			return err
		}

		userInput = strings.TrimSpace(strings.ToUpper(userInput))

		if strings.ToLower(userInput) == "y" {
			sm2 := ysmrr.NewSpinnerManager()
			loading := sm2.AddSpinner("Loading SaaS portal...")
			sm2.Start()

			for {
				serviceEnvironment, err = dataaccess.DescribeServiceEnvironment(cmd.Context(), token, ServiceID, EnvironmentID)
				if err != nil {
					utils.PrintError(err)
					return err
				}

				if checkIfSaaSPortalReady(serviceEnvironment) {
					loading.Complete()
					sm2.Stop()
					utils.PrintURL("Your SaaS offer is ready at", getSaaSPortalURL(serviceEnvironment, ServiceID, EnvironmentID))
					break
				}
			}
		}
	}

	// Step 3: Launch service to production if the service is in dev environment
	if interactive {
		if strings.ToLower(serviceEnvironment.Type) == "dev" {
			// Ask the user if they want to launch the service to production
			fmt.Print("Do you want to launch it to production? [Y/n] You can always promote it later: ")
			var userInput string
			_, err = fmt.Scanln(&userInput)
			if err != nil {
				utils.PrintError(err)
				return err
			}
			userInput = strings.TrimSpace(strings.ToUpper(userInput))

			if strings.ToLower(userInput) == "y" {
				sm2 := ysmrr.NewSpinnerManager()
				launching := sm2.AddSpinner("Launching service to production...")
				sm2.Start()

				prodEnvironment, err := dataaccess.FindEnvironment(cmd.Context(), token, ServiceID, "prod")
				if err != nil && !errors.As(err, &dataaccess.ErrEnvironmentNotFound) {
					utils.PrintError(err)
					return err
				}

				var prodEnvironmentID string
				if errors.As(err, &dataaccess.ErrEnvironmentNotFound) {
					// Get default deployment config ID
					defaultDeploymentConfigID, err := dataaccess.GetDefaultDeploymentConfigID(cmd.Context(), token)
					if err != nil {
						utils.PrintError(err)
						return err
					}
					prodEnvironmentID, err = dataaccess.CreateServiceEnvironment(
						cmd.Context(),
						token,
						"Production",
						"Production environment",
						ServiceID,
						"PUBLIC",
						"PROD",
						utils.ToPtr(EnvironmentID),
						defaultDeploymentConfigID,
						true,
						nil)
					if err != nil {
						utils.PrintError(err)
						return err
					}
				} else {
					prodEnvironmentID = prodEnvironment.Id
				}

				// Promote the service to production
				err = dataaccess.PromoteServiceEnvironment(cmd.Context(), token, ServiceID, EnvironmentID)
				if err != nil {
					utils.PrintError(err)
					return err
				}

				launching.Complete()
				sm2.Stop()

				// Retrieve the prod SaaS portal URL
				prodEnvironment, err = dataaccess.DescribeServiceEnvironment(cmd.Context(), token, ServiceID, prodEnvironmentID)
				if err != nil {
					utils.PrintError(err)
					return err
				}

				if checkIfSaaSPortalReady(prodEnvironment) {
					utils.PrintURL("Your SaaS portal is ready at", getSaaSPortalURL(prodEnvironment, ServiceID, prodEnvironmentID))
				} else if interactive {
					// Ask the user if they want to wait for the SaaS portal URL
					fmt.Print("Do you want to wait to access the prod SaaS offer? [Y/n] It may take a few minutes: ")
					_, err = fmt.Scanln(&userInput)
					if err != nil {
						utils.PrintError(err)
						return err
					}
					userInput = strings.TrimSpace(strings.ToUpper(userInput))

					if strings.ToLower(userInput) == "y" {
						sm3 := ysmrr.NewSpinnerManager()
						loading := sm3.AddSpinner("Preparing SaaS offer...")
						sm3.Start()

						for {
							serviceEnvironment, err = dataaccess.DescribeServiceEnvironment(cmd.Context(), token, ServiceID, prodEnvironmentID)
							if err != nil {
								utils.PrintError(err)
								return err
							}

							if checkIfSaaSPortalReady(serviceEnvironment) {
								loading.Complete()
								sm3.Stop()
								utils.PrintURL("Your SaaS offer is ready at", getSaaSPortalURL(serviceEnvironment, ServiceID, prodEnvironmentID))
								break
							}
						}
					}
				}
			}
		}
	}

	// Step 4: Next steps
	if interactive {
		dataaccess.PrintNextStepsAfterBuildMsg()
	}

	return nil
}

func buildService(ctx context.Context, fileData []byte, token, name, specType string, description, serviceLogoURL, environment, environmentType *string, release,
	releaseAsPreferred bool, releaseName *string, dryRun bool) (serviceID string, environmentID string, productTierID string, undefinedResources map[string]string, err error) {
	if name == "" {
		return "", "", "", make(map[string]string), errors.New("name is required")
	}

	if specType == "" {
		return "", "", "", make(map[string]string), errors.New("specType is required")
	}

	switch specType {
	case ServicePlanSpecType:
		request := openapiclient.BuildServiceFromServicePlanSpecRequest2{
			Name:               name,
			Description:        description,
			ServiceLogoURL:     serviceLogoURL,
			Environment:        environment,
			EnvironmentType:    environmentType,
			FileContent:        base64.StdEncoding.EncodeToString(fileData),
			Release:            utils.ToPtr(release),
			ReleaseAsPreferred: utils.ToPtr(releaseAsPreferred),
			ReleaseVersionName: releaseName,
			Dryrun:             utils.ToPtr(dryRun),
		}

		buildRes, err := dataaccess.BuildServiceFromServicePlanSpec(ctx, token, request)
		if err != nil {
			return "", "", "", make(map[string]string), err
		}
		if buildRes == nil {
			return "", "", "", make(map[string]string), errors.New("empty response from server")
		}
		return buildRes.GetServiceID(), buildRes.GetServiceEnvironmentID(), buildRes.GetProductTierID(), buildRes.GetUndefinedResources(), nil

	case DockerComposeSpecType:
		// Load the YAML content
		var parsedYaml map[string]interface{}
		parsedYaml, err = loader.ParseYAML(fileData)
		if err != nil {
			err = errors.Wrap(err, "failed to parse YAML content")
			return
		}

		// Decode spec YAML into a compose project
		var project *types.Project
		if project, err = loader.LoadWithContext(ctx, types.ConfigDetails{
			ConfigFiles: []types.ConfigFile{
				{
					Config: parsedYaml,
				},
			},
		}); err != nil {
			err = errors.Wrap(err, "invalid compose")
			return
		}

		// Convert config volumes to configs
		var modified bool
		if project, modified, err = convertVolumesToConfigs(project); err != nil {
			return "", "", "", make(map[string]string), err
		}

		// Convert the project back to YAML, in case it was modified
		if modified {
			var parsedYamlContent []byte
			if parsedYamlContent, err = project.MarshalYAML(); err != nil {
				err = errors.Wrap(err, "failed to marshal project to YAML")
				return
			}
			fileData = parsedYamlContent
		}

		// Get the configs from the project
		var configs *map[string]string
		if project.Configs != nil {
			configsTemp := make(map[string]string)
			for configName, config := range project.Configs {
				var configFileContent []byte
				configFileContent, err = os.ReadFile(filepath.Clean(config.File))
				if err != nil {
					return "", "", "", make(map[string]string), err
				}

				configsTemp[configName] = base64.StdEncoding.EncodeToString(configFileContent)
			}
			configs = &configsTemp
		}

		// Get the secrets from the project
		var secrets *map[string]string
		if project.Secrets != nil {
			secretsTemp := make(map[string]string)
			for secretName, secret := range project.Secrets {
				var fileContent []byte
				fileContent, err = os.ReadFile(filepath.Clean(secret.File))
				if err != nil {
					return "", "", "", make(map[string]string), err
				}
				secretsTemp[secretName] = base64.StdEncoding.EncodeToString(fileContent)
			}
			secrets = &secretsTemp
		}

		request := openapiclient.BuildServiceFromComposeSpecRequest2{
			Name:               name,
			Description:        description,
			ServiceLogoURL:     serviceLogoURL,
			Environment:        environment,
			EnvironmentType:    environmentType,
			FileContent:        base64.StdEncoding.EncodeToString(fileData),
			Release:            utils.ToPtr(release),
			ReleaseAsPreferred: utils.ToPtr(releaseAsPreferred),
			ReleaseVersionName: releaseName,
			Configs:            configs,
			Secrets:            secrets,
			Dryrun:             utils.ToPtr(dryRun),
		}

		buildRes, err := dataaccess.BuildServiceFromComposeSpec(ctx, token, request)
		if err != nil {
			return "", "", "", make(map[string]string), err
		}
		if buildRes == nil {
			return "", "", "", make(map[string]string), errors.New("empty response from server")
		}
		return buildRes.GetServiceID(), buildRes.GetServiceEnvironmentID(), buildRes.GetProductTierID(), buildRes.GetUndefinedResources(), nil

	default:
		return "", "", "", make(map[string]string), errors.New("invalid spec type")
	}
}

func checkIfSaaSPortalReady(serviceEnvironment *openapiclient.DescribeServiceEnvironmentResult) bool {
	if serviceEnvironment.SaasPortalUrl != nil && serviceEnvironment.SaasPortalStatus != nil && *serviceEnvironment.SaasPortalStatus == "RUNNING" {
		return true
	}

	return false
}

func getSaaSPortalURL(serviceEnvironment *openapiclient.DescribeServiceEnvironmentResult, serviceID, environmentID string) string {
	if serviceEnvironment.SaasPortalUrl != nil {
		return fmt.Sprintf("https://"+*serviceEnvironment.SaasPortalUrl+"/service-plans?serviceId=%s&environmentId=%s", serviceID, environmentID)
	}

	return ""
}

func isValidSpecType(s string) bool {
	for _, v := range validSpecType {
		if v == s {
			return true
		}
	}
	return false
}

// Most compose files mount the configs directly as volumes. This function converts the volumes to configs.
func convertVolumesToConfigs(project *types.Project) (converted *types.Project, modified bool, err error) {
	modified = false
	volumesToBeRemoved := make(map[int]map[int]struct{}) // map of service index to list of volume indexes to be removed
	for svcIdx, service := range project.Services {
		for volIdx, volume := range service.Volumes {
			// Check if the volume source exists. If so, it needs to be a directory with files or the source is itself a file
			if volume.Source != "" {
				source := filepath.Clean(volume.Source)
				if _, err = os.Stat(source); os.IsNotExist(err) {
					err = nil
					continue
				}

				// Check if the source is a directory
				var fileInfo os.FileInfo
				fileInfo, err = os.Stat(source)
				if err != nil {
					err = errors.Wrapf(err, "failed to get file info for %s", source)
					return
				}

				if fileInfo.IsDir() {
					// Check if the directory has files
					var files []string
					files, err = listFiles(source)
					if err != nil {
						err = errors.Wrapf(err, "failed to list files in %s", source)
						return
					}

					if len(files) == 0 {
						continue
					}

					// Create a config for each file
					for _, fileInDir := range files {
						sourceFileNameSHA := utils.HashSha256(fileInDir)
						config := types.ConfigObjConfig{
							Name: sourceFileNameSHA,
							File: fileInDir,
						}
						project.Configs[sourceFileNameSHA] = config

						// Also append to the configs list for this service
						var absolutePathToDir string
						absolutePathToDir, err = filepath.Abs(source)
						if err != nil {
							err = errors.Wrapf(err, "failed to get absolute path for %s", source)
							return
						}
						var relativePathInTarget string
						relativePathInTarget, err = filepath.Rel(absolutePathToDir, fileInDir)
						if err != nil {
							err = errors.Wrapf(err, "failed to get relative path for %s", fileInDir)
							return
						}
						service.Configs = append(service.Configs, types.ServiceConfigObjConfig{
							Source: sourceFileNameSHA,
							Target: filepath.Join(volume.Target, relativePathInTarget),
						})
					}
				} else {
					sourceFileNameSHA := utils.HashSha256(source)
					config := types.ConfigObjConfig{
						Name: sourceFileNameSHA,
						File: source,
					}
					project.Configs[sourceFileNameSHA] = config

					// Also append to the configs list for this service
					service.Configs = append(service.Configs, types.ServiceConfigObjConfig{
						Source: sourceFileNameSHA,
						Target: volume.Target,
					})
				}

				// Remove the volume from the service
				if volumesToBeRemoved[svcIdx] == nil {
					volumesToBeRemoved[svcIdx] = make(map[int]struct{})
				}
				volumesToBeRemoved[svcIdx][volIdx] = struct{}{}
			}
		}

		// Update the service in the project
		project.Services[svcIdx] = service
	}

	// Remove the volumes from the services
	for svcIdx, volumes := range volumesToBeRemoved {
		volumesBefore := make([]types.ServiceVolumeConfig, len(project.Services[svcIdx].Volumes))
		copy(volumesBefore, project.Services[svcIdx].Volumes)

		project.Services[svcIdx].Volumes = nil
		for volIdx := range volumesBefore {
			if _, ok := volumes[volIdx]; !ok {
				project.Services[svcIdx].Volumes = append(project.Services[svcIdx].Volumes, volumesBefore[volIdx])
			}
		}
	}

	converted = project
	modified = len(volumesToBeRemoved) > 0
	return
}

func listFiles(dir string) (files []string, err error) {
	err = filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		// Skip the directory itself
		if path == dir {
			return nil
		}

		if !info.IsDir() {
			files = append(files, path)
		}

		return nil
	})

	return
}
