package deploy

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/omnistrate-oss/omnistrate-ctl/internal/config"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/dataaccess"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/utils"

	"github.com/chelnak/ysmrr"
	openapiclient "github.com/omnistrate-oss/omnistrate-sdk-go/v1"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"
)

// buildServiceFromRepo builds a service from repository using the same logic as build-from-repo command
func buildServiceFromRepo(ctx context.Context, token, name string, releaseDescription *string, deploymentType, awsAccountID, gcpProjectID, gcpProjectNumber string) (serviceID string, environmentID string, productTierID string, undefinedResources map[string]string, err error) {
	if name == "" {
		return "", "", "", make(map[string]string), errors.New("name is required")
	}

	// Initialize spinner manager for build process
	sm := ysmrr.NewSpinnerManager()
	sm.Start()
	defer sm.Stop()

	// Step 1: Retrieve current working directory
	spinner := sm.AddSpinner("Retrieving current working directory")
	time.Sleep(1 * time.Second)
	cwd, err := os.Getwd()
	if err != nil {
		utils.HandleSpinnerError(spinner, sm, err)
		return "", "", "", make(map[string]string), err
	}
	spinner.UpdateMessage("Retrieving current working directory: Success")
	spinner.Complete()

	rootDir := cwd

	// Step 2: Retrieve the repository name and owner
	spinner = sm.AddSpinner("Retrieving repository name")
	time.Sleep(1 * time.Second)
	output, err := exec.Command("sh", "-c", `git config --get remote.origin.url | sed -E 's/:([^\/])/\/\1/g' | sed -e 's/ssh\/\/\///g' | sed -e 's/git@/https:\/\//g'`).Output()
	var repoOwner, repoName string
	if err != nil {
		// If git remote fails, use the provided name or directory name as fallback
		if name == "" {
			name = filepath.Base(cwd)
		}
		repoName = name
		repoOwner = "unknown" // Default fallback
		spinner.UpdateMessage(fmt.Sprintf("Retrieving repository name: Using fallback name '%s'", name))
		spinner.Complete()
	} else {
		repoURL := strings.TrimSpace(string(output))
		repoName = filepath.Base(repoURL)
		repoOwner = filepath.Base(filepath.Dir(repoURL))
		repoName = strings.TrimSuffix(repoName, ".git")
		if name == "" {
			name = repoName
		}
		spinner.UpdateMessage(fmt.Sprintf("Retrieving repository name: %s/%s", repoOwner, repoName))
		spinner.Complete()
	}

	// Step 2.5: Check for GitHub Personal Access Token
	spinner = sm.AddSpinner("Checking GitHub authentication")
	time.Sleep(1 * time.Second)
	var pat string
	var ghUsername string
	if config.IsGithubTokenEnvVarConfigured() {
		pat, err = config.LookupGitHubPersonalAccessToken()
		if err == nil {
			ghUsername = config.GithubTokenUserName
			spinner.UpdateMessage("Checking GitHub authentication: Using environment variable")
		}
	} else {
		pat, err = config.LookupGitHubPersonalAccessToken()
		if err != nil && !errors.As(err, &config.ErrGitHubPATNotFound) {
			utils.HandleSpinnerError(spinner, sm, err)
			return "", "", "", make(map[string]string), err
		}
		if !errors.As(err, &config.ErrGitHubPATNotFound) {
			ghUsernameOutput, err := exec.Command("gh", "api", "user", "-q", ".login").Output()
			if err == nil {
				ghUsername = strings.TrimSpace(string(ghUsernameOutput))
				spinner.UpdateMessage(fmt.Sprintf("Checking GitHub authentication: %s", ghUsername))
			} else {
				spinner.UpdateMessage("Checking GitHub authentication: Using stored PAT")
			}
		} else {
			spinner.UpdateMessage("GitHub PAT not found (will use public registry if needed)")
		}
	}
	spinner.Complete()

	// Step 3: Check for Dockerfiles in current directory and subdirectories
	spinner = sm.AddSpinner("Checking for Dockerfiles")
	time.Sleep(1 * time.Second)
	
	dockerfilePaths := make(map[string]string) // service -> dockerfile path
	err = filepath.Walk(rootDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.Name() == "Dockerfile" {
			// Use relative path from root to create service name
			relPath, err := filepath.Rel(rootDir, filepath.Dir(path))
			if err != nil {
				relPath = filepath.Dir(path)
			}
			
			var serviceName string
			if relPath == "." || relPath == "" {
				serviceName = name
			} else {
				serviceName = strings.ReplaceAll(relPath, string(filepath.Separator), "-")
			}
			dockerfilePaths[serviceName] = path
		}
		return nil
	})
	if err != nil {
		utils.HandleSpinnerError(spinner, sm, err)
		return "", "", "", make(map[string]string), err
	}

	if len(dockerfilePaths) == 0 {
		utils.HandleSpinnerError(spinner, sm, errors.New("Dockerfile not found in current directory or subdirectories"))
		return "", "", "", make(map[string]string), errors.New("Dockerfile not found in current directory or subdirectories. Repository-based build requires a Dockerfile")
	}

	spinner.UpdateMessage(fmt.Sprintf("Checking for Dockerfiles: Found %d Dockerfile(s)", len(dockerfilePaths)))
	spinner.Complete()

	// Step 4: Build and push Docker images
	spinner = sm.AddSpinner("Building and pushing Docker images")
	time.Sleep(1 * time.Second)
	
	dockerfilePathsArr := make([]string, 0)
	for _, dockerfilePath := range dockerfilePaths {
		dockerfilePathsArr = append(dockerfilePathsArr, dockerfilePath)
	}
	
	dockerPathsToImageUrls := make(map[string]string)
	platforms := []string{"linux/amd64"} // Default platform
	platformsStr := strings.Join(platforms, ",")

	for serviceName, dockerfilePath := range dockerfilePaths {
		// Set current working directory to the service context
		err = os.Chdir(filepath.Dir(dockerfilePath))
		if err != nil {
			utils.HandleSpinnerError(spinner, sm, err)
			return "", "", "", make(map[string]string), err
		}

		// Generate image URL
		label := strings.ToLower(utils.GetFirstDifferentSegmentInFilePaths(dockerfilePath, dockerfilePathsArr))
		var imageUrl string
		if label == "" {
			imageUrl = fmt.Sprintf("ghcr.io/%s/%s", strings.ToLower(repoOwner), repoName)
		} else {
			imageUrl = fmt.Sprintf("ghcr.io/%s/%s-%s", strings.ToLower(repoOwner), repoName, label)
		}

		// Add repository label to Dockerfile
		dockerfileData, err := os.ReadFile(dockerfilePath)
		if err != nil {
			utils.HandleSpinnerError(spinner, sm, err)
			return "", "", "", make(map[string]string), err
		}

		// Add GitHub source label if not already present
		dockerfileContent := string(dockerfileData)
		if !strings.Contains(dockerfileContent, "org.opencontainers.image.source") {
			dockerfileData = append(dockerfileData, []byte(fmt.Sprintf("\nLABEL org.opencontainers.image.source=\"https://github.com/%s/%s\"\n", repoOwner, repoName))...)
			err = os.WriteFile(dockerfilePath, dockerfileData, 0644)
			if err != nil {
				utils.HandleSpinnerError(spinner, sm, fmt.Errorf("failed to update Dockerfile with labels: %w", err))
				return "", "", "", make(map[string]string), err
			}
		}

		spinner.UpdateMessage(fmt.Sprintf("Building Docker image for %s: %s", serviceName, imageUrl))

		// Build Docker image
		buildCmd := exec.Command("docker", "buildx", "build", "--pull", "--platform", platformsStr, ".", "-f", dockerfilePath, "-t", imageUrl, "--load")
		
		var buildOutput bytes.Buffer
		buildCmd.Stdout = &buildOutput
		buildCmd.Stderr = &buildOutput
		
		err = buildCmd.Run()
		if err != nil {
			utils.HandleSpinnerError(spinner, sm, fmt.Errorf("failed to build Docker image: %w\nOutput: %s", err, buildOutput.String()))
			return "", "", "", make(map[string]string), err
		}

		// Login to GitHub Container Registry if PAT is available
		if pat != "" {
			loginCmd := exec.Command("docker", "login", "ghcr.io", "-u", repoOwner, "--password-stdin")
			loginCmd.Stdin = strings.NewReader(pat)
			
			var loginOutput bytes.Buffer
			loginCmd.Stdout = &loginOutput
			loginCmd.Stderr = &loginOutput
			
			err = loginCmd.Run()
			if err != nil {
				utils.HandleSpinnerError(spinner, sm, fmt.Errorf("failed to login to GHCR: %w\nOutput: %s", err, loginOutput.String()))
				return "", "", "", make(map[string]string), err
			}
		}

		// Push Docker image
		pushCmd := exec.Command("docker", "push", imageUrl)
		
		var pushOutput bytes.Buffer
		pushCmd.Stdout = &pushOutput
		pushCmd.Stderr = &pushOutput
		
		err = pushCmd.Run()
		if err != nil {
			if pat == "" {
				utils.HandleSpinnerError(spinner, sm, fmt.Errorf("failed to push Docker image (no GitHub PAT configured): %w\nOutput: %s", err, pushOutput.String()))
			} else {
				utils.HandleSpinnerError(spinner, sm, fmt.Errorf("failed to push Docker image: %w\nOutput: %s", err, pushOutput.String()))
			}
			return "", "", "", make(map[string]string), err
		}

		dockerPathsToImageUrls[dockerfilePath] = imageUrl
		spinner.UpdateMessage(fmt.Sprintf("Successfully built and pushed: %s", imageUrl))
	}

	// Return to root directory
	err = os.Chdir(rootDir)
	if err != nil {
		utils.HandleSpinnerError(spinner, sm, err)
		return "", "", "", make(map[string]string), err
	}
	
	spinner.UpdateMessage("Building and pushing Docker images: All images built and pushed successfully")
	spinner.Complete()

	// Step 5: Generate compose spec using built images
	spinner = sm.AddSpinner("Generating compose spec with built images")
	time.Sleep(1 * time.Second)

	// Create compose spec with actual image URLs instead of build contexts
	services := make(map[string]interface{})
	for serviceName, dockerfilePath := range dockerfilePaths {
		imageUrl := dockerPathsToImageUrls[dockerfilePath]
		
		service := map[string]interface{}{
			"image": imageUrl,
		}

		// Add deployment configuration based on deployment type
		if deploymentType == "byoa" {
			service["deployment"] = map[string]interface{}{
				"byoaDeployment": map[string]interface{}{
					"instanceType":     "t3.micro",
					"minimumInstances": 1,
					"maximumInstances": 3,
				},
			}
			
			// Add cloud provider account configuration
			if awsAccountID != "" {
				service["x-omnistrate-cloud-provider"] = "aws"
				service["x-omnistrate-aws-account-id"] = awsAccountID
			} else if gcpProjectID != "" {
				service["x-omnistrate-cloud-provider"] = "gcp"
				service["x-omnistrate-gcp-project-id"] = gcpProjectID
				service["x-omnistrate-gcp-project-number"] = gcpProjectNumber
			}
		} else {
			// Default to hosted deployment
			service["deployment"] = map[string]interface{}{
				"hostedDeployment": map[string]interface{}{
					"instanceType":     "t3.micro",
					"minimumInstances": 1,
					"maximumInstances": 3,
				},
			}
		}

		// Try to detect exposed ports from Dockerfile
		if dockerfileData, err := os.ReadFile(dockerfilePath); err == nil {
			lines := strings.Split(string(dockerfileData), "\n")
			for _, line := range lines {
				line = strings.TrimSpace(line)
				if strings.HasPrefix(strings.ToUpper(line), "EXPOSE ") {
					portStr := strings.TrimPrefix(strings.ToUpper(line), "EXPOSE ")
					portStr = strings.TrimSpace(portStr)
					if port, err := strconv.Atoi(portStr); err == nil {
						service["ports"] = []string{fmt.Sprintf("%d:%d", port, port)}
						break
					}
				}
			}
		}

		// Default port if none found
		if _, hasPort := service["ports"]; !hasPort {
			service["ports"] = []string{"8080:8080"}
		}

		services[serviceName] = service
	}

	composeSpec := map[string]interface{}{
		"version":  "3.8",
		"services": services,
	}

	// Add image registry authentication if GitHub PAT is available
	if pat != "" {
		composeSpec["x-omnistrate-image-registry-attributes"] = map[string]interface{}{
			"ghcr.io": map[string]interface{}{
				"auth": map[string]interface{}{
					"username": repoOwner,
					"password": pat,
				},
			},
		}
	}

	// Convert to YAML format
	yamlData, err := yaml.Marshal(composeSpec)
	if err != nil {
		utils.HandleSpinnerError(spinner, sm, err)
		return "", "", "", make(map[string]string), fmt.Errorf("failed to generate compose spec: %w", err)
	}

	spinner.UpdateMessage("Generating compose spec with built images: Success")
	spinner.Complete()

	// Step 6: Build the service using the generated compose spec
	spinner = sm.AddSpinner("Building service from compose spec")
	time.Sleep(1 * time.Second)

	request := openapiclient.BuildServiceFromComposeSpecRequest2{
		Name:               name,
		Description:        utils.ToPtr("Auto-generated service from repository"),
		ServiceLogoURL:     nil,
		Environment:        nil,
		EnvironmentType:    nil,
		FileContent:        base64.StdEncoding.EncodeToString(yamlData),
		Release:            utils.ToPtr(true),
		ReleaseAsPreferred: utils.ToPtr(true),
		ReleaseVersionName: releaseDescription,
		Dryrun:             utils.ToPtr(false),
	}

	buildRes, err := dataaccess.BuildServiceFromComposeSpec(ctx, token, request)
	if err != nil {
		utils.HandleSpinnerError(spinner, sm, err)
		return "", "", "", make(map[string]string), err
	}
	if buildRes == nil {
		utils.HandleSpinnerError(spinner, sm, errors.New("empty response from server"))
		return "", "", "", make(map[string]string), errors.New("empty response from server")
	}

	spinner.UpdateMessage("Building service from compose spec: Success")
	spinner.Complete()

	undefinedResources = make(map[string]string)
	if buildRes.UndefinedResources != nil {
		undefinedResources = *buildRes.UndefinedResources
	}

	return buildRes.ServiceID, buildRes.ServiceEnvironmentID, buildRes.ProductTierID, undefinedResources, nil
}