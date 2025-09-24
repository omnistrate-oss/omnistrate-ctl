package deploy

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
	"github.com/omnistrate-oss/omnistrate-ctl/internal/config"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/dataaccess"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/utils"

	"github.com/chelnak/ysmrr"
	openapiclient "github.com/omnistrate-oss/omnistrate-sdk-go/v1"
	"github.com/pkg/browser"
	"github.com/pkg/errors"
)

const (
	ComposeFileName = "compose.yaml"
	GitHubPATGenerateURL = "https://github.com/settings/tokens"
	DefaultProdEnvName   = "Production"
	defaultServiceName   = "default" // Default service name when no compose spec exists in the repo

	// DockerComposeSpecType is the type string for docker compose specs
	DockerComposeSpecType = "docker-compose"
	// ServicePlanSpecType is the type string for service plan specs
	ServicePlanSpecType = "service-plan"
)

// buildServiceFromRepo builds a service from repository using the same logic as build-from-repo command
func buildServiceFromRepo(ctx context.Context, token, name string, releaseDescription *string, deploymentType, awsAccountID, gcpProjectID, gcpProjectNumber, azureSubscriptionID, azureTenantID string, envVars []string, skipDockerBuild, skipServiceBuild, skipEnvironmentPromotion, skipSaasPortalInit bool, dryRun bool, platforms []string, resetPAT bool) (serviceID string, environmentID string, productTierID string, undefinedResources map[string]string, err error) {
	if name == "" {
		return "", "", "", make(map[string]string), errors.New("name is required")
	}

	// Initialize spinner manager for build process
	sm := ysmrr.NewSpinnerManager()
	sm.Start()
	defer sm.Stop()


	

	// Step 0: Check if gh is installed
	spinner := sm.AddSpinner("Checking if gh installed")
	time.Sleep(1 * time.Second)
	err = exec.Command("gh", "version").Run()
	if err != nil {
		utils.HandleSpinnerError(spinner, sm, err)
		return "", "", "", make(map[string]string), err
	}
	spinner.UpdateMessage("Checking if gh installed: Yes")
	spinner.Complete()


	// Step 1: Check if the user is in the root of the repository
	spinner = sm.AddSpinner("Checking if user is in the root of the repository")
	time.Sleep(1 * time.Second)
	cwd, err := os.Getwd()
	if err != nil {
		utils.HandleSpinnerError(spinner, sm, err)
		return "", "", "", make(map[string]string), err
	}
	if _, err := os.Stat(filepath.Join(cwd, ".git")); os.IsNotExist(err) {
		utils.HandleSpinnerError(spinner, sm, errors.New("you are not in the root of a git repository"))
		return "", "", "", make(map[string]string), err
	}
	spinner.UpdateMessage("Checking if user is in the root of the repository: Yes")
	spinner.Complete()

	rootDir := cwd

	// Step 2: Retrieve the repository name and owner
	spinner = sm.AddSpinner("Retrieving repository name")
	time.Sleep(1 * time.Second)
	output, err := exec.Command("sh", "-c", `git config --get remote.origin.url | sed -E 's/:([^\/])/\/\1/g' | sed -e 's/ssh\/\/\///g' | sed -e 's/git@/https:\/\//g'`).Output()
	if err != nil {
		utils.HandleSpinnerError(spinner, sm, err)
		return "", "", "", make(map[string]string), err
	}
	repoURL := strings.TrimSpace(string(output))
	repoName := filepath.Base(repoURL)
	repoOwner := filepath.Base(filepath.Dir(repoURL))
	repoName = strings.TrimSuffix(repoName, ".git")
	if name == "" {
		name = repoName
	}
	spinner.UpdateMessage(fmt.Sprintf("Retrieving repository name: %s/%s", repoOwner, repoName))
	spinner.Complete()

	
	// Step 3: Check if there exists a compose spec in the repository
	spinner = sm.AddSpinner("Checking if there exists a compose spec in the repository")
	time.Sleep(1 * time.Second) // Add a delay to show the spinner
	var composeSpecExists bool
	file := ComposeFileName
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
			return "", "", "", make(map[string]string), err
		}

		fileData, err = os.ReadFile(file)
		if err != nil {
			return "", "", "", make(map[string]string), err
		}

		// Load the YAML content
		parsedYaml, err = loader.ParseYAML(fileData)
		if err != nil {
			err = errors.Wrap(err, "failed to parse YAML content")
			return "", "", "", make(map[string]string), err
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
			return "", "", "", make(map[string]string), err
		}

		for _, service := range project.Services {
			if service.Build != nil {
				composeSpecHasBuildContext = true

				absContextPath, err := filepath.Abs(service.Build.Context)
				if err != nil {
					utils.HandleSpinnerError(spinner, sm, err)
					return "", "", "", make(map[string]string), err
				}

				dockerfilePaths[service.Name] = filepath.Join(absContextPath, service.Build.Dockerfile)
			}
		}
	} else {
		dockerfilePaths[defaultServiceName], err = filepath.Abs("Dockerfile")
		if err != nil {
			utils.HandleSpinnerError(spinner, sm, err)
			return "", "", "", make(map[string]string), err
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
				return "", "", "", make(map[string]string), err
			}

			if !errors.As(err, &config.ErrGitHubPATNotFound) {
				if config.IsGithubTokenEnvVarConfigured() {
					ghUsername = config.GithubTokenUserName
				} else {
					ghUsernameOutput, err := exec.Command("gh", "api", "user", "-q", ".login").Output()
					if err != nil {
						utils.HandleSpinnerError(spinner, sm, err)
						return "", "", "", make(map[string]string), err
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
					return "", "", "", make(map[string]string), err
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
				return "", "", "", make(map[string]string), err
			}
			spinner.UpdateMessage("Checking if Docker installed: Yes")
			spinner.Complete()

			// Step 6: Check if the Docker daemon is running
			spinner = sm.AddSpinner("Checking if Docker daemon is running")
			time.Sleep(1 * time.Second)                // Add a delay to show the spinner
			err = exec.Command("docker", "info").Run() // Simple way to check if Docker is available
			if err != nil {
				utils.HandleSpinnerError(spinner, sm, err)
				return "", "", "", make(map[string]string), err
			}
			spinner.UpdateMessage("Checking if Docker daemon is running: Yes")
			spinner.Complete()

			// Step 7: Check if there is an existing GitHub pat
			sm, pat, err = getOrCreatePAT(sm, resetPAT)
			if err != nil {
				utils.HandleSpinnerError(spinner, sm, err)
				return "", "", "", make(map[string]string), err
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
					return "", "", "", make(map[string]string), err
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
					return "", "", "", make(map[string]string), err
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
						return "", "", "", make(map[string]string), err
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
				return "", "", "", make(map[string]string), err
			}

			sm = ysmrr.NewSpinnerManager()
			sm.Start()

			for service, dockerfilePath := range dockerfilePaths {
				// Set current working directory to the service context
				err = os.Chdir(filepath.Dir(dockerfilePath))
				if err != nil {
					utils.HandleSpinnerError(spinner, sm, err)
					return "", "", "", make(map[string]string), err
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

				// Use the platforms parameter passed to the function
				platformsStr := strings.Join(platforms, ",")

				buildCmd := exec.Command("docker", "buildx", "build", "--pull", "--platform", platformsStr, ".", "-f", dockerfilePath, "-t", imageUrl, "--load")

				// Redirect stdout and stderr to the terminal
				buildCmd.Stdout = os.Stdout
				buildCmd.Stderr = os.Stderr

				fmt.Printf("Invoking 'docker buildx build --pull --platform %s . -f %s -t %s --load'...\n", platformsStr, dockerfilePath, imageUrl)
				err = buildCmd.Run()
				if err != nil {
					utils.HandleSpinnerError(spinner, sm, err)
					return "", "", "", make(map[string]string), err
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
					return "", "", "", make(map[string]string), err
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
					return "", "", "", make(map[string]string), err
				}

				// Convert output to string and search for the Digest line
				var digest string
				digestOutputStr := string(digestOutput)
				for _, line := range strings.Split(digestOutputStr, "\n") {
					if strings.Contains(line, "Digest:") {
						parts := strings.Split(line, ":")
						if len(parts) < 3 {
							utils.HandleSpinnerError(spinner, sm, errors.New("unable to retrieve the digest"))
							return "", "", "", make(map[string]string), err
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
					return "", "", "", make(map[string]string), err
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
					return "", "", "", make(map[string]string), err
				}

				sm = ysmrr.NewSpinnerManager()
				sm.Start()
			}

			// Change back to the root directory
			err = os.Chdir(rootDir)
			if err != nil {
				utils.HandleSpinnerError(spinner, sm, err)
				return "", "", "", make(map[string]string), err
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
					return "", "", "", make(map[string]string), err
				}
				formattedEnvVars = append(formattedEnvVars, openapiclient.EnvironmentVariable{
					Key:   envVarParts[0],
					Value: envVarParts[1],
				})
			}


			// Generate compose spec from image
			// imageRef := "postgres:16"
			// fmt.Printf("DEBUG: image reference used for compose spec generation: %q\n", imageRef)
			// generateComposeSpecRequest := openapiclient.GenerateComposeSpecFromContainerImageRequest2{
			// 	ImageRegistry:        "docker.io",
			// 	Image:                imageRef,
			// 	Username:             nil,
			// 	Password:             nil,
			// 	EnvironmentVariables: formattedEnvVars,
			// }



				// Generate compose spec from image
			generateComposeSpecRequest := openapiclient.GenerateComposeSpecFromContainerImageRequest2{
				ImageRegistry:        "ghcr.io",
				Image:                "https://github.com/porsager/postgres",
				Username:             nil,
				Password:             nil,
				EnvironmentVariables: formattedEnvVars,
			}

			var generateComposeSpecRes *openapiclient.GenerateComposeSpecFromContainerImageResult
			generateComposeSpecRes, err = dataaccess.GenerateComposeSpecFromContainerImage(ctx, token, generateComposeSpecRequest)
			if err != nil {
				utils.HandleSpinnerError(spinner, sm, err)
				return "", "", "", make(map[string]string), err
			}

			// Decode the base64 encoded file content
			fileData, err = base64.StdEncoding.DecodeString(generateComposeSpecRes.FileContent)
			if err != nil {
				utils.PrintError(err)
				return "", "", "", make(map[string]string), err
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
					user, err := dataaccess.DescribeUser(ctx, token)
					if err != nil {
						utils.HandleSpinnerError(spinner, sm, err)
						return "", "", "", make(map[string]string), err
					}

					gcpServiceAccountEmail := fmt.Sprintf("bootstrap-%s@%s.iam.gserviceaccount.com", *user.OrgId, gcpProjectID)
					fileData = append(fileData, []byte(fmt.Sprintf("      GcpServiceAccountEmail: '%s'\n", gcpServiceAccountEmail))...)
				}
				if azureSubscriptionID != ""{
					fileData = append(fileData, []byte(fmt.Sprintf("      AzureSubscriptionId: '%s'\n", azureSubscriptionID))...)
					fileData = append(fileData, []byte(fmt.Sprintf("      AzureTenantId: '%s'\n", azureTenantID))...)
				}
			}

			// Write the compose spec to a file
			err = os.WriteFile(file, fileData, 0600)
			if err != nil {
				utils.HandleSpinnerError(spinner, sm, err)
				return "", "", "", make(map[string]string), err
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
						user, err := dataaccess.DescribeUser(ctx, token)
						if err != nil {
							utils.HandleSpinnerError(spinner, sm, err)
							return "", "", "", make(map[string]string), err
						}

						gcpServiceAccountEmail := fmt.Sprintf("bootstrap-%s@%s.iam.gserviceaccount.com", *user.OrgId, gcpProjectID)
						fileData = append(fileData, []byte(fmt.Sprintf("      GcpServiceAccountEmail: '%s'\n", gcpServiceAccountEmail))...)
					}
					if azureSubscriptionID != ""{
					fileData = append(fileData, []byte(fmt.Sprintf("      AzureSubscriptionId: '%s'\n", azureSubscriptionID))...)
					fileData = append(fileData, []byte(fmt.Sprintf("      AzureTenantId: '%s'\n", azureTenantID))...)
				}
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
				return "", "", "", make(map[string]string), err
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
			return "", "", "", make(map[string]string), err
		}
	}

	// Step 14: Render the compose file: variable interpolation (if env_file appears), ${{ secrets.GitHubPAT }} replacement, build context replacement
	spinner = sm.AddSpinner("Rendering compose spec")

	if strings.Contains(string(fileData), "env_file:") {
		fileData, err = renderEnvFileAndInterpolateVariables(fileData, rootDir, file, sm, spinner)
		if err != nil {
			utils.HandleSpinnerError(spinner, sm, err)
			return "", "", "", make(map[string]string), err
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
			return "", "", "", make(map[string]string), err
		}

		spinner.UpdateMessage(fmt.Sprintf("Dry run: Wrote compose spec to %s", dryRunFile))
		spinner.Complete()
		sm.Stop()
		fmt.Printf("Dry run completed. Final compose spec written to %s\n", dryRunFile)
		return "", "", "", make(map[string]string), err
	}

	// Skip service build if flag is set
	if skipServiceBuild {
		spinner.UpdateMessage("Skipping service build (--skip-service-build flag is set)")
		spinner.Complete()
		sm.Stop()
		fmt.Println("Service build was skipped. No service was created.")
		return "", "", "", make(map[string]string), err
	}

	// Use custom service name if provided, otherwise use repo name
	serviceNameToUse := repoName
	if name != "" {
		serviceNameToUse = name
	}

	// Prepare release description pointer
	var releaseDescriptionPtr *string
	if releaseDescription != nil && *releaseDescription != "" {
		releaseDescriptionPtr = releaseDescription
	}

	// Build the service
	// Call the buildService function from the appropriate package (import if needed)
	serviceID, devEnvironmentID, devPlanID, undefinedResources, err := buildService(
		ctx,
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
	)
	if err != nil {
		utils.HandleSpinnerError(spinner, sm, err)
		return "", "", "", make(map[string]string), err
	}


	return serviceID,devEnvironmentID,devPlanID,undefinedResources,nil
}

// processFileReferences processes file references like ${{ file:path }} in the compose spec
func processFileReferences(fileData []byte, rootDir string) ([]byte, error) {
	content := string(fileData)
	
	// Pattern to match ${{ file:path }}
	re := regexp.MustCompile(`\$\{\{\s*file:([^\s}]+)\s*\}\}`)
	
	// Process all matches
	for {
		if !re.MatchString(content) {
			break
		}
		
		var processingErr error
		content = re.ReplaceAllStringFunc(content, func(match string) string {
			submatch := re.FindStringSubmatch(match)
			if len(submatch) < 2 {
				processingErr = fmt.Errorf("invalid file reference: %s", match)
				return match
			}
			
			filePath := submatch[1]
			if filePath == "" {
				processingErr = fmt.Errorf("empty file path in reference: %s", match)
				return match
			}
			
			// Resolve file path
			var fullPath string
			if filepath.IsAbs(filePath) {
				fullPath = filePath
			} else {
				fullPath = filepath.Join(rootDir, filePath)
			}
			
			// Read file content
			fileContent, err := os.ReadFile(fullPath)
			if err != nil {
				processingErr = fmt.Errorf("failed to read file %s: %v", fullPath, err)
				return match
			}
			
			return string(fileContent)
		})
		
		if processingErr != nil {
			return nil, processingErr
		}
	}
	
	return []byte(content), nil
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
				return "", "", "", make(map[string]string), err
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
