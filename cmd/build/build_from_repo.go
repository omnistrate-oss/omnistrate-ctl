package build

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/compose-spec/compose-go/loader"
	"github.com/compose-spec/compose-go/types"
	"github.com/fatih/color"
	"github.com/omnistrate-oss/omnistrate-ctl/cmd/common"

	"github.com/chelnak/ysmrr"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/config"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/dataaccess"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/model"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/utils"
	openapiclient "github.com/omnistrate-oss/omnistrate-sdk-go/v1"
	"github.com/pkg/browser"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

const (
	buildFromRepoExample = `# Build service from git repository
omctl build-from-repo

# Build service from git repository with environment variables, deployment type and cloud provider account details
omctl build-from-repo --env-var POSTGRES_PASSWORD=default --deployment-type byoa --aws-account-id 442426883376

# Build service from an existing compose spec in the repository
omctl build-from-repo --file compose.yaml

# Build service with a custom service name
omctl build-from-repo --product-name my-custom-service

# Skip building and pushing Docker image
omctl build-from-repo --skip-docker-build

# Skip multiple stages
omctl build-from-repo --skip-docker-build --skip-environment-promotion

# Run in dry-run mode (build image locally but don't push or create service)
omctl build-from-repo --dry-run

# Build for multiple platforms
omctl build-from-repo --platforms linux/amd64 --platforms linux/arm64

# Build with release description
omctl build-from-repo --release-description "v1.0.0-alpha"

# Build using github token from environment variable (GH_PAT)
set GH_PAT=ghp_xxxxxxxx
omctl build-from-repo
"`
	GitHubPATGenerateURL      = "https://github.com/settings/tokens"
	DefaultProdEnvName        = "Production"
	defaultServiceName        = "default" // Default service name when no compose spec exists in the repo. It won't show up in the resulting image or compose spec. Only intermediate use.
	ignoreKeyForFileEmbedding = "x-embed-$"
)

var BuildFromRepoCmd = &cobra.Command{
	Use:          "build-from-repo",
	Short:        "Build Service from Git Repository",
	Long:         "This command helps to build service from git repository. Run this command from the root of the repository. Make sure you have the Dockerfile in the repository and have the Docker daemon running on your machine. By default, the service name will be the repository name, but you can specify a custom service name with the --product-name flag.\n\nYou can also skip specific stages of the build process using the --skip-* flags. For example, you can skip building the Docker image with --skip-docker-build, skip creating the service with --skip-service-build, skip environment promotion with --skip-environment-promotion, or skip SaaS portal initialization with --skip-saas-portal-init.\n\nFor testing purposes, use the --dry-run flag to only build the Docker image locally without pushing, skip service creation, and generate a local spec file with a '-dry-run' suffix. Note that --dry-run cannot be used together with any of the --skip-* flags as they are mutually exclusive.",
	Example:      buildFromRepoExample,
	RunE:         runBuildFromRepo,
	SilenceUsage: true,
}

func init() {
	BuildFromRepoCmd.Flags().StringArray("env-var", nil, "Specify environment variables required for running the image. Effective only when the compose.yaml is absent. Use the format: --env-var key1=var1 --env-var key2=var2. Only effective when no compose spec exists in the repo.")
	BuildFromRepoCmd.Flags().String("deployment-type", "", "Set the deployment type. Options: 'hosted' or 'byoa' (Bring Your Own Account). Only effective when no compose spec exists in the repo.")
	BuildFromRepoCmd.Flags().String("aws-account-id", "", "AWS account ID. Must be used with --deployment-type")
	BuildFromRepoCmd.Flags().String("gcp-project-id", "", "GCP project ID. Must be used with --gcp-project-number and --deployment-type")
	BuildFromRepoCmd.Flags().String("gcp-project-number", "", "GCP project number. Must be used with --gcp-project-id and --deployment-type")
	BuildFromRepoCmd.Flags().String("azure-subscription-id", "", "Azure subscription ID. Must be used with --azure-tenant-id and --deployment-type")
	BuildFromRepoCmd.Flags().String("azure-tenant-id", "", "Azure tenant ID. Must be used with --azure-subscription-id and --deployment-type")
	BuildFromRepoCmd.Flags().Bool("reset-pat", false, "Reset the GitHub Personal Access Token (PAT) for the current user.")
	BuildFromRepoCmd.Flags().StringP("output", "o", "text", "Output format. Only text is supported")
	BuildFromRepoCmd.Flags().StringP("file", "f", ComposeFileName, "Specify the compose file to read and write to")
	BuildFromRepoCmd.Flags().String("service-name", "", "Specify a custom service name. If not provided, the repository name will be used.")
	BuildFromRepoCmd.Flags().String("product-name", "", "Specify a custom service name. If not provided, the repository name will be used.")

	// Skip flags for different stages
	BuildFromRepoCmd.Flags().Bool("skip-docker-build", false, "Skip building and pushing the Docker image")
	BuildFromRepoCmd.Flags().Bool("skip-service-build", false, "Skip building the service from the compose spec")
	BuildFromRepoCmd.Flags().Bool("skip-environment-promotion", false, "Skip creating and promoting to the production environment")
	BuildFromRepoCmd.Flags().Bool("skip-saas-portal-init", false, "Skip initializing the SaaS Portal")

	// Dry run flag
	BuildFromRepoCmd.Flags().Bool("dry-run", false, "Run in dry-run mode: only build the Docker image locally without pushing, skip service creation, and write the generated spec to a local file with '-dry-run' suffix. Cannot be used with any --skip-* flags.")

	// Force create version set flag
	BuildFromRepoCmd.Flags().Bool("force-create-service-plan-version", false, "Force create a new service plan version on release.")

	// Platform flag
	BuildFromRepoCmd.Flags().StringArray("platforms", []string{"linux/amd64"}, "Specify the platforms to build for. Use the format: --platforms linux/amd64 --platforms linux/arm64. Default is linux/amd64.")

	// Release description flag
	BuildFromRepoCmd.Flags().String("release-description", "", "Provide a description for the release version")

	// Deprecate the old --service-name flag
	if err := BuildFromRepoCmd.Flags().MarkDeprecated("service-name", "use --product-name instead"); err != nil {
		utils.PrintError(err)
		return
	}
	// Hide the deprecated flag from help
	if err := BuildFromRepoCmd.Flags().MarkHidden("service-name"); err != nil {
		utils.PrintError(err)
		return
	}

	err := BuildFromRepoCmd.MarkFlagFilename("file")
	if err != nil {
		return
	}
}

func runBuildFromRepo(cmd *cobra.Command, args []string) error {
	defer config.CleanupArgsAndFlags(cmd, &args)
	// Retrieve the flags
	envVars, err := cmd.Flags().GetStringArray("env-var")
	if err != nil {
		return err
	}

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

	azureSubscriptionID, err := cmd.Flags().GetString("azure-subscription-id")
	if err != nil {
		return err
	}
	azureTenantID, err := cmd.Flags().GetString("azure-tenant-id")
	if err != nil {
		return err
	}

	resetPAT, err := cmd.Flags().GetBool("reset-pat")
	if err != nil {
		utils.PrintError(err)
		return err
	}

	file, err := cmd.Flags().GetString("file")
	if err != nil {
		utils.PrintError(err)
		return err
	}

	// Get skip flags
	skipDockerBuild, err := cmd.Flags().GetBool("skip-docker-build")
	if err != nil {
		utils.PrintError(err)
		return err
	}

	skipServiceBuild, err := cmd.Flags().GetBool("skip-service-build")
	if err != nil {
		utils.PrintError(err)
		return err
	}

	skipEnvironmentPromotion, err := cmd.Flags().GetBool("skip-environment-promotion")
	if err != nil {
		utils.PrintError(err)
		return err
	}

	skipSaasPortalInit, err := cmd.Flags().GetBool("skip-saas-portal-init")
	if err != nil {
		utils.PrintError(err)
		return err
	}

	// Get dry-run flag
	dryRun, err := cmd.Flags().GetBool("dry-run")
	if err != nil {
		utils.PrintError(err)
		return err
	}

	// Get force-create-service-plan-version flag
	forceCreateServicePlanVersion, err := cmd.Flags().GetBool("force-create-service-plan-version")
	if err != nil {
		utils.PrintError(err)
		return err
	}

	// Get release-description flag
	releaseDescription, err := cmd.Flags().GetString("release-description")
	if err != nil {
		utils.PrintError(err)
		return err
	}

	// Get the service name from flag
	serviceName, err := cmd.Flags().GetString("service-name")
	if err != nil {
		utils.PrintError(err)
		return err
	}
	productName, err := cmd.Flags().GetString("product-name")
	if err != nil {
		utils.PrintError(err)
		return err
	}

	if serviceName != "" && productName != "" {
		err = errors.New("only one of service-name or product-name can be provided")
		utils.PrintError(err)
		return err
	}

	// Use product-name if provided, otherwise use service-name
	// Since flags are mutually exclusive, only one will be set
	if productName != "" {
		serviceName = productName
	}

	// Check for incompatible flag combinations
	if dryRun {
		// If dry-run is set, other skip flags should not be set
		if skipDockerBuild || skipServiceBuild || skipEnvironmentPromotion || skipSaasPortalInit {
			err = errors.New("--dry-run flag is not compatible with --skip-* flags (they are mutually exclusive)")
			utils.PrintError(err)
			return err
		}

		// In dry-run mode, we implicitly skip these steps
		skipServiceBuild = true
		skipEnvironmentPromotion = true
		skipSaasPortalInit = true
	}

	// Convert the file path to an absolute path
	file, err = filepath.Abs(file)
	if err != nil {
		utils.PrintError(err)
		return err
	}

	outputFlagValue, err := cmd.Flags().GetString("output")
	if err != nil {
		utils.PrintError(err)
		return err
	}

	// Validate the output format
	if outputFlagValue != "text" {
		err = errors.New("only text output format is supported")
		utils.PrintError(err)
		return err
	}

	// Validate deployment type and cloud provider account details
	if deploymentType != "" {
		if deploymentType != "hosted" && deploymentType != "byoa" {
			err = errors.New("invalid deployment type. Options: 'hosted' or 'byoa'")
			utils.PrintError(err)
			return err
		}

		if awsAccountID == "" && gcpProjectID == "" && azureSubscriptionID == "" {
			err = errors.New(fmt.Sprintf("AWS account ID or GCP project ID or Azure subscription ID are required for %s deployment type", deploymentType))
			utils.PrintError(err)
			return err
		}
		if gcpProjectID != "" && gcpProjectNumber == "" {
			err = errors.New("GCP project number is required with GCP project ID")
			utils.PrintError(err)
			return err
		}
		if gcpProjectID == "" && gcpProjectNumber != "" {
			err = errors.New("GCP project ID is required with GCP project number")
			utils.PrintError(err)
			return err
		}
		if azureSubscriptionID != "" && azureTenantID == "" {
			err = errors.New("Azure tenant ID is required with Azure subscription ID")
			utils.PrintError(err)
			return err
		}

	}

	// Initialize the spinner manager
	var sm ysmrr.SpinnerManager
	var spinner *ysmrr.Spinner
	sm = ysmrr.NewSpinnerManager()
	sm.Start()

	// Step 0: Validate user is currently logged in
	spinner = sm.AddSpinner("Checking if user is logged in")
	time.Sleep(1 * time.Second) // Add a delay to show the spinner
	spinner.Complete()
	sm.Stop()

	token, err := common.GetTokenWithLogin()
	if err != nil {
		utils.PrintError(err)
		return err
	}

	sm = ysmrr.NewSpinnerManager()
	sm.Start()

	// Only check for gh if we're not skipping Docker build
	if !skipDockerBuild {
		spinner = sm.AddSpinner("Checking if gh installed")
		time.Sleep(1 * time.Second) // Add a delay to show the spinner
		err = exec.Command("gh", "version").Run()
		if err != nil {
			utils.HandleSpinnerError(spinner, sm, err)
			return err
		}
		spinner.UpdateMessage("Checking if gh installed: Yes")
		spinner.Complete()
	}

	// Step 1: Check if the user is in the root of the repository
	spinner = sm.AddSpinner("Checking if user is in the root of the repository")
	time.Sleep(1 * time.Second) // Add a delay to show the spinner
	cwd, err := os.Getwd()
	if err != nil {
		utils.HandleSpinnerError(spinner, sm, err)
		return err
	}
	if _, err := os.Stat(filepath.Join(cwd, ".git")); os.IsNotExist(err) {
		utils.HandleSpinnerError(spinner, sm, errors.New("you are not in the root of a git repository"))
		return err
	}
	spinner.UpdateMessage("Checking if user is in the root of the repository: Yes")
	spinner.Complete()

	rootDir := cwd

	// Step 2: Retrieve the repository name
	spinner = sm.AddSpinner("Retrieving repository name")
	time.Sleep(1 * time.Second) // Add a delay to show the spinner
	output, err := exec.Command("sh", "-c", `git config --get remote.origin.url | sed -E 's/:([^\/])/\/\1/g' | sed -e 's/ssh\/\/\///g' | sed -e 's/git@/https:\/\//g'`).Output()
	if err != nil {
		utils.HandleSpinnerError(spinner, sm, err)
		return err
	}
	repoURL := strings.TrimSpace(string(output))
	repoName := filepath.Base(repoURL)
	repoOwner := filepath.Base(filepath.Dir(repoURL))
	repoName = strings.TrimSuffix(repoName, ".git") // Extract repo name
	spinner.UpdateMessage(fmt.Sprintf("Retrieving repository name: %s/%s", repoOwner, repoName))
	spinner.Complete()

	// Step 3: Check if there exists a compose spec in the repository
	spinner = sm.AddSpinner("Checking if there exists a compose spec in the repository")
	time.Sleep(1 * time.Second) // Add a delay to show the spinner
	var composeSpecExists bool
	if _, err = os.Stat(file); os.IsNotExist(err) {
		composeSpecExists = false
	} else {
		composeSpecExists = true
	}
	yesOrNo := "No"
	if composeSpecExists {
		yesOrNo = "Yes"
	}
	spinner.UpdateMessage(fmt.Sprintf("Checking if there exists a compose spec in the repository: %s", yesOrNo))
	spinner.Complete()

	var fileData []byte
	var parsedYaml map[string]interface{}
	var project *types.Project
	dockerfilePaths := make(map[string]string)        // service -> dockerfile path
	versionTaggedImageUrls := make(map[string]string) // service -> image url with digest tag
	var pat string
	var ghUsername string

	composeSpecHasBuildContext := false
	if composeSpecExists {
		// Load the compose file
		if _, err = os.Stat(file); os.IsNotExist(err) {
			utils.PrintError(err)
			return err
		}

		fileData, err = os.ReadFile(file)
		if err != nil {
			return err
		}

		// Load the YAML content
		parsedYaml, err = loader.ParseYAML(fileData)
		if err != nil {
			err = errors.Wrap(err, "failed to parse YAML content")
			return err
		}

		// Decode spec YAML into a compose project
		if project, err = loader.LoadWithContext(context.Background(), types.ConfigDetails{
			ConfigFiles: []types.ConfigFile{
				{
					Config: parsedYaml,
				},
			},
		}); err != nil {
			err = errors.Wrap(err, "invalid compose")
			return err
		}

		for _, service := range project.Services {
			if service.Build != nil {
				composeSpecHasBuildContext = true

				absContextPath, err := filepath.Abs(service.Build.Context)
				if err != nil {
					utils.HandleSpinnerError(spinner, sm, err)
					return err
				}

				dockerfilePaths[service.Name] = filepath.Join(absContextPath, service.Build.Dockerfile)
			}
		}
	} else {
		dockerfilePaths[defaultServiceName], err = filepath.Abs("Dockerfile")
		if err != nil {
			utils.HandleSpinnerError(spinner, sm, err)
			return err
		}
	}

	dockerfilePathsArr := make([]string, 0)
	for _, dockerfilePath := range dockerfilePaths {
		dockerfilePathsArr = append(dockerfilePathsArr, dockerfilePath)
	}

	if !composeSpecExists || composeSpecHasBuildContext {
		// Skip Docker build if flag is set
		if skipDockerBuild {
			spinner = sm.AddSpinner("Skipping Docker build (--skip-docker-build flag is set)")
			spinner.Complete()

			// We still need to get the GitHub username for the compose spec
			spinner = sm.AddSpinner("Getting GitHub username for compose spec")
			pat, err = config.LookupGitHubPersonalAccessToken()
			if err != nil && !errors.As(err, &config.ErrGitHubPATNotFound) {
				utils.HandleSpinnerError(spinner, sm, err)
				return err
			}

			if !errors.As(err, &config.ErrGitHubPATNotFound) {
				if config.IsGithubTokenEnvVarConfigured() {
					ghUsername = config.GithubTokenUserName
				} else {
					ghUsernameOutput, err := exec.Command("gh", "api", "user", "-q", ".login").Output()
					if err != nil {
						utils.HandleSpinnerError(spinner, sm, err)
						return err
					}
					ghUsername = strings.TrimSpace(string(ghUsernameOutput))
				}
				spinner.UpdateMessage(fmt.Sprintf("Getting GitHub username for compose spec: %s", ghUsername))
			} else {
				spinner.UpdateMessage("GitHub PAT not found, will prompt if needed later")
			}
			spinner.Complete()

			// Set placeholder image URLs if needed
			for service := range dockerfilePaths {
				label := strings.ToLower(utils.GetFirstDifferentSegmentInFilePaths(dockerfilePaths[service], dockerfilePathsArr))
				var imageUrl string
				if label == "" {
					imageUrl = fmt.Sprintf("ghcr.io/%s/%s", strings.ToLower(repoOwner), repoName)
				} else {
					imageUrl = fmt.Sprintf("ghcr.io/%s/%s-%s", strings.ToLower(repoOwner), repoName, label)
				}
				versionTaggedImageUrls[service] = fmt.Sprintf("%s:latest", imageUrl)
			}
		} else {
			// Step 4: Check if the Dockerfile exists
			for _, dockerfilePath := range dockerfilePaths {
				spinner = sm.AddSpinner(fmt.Sprintf("Checking if %s exists in the repository", dockerfilePath))
				time.Sleep(1 * time.Second) // Add a delay to show the spinner

				if _, err = os.Stat(dockerfilePath); os.IsNotExist(err) {
					utils.HandleSpinnerError(spinner, sm, errors.New(fmt.Sprintf("%s not found in the repository", dockerfilePath)))
					return err
				}

				spinner.UpdateMessage(fmt.Sprintf("Checking if %s exists in the repository: Yes", dockerfilePath))
				spinner.Complete()
			}

			// Step 5: Check if Docker is installed
			spinner = sm.AddSpinner("Checking if Docker installed")
			time.Sleep(1 * time.Second)                   // Add a delay to show the spinner
			err = exec.Command("docker", "version").Run() // Simple way to check if Docker is available
			if err != nil {
				utils.HandleSpinnerError(spinner, sm, err)
				return err
			}
			spinner.UpdateMessage("Checking if Docker installed: Yes")
			spinner.Complete()

			// Step 6: Check if the Docker daemon is running
			spinner = sm.AddSpinner("Checking if Docker daemon is running")
			time.Sleep(1 * time.Second)                // Add a delay to show the spinner
			err = exec.Command("docker", "info").Run() // Simple way to check if Docker is available
			if err != nil {
				utils.HandleSpinnerError(spinner, sm, err)
				return err
			}
			spinner.UpdateMessage("Checking if Docker daemon is running: Yes")
			spinner.Complete()

			// Step 7: Check if there is an existing GitHub pat
			sm, pat, err = getOrCreatePAT(sm, resetPAT)
			if err != nil {
				utils.HandleSpinnerError(spinner, sm, err)
				return err
			}

			// Step 8: Retrieve the GitHub username
			spinner = sm.AddSpinner("Retrieving GitHub username")
			time.Sleep(1 * time.Second) // Add a delay to show the spinner
			if config.IsGithubTokenEnvVarConfigured() {
				ghUsername = config.GithubTokenUserName
			} else {
				ghUsernameOutput, err := exec.Command("gh", "api", "user", "-q", ".login").Output()
				if err != nil {
					utils.HandleSpinnerError(spinner, sm, err)
					return err
				}
				ghUsername = strings.TrimSpace(string(ghUsernameOutput))
			}
			spinner.UpdateMessage(fmt.Sprintf("Retrieving GitHub username: %s", ghUsername))
			spinner.Complete()

			// Step 9: Label the docker image with the repository name
			spinner = sm.AddSpinner("Labeling Docker image with the repository name")
			for _, dockerfilePath := range dockerfilePaths {
				// Read the Dockerfile
				var dockerfileData []byte
				dockerfileData, err = os.ReadFile(dockerfilePath)
				if err != nil {
					utils.HandleSpinnerError(spinner, sm, err)
					return err
				}

				// Check if the Dockerfile already has the label
				if strings.Contains(string(dockerfileData), "LABEL org.opencontainers.image.source") {
					spinner.UpdateMessage("Labeling Docker image with the repository name: Already labeled")
				} else {
					// Append the label to the Dockerfile
					dockerfileData = append(dockerfileData, []byte(fmt.Sprintf("\nLABEL org.opencontainers.image.source=\"https://github.com/%s/%s\"\n", repoOwner, repoName))...)

					// Write the Dockerfile back
					err = os.WriteFile(dockerfilePath, dockerfileData, 0600)
					if err != nil {
						utils.HandleSpinnerError(spinner, sm, err)
						return err
					}

					spinner.UpdateMessage(fmt.Sprintf("Labeling Docker image with the repository name: %s/%s", repoOwner, repoName))
				}
			}

			spinner.Complete()

			// Step 10: Login to GitHub Container Registry
			spinner = sm.AddSpinner("Logging in to ghcr.io")
			spinner.Complete()
			sm.Stop()
			loginCmd := exec.Command("docker", "login", "ghcr.io", "--username", ghUsername, "--password", pat)

			// Redirect stdout and stderr to the terminal
			loginCmd.Stdout = os.Stdout
			loginCmd.Stderr = os.Stderr

			fmt.Printf("Invoking 'docker login ghcr.io --username %s --password ******'...\n", ghUsername)
			err = loginCmd.Run()
			if err != nil {
				utils.HandleSpinnerError(spinner, sm, err)
				return err
			}

			sm = ysmrr.NewSpinnerManager()
			sm.Start()

			for service, dockerfilePath := range dockerfilePaths {
				// Set current working directory to the service context
				err = os.Chdir(filepath.Dir(dockerfilePath))
				if err != nil {
					utils.HandleSpinnerError(spinner, sm, err)
					return err
				}

				// Step 11: Build docker image
				label := strings.ToLower(utils.GetFirstDifferentSegmentInFilePaths(dockerfilePath, dockerfilePathsArr))
				var imageUrl string
				if label == "" {
					imageUrl = fmt.Sprintf("ghcr.io/%s/%s", strings.ToLower(repoOwner), repoName)
				} else {
					imageUrl = fmt.Sprintf("ghcr.io/%s/%s-%s", strings.ToLower(repoOwner), repoName, label)
				}

				spinner = sm.AddSpinner(fmt.Sprintf("Building Docker image: %s", imageUrl))
				spinner.Complete()
				sm.Stop()

				// Get the platforms defined in flag
				platformsFlag, err := cmd.Flags().GetStringArray("platforms")
				if err != nil {
					utils.HandleSpinnerError(spinner, sm, err)
					return err
				}

				// Join the platforms list with comma as separator
				platformsStr := strings.Join(platformsFlag, ",")

				buildCmd := exec.Command("docker", "buildx", "build", "--pull", "--platform", platformsStr, ".", "-f", dockerfilePath, "-t", imageUrl, "--load")

				// Redirect stdout and stderr to the terminal
				buildCmd.Stdout = os.Stdout
				buildCmd.Stderr = os.Stderr

				fmt.Printf("Invoking 'docker buildx build --pull --platform %s . -f %s -t %s --load'...\n", platformsStr, dockerfilePath, imageUrl)
				err = buildCmd.Run()
				if err != nil {
					utils.HandleSpinnerError(spinner, sm, err)
					return err
				}

				sm = ysmrr.NewSpinnerManager()
				sm.Start()

				// In dry-run mode, skip pushing to registry and use local image tag
				if dryRun {
					spinner = sm.AddSpinner("Dry run: Using local image tag (skipping push)")
					spinner.Complete()
					versionTaggedImageUrls[service] = fmt.Sprintf("%s:latest", imageUrl)
					continue
				}

				// Step 12: Push docker image to GitHub Container Registry
				spinner = sm.AddSpinner("Pushing Docker image to GitHub Container Registry")
				spinner.Complete()
				sm.Stop()
				pushCmd := exec.Command("docker", "push", imageUrl)

				// Redirect stdout and stderr to the terminal
				pushCmd.Stdout = os.Stdout
				pushCmd.Stderr = os.Stderr

				fmt.Printf("Invoking 'docker push %s'...\n", imageUrl)
				err = pushCmd.Run()
				if err != nil {
					utils.HandleSpinnerError(spinner, sm, err)
					return err
				}

				sm = ysmrr.NewSpinnerManager()
				sm.Start()

				// Retrieve the digest
				spinner = sm.AddSpinner("Retrieving the digest for the image")
				digestCmd := exec.Command("docker", "buildx", "imagetools", "inspect", imageUrl)

				var digestOutput []byte
				digestOutput, err = digestCmd.Output()
				if err != nil {
					utils.HandleSpinnerError(spinner, sm, err)
					return err
				}

				// Convert output to string and search for the Digest line
				var digest string
				digestOutputStr := string(digestOutput)
				for _, line := range strings.Split(digestOutputStr, "\n") {
					if strings.Contains(line, "Digest:") {
						parts := strings.Split(line, ":")
						if len(parts) < 3 {
							utils.HandleSpinnerError(spinner, sm, errors.New("unable to retrieve the digest"))
							return err
						}
						digest = fmt.Sprintf("sha-%s", strings.TrimSpace(strings.Split(line, ":")[2]))
						break
					}
				}

				spinner.Complete()
				sm.Stop()

				fmt.Printf("Retrieved digest: %s\n", digest)

				sm = ysmrr.NewSpinnerManager()
				sm.Start()

				imageUrlWithDigestTag := fmt.Sprintf("%s:%s", imageUrl, digest)
				versionTaggedImageUrls[service] = imageUrlWithDigestTag

				// Tag the image with the digest
				spinner = sm.AddSpinner("Tagging the image with the digest")
				spinner.Complete()
				sm.Stop()

				tagCmd := exec.Command("docker", "tag", imageUrl, imageUrlWithDigestTag)

				tagCmd.Stdout = os.Stdout
				tagCmd.Stderr = os.Stderr

				fmt.Printf("Invoking 'docker tag %s %s'...\n", imageUrl, imageUrlWithDigestTag)
				if err = tagCmd.Run(); err != nil {
					utils.HandleSpinnerError(spinner, sm, err)
					return err
				}

				sm = ysmrr.NewSpinnerManager()
				sm.Start()

				// Push the image with the digest tag
				spinner = sm.AddSpinner("Pushing the image with the digest tag")
				spinner.Complete()
				sm.Stop()

				pushCmd = exec.Command("docker", "push", imageUrlWithDigestTag)

				// Redirect stdout and stderr to the terminal
				pushCmd.Stdout = os.Stdout
				pushCmd.Stderr = os.Stderr

				fmt.Printf("Invoking 'docker push %s'...\n", imageUrlWithDigestTag)
				err = pushCmd.Run()
				if err != nil {
					utils.HandleSpinnerError(spinner, sm, err)
					return err
				}

				sm = ysmrr.NewSpinnerManager()
				sm.Start()
			}

			// Change back to the root directory
			err = os.Chdir(rootDir)
			if err != nil {
				utils.HandleSpinnerError(spinner, sm, err)
				return err
			}
		}

		// Step 13: Generate compose spec from the Docker image
		spinner = sm.AddSpinner("Generating compose spec from the Docker image")
		if !composeSpecExists {
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
				ImageRegistry:        "ghcr.io",
				Image:                strings.TrimPrefix(versionTaggedImageUrls[defaultServiceName], "ghcr.io/"),
				Username:             utils.ToPtr(ghUsername),
				Password:             utils.ToPtr(pat),
				EnvironmentVariables: formattedEnvVars,
			}

			var generateComposeSpecRes *openapiclient.GenerateComposeSpecFromContainerImageResult
			generateComposeSpecRes, err = dataaccess.GenerateComposeSpecFromContainerImage(cmd.Context(), token, generateComposeSpecRequest)
			if err != nil {
				utils.HandleSpinnerError(spinner, sm, err)
				return err
			}

			// Decode the base64 encoded file content
			fileData, err = base64.StdEncoding.DecodeString(generateComposeSpecRes.FileContent)
			if err != nil {
				utils.PrintError(err)
				return err
			}

			// Replace the actual PAT with ${{ secrets.GitHubPAT }}
			fileData = []byte(strings.ReplaceAll(string(fileData), pat, "${{ secrets.GitHubPAT }}"))

			// Replace the image tag with build tag
			fileData = []byte(strings.ReplaceAll(string(fileData), fmt.Sprintf("image: %s", versionTaggedImageUrls[defaultServiceName]), "build:\n      context: .\n      dockerfile: Dockerfile"))
			composeSpecHasBuildContext = true

			// Append the deployment section to the compose spec
			switch deploymentType {
			case "hosted":
				fileData = append(fileData, []byte("  deployment:\n")...)
				fileData = append(fileData, []byte("    hostedDeployment:\n")...)
			case "byoa":
				fileData = append(fileData, []byte("  deployment:\n")...)
				fileData = append(fileData, []byte("    byoaDeployment:\n")...)
			}

			if deploymentType != "" {
				if awsAccountID != "" {
					fileData = append(fileData, []byte(fmt.Sprintf("      AwsAccountId: '%s'\n", awsAccountID))...)
					awsBootstrapRoleAccountARN := fmt.Sprintf("arn:aws:iam::%s:role/omnistrate-bootstrap-role", awsAccountID)
					fileData = append(fileData, []byte(fmt.Sprintf("      AwsBootstrapRoleAccountArn: '%s'\n", awsBootstrapRoleAccountARN))...)
				}
				if gcpProjectID != "" {
					fileData = append(fileData, []byte(fmt.Sprintf("      GcpProjectId: '%s'\n", gcpProjectID))...)
					fileData = append(fileData, []byte(fmt.Sprintf("      GcpProjectNumber: '%s'\n", gcpProjectNumber))...)

					// Get organization id
					user, err := dataaccess.DescribeUser(cmd.Context(), token)
					if err != nil {
						utils.HandleSpinnerError(spinner, sm, err)
						return err
					}

					gcpServiceAccountEmail := fmt.Sprintf("bootstrap-%s@%s.iam.gserviceaccount.com", *user.OrgId, gcpProjectID)
					fileData = append(fileData, []byte(fmt.Sprintf("      GcpServiceAccountEmail: '%s'\n", gcpServiceAccountEmail))...)
				}
				fileData = appendAzureConfig(fileData, azureSubscriptionID, azureTenantID)
			}

			// Write the compose spec to a file
			err = os.WriteFile(file, fileData, 0600)
			if err != nil {
				utils.HandleSpinnerError(spinner, sm, err)
				return err
			}
			spinner.UpdateMessage(fmt.Sprintf("Generating compose spec from the Docker image: saved to %s", file))
			spinner.Complete()
		} else {
			// Append the deployment section to the compose spec if it doesn't exist
			if !strings.Contains(string(fileData), "deployment:") {
				switch deploymentType {
				case "hosted":
					fileData = append(fileData, []byte("  deployment:\n")...)
					fileData = append(fileData, []byte("    hostedDeployment:\n")...)
				case "byoa":
					fileData = append(fileData, []byte("  deployment:\n")...)
					fileData = append(fileData, []byte("    byoaDeployment:\n")...)
				}

				if deploymentType != "" {
					if awsAccountID != "" {
						fileData = append(fileData, []byte(fmt.Sprintf("      AwsAccountId: '%s'\n", awsAccountID))...)
						awsBootstrapRoleAccountARN := fmt.Sprintf("arn:aws:iam::%s:role/omnistrate-bootstrap-role", awsAccountID)
						fileData = append(fileData, []byte(fmt.Sprintf("      AwsBootstrapRoleAccountArn: '%s'\n", awsBootstrapRoleAccountARN))...)
					}
					if gcpProjectID != "" {
						fileData = append(fileData, []byte(fmt.Sprintf("      GcpProjectId: '%s'\n", gcpProjectID))...)
						fileData = append(fileData, []byte(fmt.Sprintf("      GcpProjectNumber: '%s'\n", gcpProjectNumber))...)

						// Get organization id
						user, err := dataaccess.DescribeUser(cmd.Context(), token)
						if err != nil {
							utils.HandleSpinnerError(spinner, sm, err)
							return err
						}

						gcpServiceAccountEmail := fmt.Sprintf("bootstrap-%s@%s.iam.gserviceaccount.com", *user.OrgId, gcpProjectID)
						fileData = append(fileData, []byte(fmt.Sprintf("      GcpServiceAccountEmail: '%s'\n", gcpServiceAccountEmail))...)
					}
					fileData = appendAzureConfig(fileData, azureSubscriptionID, azureTenantID)
				}
			}

			// Append the image registry attributes to the compose spec if it doesn't exist
			if !strings.Contains(string(fileData), "x-omnistrate-image-registry-attributes") {
				fileData = append(fileData, []byte(fmt.Sprintf(`
x-omnistrate-image-registry-attributes:
  ghcr.io:
    auth:
      password: ${{ secrets.GitHubPAT }}
      username: %s
`, ghUsername))...)
			}

			// Write the compose spec to a file
			err = os.WriteFile(file, fileData, 0600)
			if err != nil {
				utils.HandleSpinnerError(spinner, sm, err)
				return err
			}
			spinner.UpdateMessage(fmt.Sprintf("Generating compose spec from the Docker image: saved to %s", file))
			spinner.Complete()
		}
	}

	// Step 13: Get or create a GitHub PAT if needed
	if strings.Contains(string(fileData), "${{ secrets.GitHubPAT }}") && pat == "" {
		sm, pat, err = getOrCreatePAT(sm, resetPAT)
		if err != nil {
			utils.HandleSpinnerError(spinner, sm, err)
			return err
		}
	}

	// Step 14: Render the compose file: variable interpolation (if env_file appears), ${{ secrets.GitHubPAT }} replacement, build context replacement
	spinner = sm.AddSpinner("Rendering compose spec")

	if strings.Contains(string(fileData), "env_file:") {
		fileData, err = renderEnvFileAndInterpolateVariables(fileData, rootDir, file, sm, spinner)
		if err != nil {
			utils.HandleSpinnerError(spinner, sm, err)
			return err
		}
	}

	// Render the ${{ secrets.GitHubPAT }} in the compose file if needed
	if strings.Contains(string(fileData), "${{ secrets.GitHubPAT }}") {
		fileData = []byte(strings.ReplaceAll(string(fileData), "${{ secrets.GitHubPAT }}", pat))
	}

	// Render build context sections into image fields in the compose file if needed
	if composeSpecHasBuildContext {
		dockerPathsToImageUrls := make(map[string]string)
		for service, imageUrl := range versionTaggedImageUrls {
			dockerPathsToImageUrls[dockerfilePaths[service]] = imageUrl
		}
		fileData = []byte(utils.ReplaceBuildContext(string(fileData), dockerPathsToImageUrls))
	}

	spinner.UpdateMessage("Rendering compose spec: complete")
	spinner.Complete()

	// Step 15: Building service from the compose spec
	spinner = sm.AddSpinner("Building service from the compose spec")

	// If we're in dry-run mode, save the compose spec to a file with '-dry-run' suffix
	if dryRun {
		// Get the file extension
		fileExt := filepath.Ext(file)
		baseName := file[:len(file)-len(fileExt)]
		dryRunFile := fmt.Sprintf("%s-dry-run%s", baseName, fileExt)

		// Write the compose spec to the dry-run file
		err = os.WriteFile(dryRunFile, fileData, 0600)
		if err != nil {
			utils.HandleSpinnerError(spinner, sm, err)
			return err
		}

		spinner.UpdateMessage(fmt.Sprintf("Dry run: Wrote compose spec to %s", dryRunFile))
		spinner.Complete()
		sm.Stop()
		fmt.Printf("Dry run completed. Final compose spec written to %s\n", dryRunFile)
		return nil
	}

	// Skip service build if flag is set
	if skipServiceBuild {
		spinner.UpdateMessage("Skipping service build (--skip-service-build flag is set)")
		spinner.Complete()
		sm.Stop()
		fmt.Println("Service build was skipped. No service was created.")
		return nil
	}

	// Use custom service name if provided, otherwise use repo name
	serviceNameToUse := repoName
	if serviceName != "" {
		serviceNameToUse = serviceName
	}

	// Prepare release description pointer
	var releaseDescriptionPtr *string
	if releaseDescription != "" {
		releaseDescriptionPtr = &releaseDescription
	}

	// Build the service
	serviceID, devEnvironmentID, devPlanID, undefinedResources, _, err := buildService(
		cmd.Context(),
		fileData,
		token,
		serviceNameToUse,
		DockerComposeSpecType,
		nil,
		nil,
		nil,
		nil,
		true,
		true,
		releaseDescriptionPtr,
		false,
		forceCreateServicePlanVersion,
	)
	if err != nil {
		utils.HandleSpinnerError(spinner, sm, err)
		return err
	}

	spinner.UpdateMessage(fmt.Sprintf("Building service from the compose spec: built service %s (service ID: %s)", serviceNameToUse, serviceID))
	spinner.Complete()

	// Print warning if there are any undefined resources
	if len(undefinedResources) > 0 {
		sm.Stop()

		utils.PrintWarning("The following resources appear in the service plan but were not defined in the spec:")
		for resourceName, resourceID := range undefinedResources {
			utils.PrintWarning(fmt.Sprintf("  %s: %s", resourceName, resourceID))
		}
		utils.PrintWarning("These resources were not processed during the build. If you no longer need them, please deprecate and remove them from the service plan manually in UI or using the API.")

		sm = ysmrr.NewSpinnerManager()
		sm.Start()
	}

	// Skip environment promotion if flag is set
	var prodEnvironmentID string
	if skipEnvironmentPromotion {
		spinner = sm.AddSpinner("Skipping environment promotion (--skip-environment-promotion flag is set)")
		spinner.Complete()
	} else {
		// Step 16: Check if the production environment is set up
		spinner = sm.AddSpinner("Checking if the production environment is set up")
		time.Sleep(1 * time.Second) // Add a delay to show the spinner
		prodEnvironmentID, err = checkIfProdEnvExists(cmd.Context(), token, serviceID)
		if err != nil {
			utils.HandleSpinnerError(spinner, sm, err)
			return err
		}
		yesOrNo = "No"
		if prodEnvironmentID != "" {
			yesOrNo = "Yes"
		}
		spinner.UpdateMessage(fmt.Sprintf("Checking if the production environment is set up: %s", yesOrNo))
		spinner.Complete()
		// Step 17: Create a production environment if it does not exist
		if prodEnvironmentID == "" {
			spinner = sm.AddSpinner("Creating a production environment")
			prodEnvironmentID, err = createProdEnv(cmd.Context(), token, serviceID, devEnvironmentID)
			if err != nil {
				utils.HandleSpinnerError(spinner, sm, err)
				return err
			}
			spinner.UpdateMessage(fmt.Sprintf("Creating a production environment: created environment %s (environment ID: %s)", DefaultProdEnvName, prodEnvironmentID))
			spinner.Complete()
		}

		// Step 18: Promote the service to the production environment
		spinner = sm.AddSpinner(fmt.Sprintf("Promoting the service to the %s environment", DefaultProdEnvName))
		err = dataaccess.PromoteServiceEnvironment(cmd.Context(), token, serviceID, devEnvironmentID)
		if err != nil {
			utils.PrintError(err)
			return err
		}
		spinner.UpdateMessage("Promoting the service to the production environment: Success")
		spinner.Complete()

		// Step 19: Set this service plan as the default service plan in production
		spinner = sm.AddSpinner("Setting the service plan as the default service plan in production")

		// Describe the dev product tier
		var devProductTier *openapiclient.DescribeProductTierResult
		devProductTier, err = dataaccess.DescribeProductTier(cmd.Context(), token, serviceID, devPlanID)
		if err != nil {
			utils.HandleSpinnerError(spinner, sm, err)
			return err
		}

		// Find the production plan with the same name as the dev plan
		var prodPlanID string
		service, err := dataaccess.DescribeService(cmd.Context(), token, serviceID)
		if err != nil {
			utils.HandleSpinnerError(spinner, sm, err)
			return err
		}
		for _, env := range service.ServiceEnvironments {
			if env.Id != prodEnvironmentID {
				continue
			}
			for _, plan := range env.ServicePlans {
				if plan.Name == devProductTier.Name {
					prodPlanID = plan.ProductTierID
					break
				}
			}
		}

		// Find the latest version of the production plan
		targetVersion, err := dataaccess.FindLatestVersion(cmd.Context(), token, serviceID, prodPlanID)
		if err != nil {
			utils.HandleSpinnerError(spinner, sm, err)
			return err
		}

		// Set the default service plan
		_, err = dataaccess.SetDefaultServicePlan(cmd.Context(), token, serviceID, prodPlanID, targetVersion)
		if err != nil {
			utils.HandleSpinnerError(spinner, sm, err)
			return err
		}
		spinner.UpdateMessage("Setting current version as the default service plan version in production: Success")
		spinner.Complete()
	}

	// Step 20: Initialize the SaaS Portal
	var prodEnvironment *openapiclient.DescribeServiceEnvironmentResult

	if skipSaasPortalInit || skipEnvironmentPromotion {
		// Skip SaaS Portal initialization if either flag is set
		spinner = sm.AddSpinner("Skipping SaaS Portal initialization")
		if skipSaasPortalInit {
			spinner.UpdateMessage("Skipping SaaS Portal initialization (--skip-saas-portal-init flag is set)")
		} else {
			spinner.UpdateMessage("Skipping SaaS Portal initialization (--skip-environment-promotion flag is set)")
		}
		spinner.Complete()
	} else if config.IsProd() && !skipEnvironmentPromotion && prodEnvironmentID != "" {
		prodEnvironment, err = dataaccess.DescribeServiceEnvironment(cmd.Context(), token, serviceID, prodEnvironmentID)
		if err != nil {
			utils.PrintError(err)
			return err
		}

		if !checkIfSaaSPortalReady(prodEnvironment) {
			spinner = sm.AddSpinner("Initializing the SaaS Portal. This may take a few minutes.")

			for {
				prodEnvironment, err = dataaccess.DescribeServiceEnvironment(cmd.Context(), token, serviceID, prodEnvironmentID)
				if err != nil {
					utils.PrintError(err)
					return err
				}

				if checkIfSaaSPortalReady(prodEnvironment) {
					break
				}

				time.Sleep(5 * time.Second)
			}

			spinner.Complete()
		}

		// Step 21: Retrieve the SaaS Portal URL
		spinner = sm.AddSpinner("Retrieving the SaaS Portal URL")
		time.Sleep(1 * time.Second) // Add a delay to show the spinner
		spinner.Complete()
	}

	sm.Stop()

	println()
	println()
	println()
	fmt.Println("Congratulations! Your service has been successfully built and deployed.")
	if config.IsProd() && !skipSaasPortalInit && !skipEnvironmentPromotion && prodEnvironment != nil {
		utils.PrintURL("You can access the SaaS Portal at", getSaaSPortalURL(prodEnvironment, serviceID, prodEnvironmentID))
	}

	println()

	// Check if the cloud provider account(s) are verified
	awsAccountUnverified := false
	gcpAccountUnverified := false
	var unverifiedAwsAccountConfigID, unverifiedGcpAccountConfigID string
	if awsAccountID != "" || gcpProjectID != "" {
		accounts, err := dataaccess.ListAccounts(cmd.Context(), token, "all")
		if err != nil {
			utils.HandleSpinnerError(spinner, sm, err)
			return err
		}

		for _, account := range accounts.AccountConfigs {
			if account.Status == model.Verifying.String() && account.AwsAccountID != nil && *account.AwsAccountID == awsAccountID {
				awsAccountUnverified = true
				unverifiedAwsAccountConfigID = account.Id
			}

			if account.Status == model.Verifying.String() && account.GcpProjectID != nil && *account.GcpProjectID == gcpProjectID && account.GcpProjectNumber != nil && *account.GcpProjectNumber == gcpProjectNumber {
				gcpAccountUnverified = true
				unverifiedGcpAccountConfigID = account.Id
			}
		}
	}

	urlMsg := color.New(color.FgCyan).SprintFunc()
	if awsAccountUnverified || gcpAccountUnverified {
		fmt.Println("Next steps:")
		fmt.Printf("1.")
		if awsAccountUnverified {
			account, err := dataaccess.DescribeAccount(cmd.Context(), token, unverifiedAwsAccountConfigID)
			if err != nil {
				utils.PrintError(err)
				return err
			}
			fmt.Printf(" Verify your cloud provider account %s following the instructions below:\n", account.Name)
			fmt.Printf("  - For CloudFormation users: Please create your CloudFormation Stack using the provided template at %s. Watch the CloudFormation guide at %s for help.\n", urlMsg(*account.AwsCloudFormationTemplateURL), urlMsg(dataaccess.AwsCloudFormationGuideURL))
			fmt.Printf("  - For Terraform users: Execute the Terraform scripts available at %s, by using the Account Config Identity ID below. For guidance our Terraform instructional video is at %s.\n", urlMsg(dataaccess.AwsGcpTerraformScriptsURL), urlMsg(dataaccess.AwsGcpTerraformGuideURL))
		}

		if gcpAccountUnverified {
			account, err := dataaccess.DescribeAccount(cmd.Context(), token, unverifiedGcpAccountConfigID)
			if err != nil {
				utils.PrintError(err)
				return err
			}
			fmt.Printf(" Verify your cloud provider account %s following the instructions below:\n", account.Name)
			fmt.Printf("  - Execute the Terraform scripts available at %s, by using the Account Config Identity ID below. For guidance our Terraform instructional video is at %s.\n", urlMsg(dataaccess.AwsGcpTerraformScriptsURL), urlMsg(dataaccess.AwsGcpTerraformGuideURL))
		}

		fmt.Printf("2. After account verified, play around with the SaaS Portal! Subscribe to your service and create instance deployments.\n")
		fmt.Printf("3. A compose spec has been generated from the Docker image. You can customize it further by editing the %s file. Refer to the documentation %s for more information.\n", filepath.Base(file), urlMsg("https://docs.omnistrate.com/getting-started/compose-spec/"))
		fmt.Printf("4. Push any changes to the repository and automatically update the service by running 'omctl build-from-repo' again.\n")
	} else {
		fmt.Println("Next steps:")
		fmt.Printf("1. Play around with the SaaS Portal! Subscribe to your service and create instance deployments.\n")
		fmt.Printf("2. A compose spec has been generated from the Docker image. You can customize it further by editing the %s file. Refer to the documentation %s for more information.\n", filepath.Base(file), urlMsg("https://docs.omnistrate.com/getting-started/compose-spec/"))
		fmt.Printf("3. Push any changes to the repository and automatically update the service by running 'omctl build-from-repo' again.\n")
	}

	return nil
}

// Helper functions

// appendAzureConfig appends Azure configuration to fileData if Azure credentials are provided
func appendAzureConfig(fileData []byte, azureSubscriptionID, azureTenantID string) []byte {
	if azureSubscriptionID != "" {
		fileData = append(fileData, []byte(fmt.Sprintf("      AzureSubscriptionId: '%s'\n", azureSubscriptionID))...)
		fileData = append(fileData, []byte(fmt.Sprintf("      AzureTenantId: '%s'\n", azureTenantID))...)
	}
	return fileData
}

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
		utils.PrintError(err)
		return "", err
	}

	prodEnvironmentID, err := dataaccess.CreateServiceEnvironment(ctx, token,
		DefaultProdEnvName,
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
		utils.PrintError(err)
		return "", err
	}

	return prodEnvironmentID, nil
}

func getOrCreatePAT(sm ysmrr.SpinnerManager, resetPAT bool) (newSm ysmrr.SpinnerManager, pat string, err error) {
	newSm = sm
	spinner := sm.AddSpinner("Checking for existing GitHub Personal Access Token")
	time.Sleep(1 * time.Second) // Add a delay to show the spinner
	pat, err = config.LookupGitHubPersonalAccessToken()
	if err != nil && !errors.As(err, &config.ErrGitHubPATNotFound) {
		utils.HandleSpinnerError(spinner, sm, err)
		return
	}
	if err == nil && !resetPAT {
		spinner.UpdateMessage("Checking for existing GitHub Personal Access Token: Yes")
		spinner.Complete()
	}
	if err != nil && !errors.As(err, &config.ErrGitHubPATNotFound) {
		utils.HandleSpinnerError(spinner, sm, err)
		return
	}
	if errors.As(err, &config.ErrGitHubPATNotFound) || resetPAT {
		// Prompt user to enter GitHub pat
		spinner.UpdateMessage("Checking for existing GitHub Personal Access Token: No GitHub Personal Access Token found.")
		spinner.Complete()
		sm.Stop()
		utils.PrintWarning("[Action Required] GitHub Personal Access Token (PAT) is required to push the Docker image to GitHub Container Registry.")
		utils.PrintWarning("Please follow the instructions below to generate a GitHub Personal Access Token with the following scopes: write:packages, delete:packages.")
		utils.PrintWarning("The token will be stored securely on your machine and will not be shared with anyone.")
		fmt.Println()
		fmt.Println("Instructions to generate a GitHub Personal Access Token:")
		fmt.Println("1. Click on the 'Generate new token' button. Choose 'Generate new token (classic)'. Authenticate with your GitHub account.")
		fmt.Println(`2. Enter / Select the following details:
  - Enter Note: "omnistrate-cli" or any other note you prefer
  - Select Expiration: "No expiration"
  - Select the following scopes:	
    - write:packages
    - delete:packages
	- read:org`)
		fmt.Println("3. Click 'Generate token' and copy the token to your clipboard.")
		fmt.Println()

		fmt.Println("Redirecting you to the GitHub Personal Access Token generation page in your default browser in a few seconds...")
		fmt.Println()
		fmt.Print("If the browser does not open automatically, open the following URL:\n\n")
		fmt.Printf("%s\n\n", GitHubPATGenerateURL)

		time.Sleep(5 * time.Second)
		err = browser.OpenURL(GitHubPATGenerateURL)
		if err != nil {
			err = errors.New(fmt.Sprintf("Error opening browser: %v\n", err))
			utils.PrintError(err)
			return
		}

		utils.PrintSuccess("Please paste the GitHub Personal Access Token: ")
		var userInput string
		_, err = fmt.Scanln(&userInput)
		if err != nil {
			utils.PrintError(err)
			return
		}
		pat = strings.TrimSpace(userInput)

		// Save the GitHub PAT
		err = config.CreateOrUpdateGitHubPersonalAccessToken(pat)
		if err != nil {
			utils.PrintError(err)
			return
		}

		newSm = ysmrr.NewSpinnerManager()
		newSm.Start()
	}

	return
}

func RenderFile(fileData []byte, rootDir string, file string, sm ysmrr.SpinnerManager, spinner *ysmrr.Spinner) (
	newFileData []byte, err error) {
	newFileData = fileData

	newFileData, err = renderFileReferences(newFileData, file, sm, spinner)
	if err != nil {
		return
	}

	if strings.Contains(string(newFileData), "env_file:") {
		newFileData, err = renderEnvFileAndInterpolateVariables(newFileData, rootDir, file, sm, spinner)
		if err != nil {
			return
		}
	}
	return
}

func renderEnvFileAndInterpolateVariables(
	fileData []byte, rootDir string, file string, sm ysmrr.SpinnerManager, spinner *ysmrr.Spinner) (
	newFileData []byte, err error) {
	// Replace `$` with `$$` to avoid interpolation. Do not replace for `${...}` since it's used to specify variable interpolations
	fileData = []byte(strings.ReplaceAll(string(fileData), "$", "$$"))   // Escape $ to $$
	fileData = []byte(strings.ReplaceAll(string(fileData), "$${", "${")) // Unescape $${ to ${ for variable interpolation
	fileData = []byte(strings.ReplaceAll(string(fileData), "${{ secrets.GitHubPAT }}", "$${{ secrets.GitHubPAT }}"))

	// Write the compose spec to a temporary file
	tempFile := filepath.Join(rootDir, filepath.Base(file)+".tmp")
	err = os.WriteFile(tempFile, fileData, 0600)
	if err != nil {
		utils.HandleSpinnerError(spinner, sm, err)
		return
	}

	// Render the compose file using docker compose config
	renderCmd := exec.Command("docker", "compose", "-f", tempFile, "config")
	cmdOut := &bytes.Buffer{}
	cmdErr := &bytes.Buffer{}
	renderCmd.Stdout = cmdOut
	renderCmd.Stderr = cmdErr

	err = renderCmd.Run()
	if err != nil {
		if spinner != nil {
			spinner.Error()
			sm.Stop()
		}
		_, _ = fmt.Fprintf(os.Stderr, "%s", cmdErr.String())
		utils.HandleSpinnerError(spinner, sm, err)

		return
	}
	newFileData = cmdOut.Bytes()

	// Remove the temporary file
	err = os.Remove(tempFile)
	if err != nil {
		utils.HandleSpinnerError(spinner, sm, err)
		return
	}

	// Docker compose config command escapes the $ character by adding a $ in front of it, so we need to unescape it
	newFileData = []byte(strings.ReplaceAll(string(newFileData), "$$", "$"))

	// Quote numeric cpus values in deploy.resources
	// Match: cpus: <number> where the number is NOT quoted
	re := regexp.MustCompile(`(?m)(^\s*cpus:\s*)([0-9.]+)\s*$`)
	newFileData = []byte(re.ReplaceAllString(string(newFileData), `$1"$2"`))

	return
}

func renderFileReferences(
	fileData []byte, file string, sm ysmrr.SpinnerManager, spinner *ysmrr.Spinner) (
	newFileData []byte, err error) {
	re := regexp.MustCompile(`(?m)^(?P<indent>[ \t]+)?(?P<key>[\S\t ]+)?{{[ \t]*\$file:(?P<filepath>[^\s}]+)[ \t]*}}`)
	var filePathIndex, indentIndex, keyIndex int
	groupNames := re.SubexpNames()
	for i, name := range groupNames {
		if name == "indent" {
			indentIndex = i
		}
		if name == "key" {
			keyIndex = i
		}
		if name == "filepath" {
			filePathIndex = i
		}
	}

	var renderingErr error
	newFileDataStr := re.ReplaceAllStringFunc(string(fileData), func(match string) (replacement string) {
		replacement = match

		submatches := re.FindStringSubmatch(match)
		addedIndentation := submatches[indentIndex]
		key := submatches[keyIndex]
		if strings.HasPrefix(key, ignoreKeyForFileEmbedding) {
			key = ""
		}

		filePath := submatches[filePathIndex]
		if len(filePath) == 0 {
			renderingErr = fmt.Errorf("no file path found in file reference '%s'", match)
			return
		}

		// Read file content
		cleanedFilePath := filepath.Clean(filePath)
		fileDir := filepath.Dir(file)
		isRelative := !filepath.IsAbs(cleanedFilePath)
		if isRelative {
			cleanedFilePath = filepath.Join(fileDir, cleanedFilePath)
		}

		if _, fileErr := os.Stat(cleanedFilePath); os.IsNotExist(fileErr) {
			renderingErr = fmt.Errorf("file '%s' does not exist", filePath)
			return
		}

		fileContent, readErr := os.ReadFile(cleanedFilePath)
		if readErr != nil {
			renderingErr = fmt.Errorf("file '%s' could not be read", filePath)
			return
		}

		// Render the file (in case it uses nested file references)
		renderedFileContentBytes, nestedRenderErr := renderFileReferences(fileContent, cleanedFilePath, sm, spinner)
		if nestedRenderErr != nil {
			renderingErr = errors.Wrapf(nestedRenderErr,
				"failed to replace file references for file '%s'", filePath)
			return
		}

		// Add indentation of parent context
		replacement = string(renderedFileContentBytes)
		if len(addedIndentation) > 0 {
			// Add indentation to each line
			lines := strings.Split(replacement, "\n")
			for i, line := range lines {
				if i == 0 {
					lines[i] = addedIndentation + key + line
				} else if len(line) > 0 {
					lines[i] = addedIndentation + line
				}
			}
			replacement = strings.Join(lines, "\n")
		} else {
			replacement = key + replacement
		}

		return
	})

	// Handle error
	if renderingErr != nil {
		err = errors.Wrapf(renderingErr, "error rendering file '%s'", file)
		if spinner != nil {
			spinner.Error()
			sm.Stop()
		}
		_, _ = fmt.Fprintf(os.Stderr, "%s", err.Error())
		utils.HandleSpinnerError(spinner, sm, err)

		return
	}

	newFileData = []byte(newFileDataStr)
	return
}
