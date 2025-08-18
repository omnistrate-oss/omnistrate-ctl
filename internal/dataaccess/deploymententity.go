package dataaccess

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/pkg/errors"
	"os"
	"path/filepath"

	"fmt"
	"io"
	"net/http"
)

func GetInstanceDeploymentEntity(ctx context.Context, token string, instanceID string, deploymentType string, deploymentName string) (output string, err error) {
	httpClient := getRetryableHttpClient()

	urlPath := fmt.Sprintf("http://localhost:80/2022-09-01-00/%s/%s/%s", deploymentType, instanceID, deploymentName)
	request, err := http.NewRequest(http.MethodGet, urlPath, nil)
	if err != nil {
		return "", err
	}
	request = request.WithContext(ctx)
	request.Header.Add("Authorization", token)

	var response *http.Response
	defer func() {
		if response != nil {
			_ = response.Body.Close()
		}
	}()

	response, err = httpClient.Do(request)
	if err != nil {
		err = errors.Wrap(err, "Could not retrieve instance deployment information. Please try executing the command within the dataplane agent pod.")
		return "", err
	}

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return "", err
	}

	if response.StatusCode != http.StatusOK {
		// Include the response body in error message to provide detailed error information
		return "", fmt.Errorf("failed to get instance deployment entity: %s - %s", response.Status, string(body))
	}

	return string(body), nil
}

func PauseInstanceDeploymentEntity(ctx context.Context, token string, instanceID string, deploymentType string, deploymentName string) (err error) {
	httpClient := getRetryableHttpClient()

	urlPath := fmt.Sprintf("http://localhost:80/2022-09-01-00/%s/pause/%s/%s", deploymentType, instanceID, deploymentName)
	request, err := http.NewRequest(http.MethodPost, urlPath, nil)
	if err != nil {
		return err
	}
	request = request.WithContext(ctx)
	request.Header.Add("Authorization", token)

	var response *http.Response
	defer func() {
		if response != nil {
			_ = response.Body.Close()
		}
	}()

	response, err = httpClient.Do(request)
	if err != nil {
		err = errors.Wrap(err, "failed to pause instance deployment entity, you need to run it on dataplane agent")
		return err
	}

	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to pause instance deployment entity: %s", response.Status)
	}

	return nil
}

func ResumeInstanceDeploymentEntity(ctx context.Context, token string, instanceID string, deploymentType string, deploymentName string, deploymentAction string) (err error) {
	httpClient := getRetryableHttpClient()

	urlPath := fmt.Sprintf("http://localhost:80/2022-09-01-00/%s/resume/%s/%s", deploymentType, instanceID, deploymentName)
	// Set payload
	var payload map[string]interface{}
	switch deploymentType {
	case "terraform":
		if deploymentAction == "" {
			err = fmt.Errorf("terraform action is required for terraform deployment type")
			return
		}

		payload = map[string]interface{}{
			"token":           token,
			"name":            deploymentName,
			"instanceID":      instanceID,
			"terraformAction": deploymentAction,
		}
	default:
		return fmt.Errorf("unsupported deployment type: %s", deploymentType)
	}

	// Convert payload to JSON bytes
	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return
	}

	// Create new request with the JSON payload
	request, err := http.NewRequest(http.MethodPost, urlPath, bytes.NewBuffer(jsonPayload))
	if err != nil {
		return
	}

	request = request.WithContext(ctx)
	request.Header.Add("Authorization", token)
	request.Header.Set("Content-Type", "application/json")

	var response *http.Response
	defer func() {
		if response != nil {
			_ = response.Body.Close()
		}
	}()

	response, err = httpClient.Do(request)
	if err != nil {
		err = errors.Wrap(err, "failed to resume instance deployment entity, you need to run it on dataplane agent")
		return err
	}

	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to resume instance deployment entity: %s", response.Status)
	}

	return nil
}

func PatchInstanceDeploymentEntity(ctx context.Context, token string, instanceID string, deploymentType string, deploymentName string, patchedFilePath string, deploymentAction string) (err error) {
	httpClient := getRetryableHttpClient()

	urlPath := fmt.Sprintf("http://localhost:80/2022-09-01-00/%s/%s/%s", deploymentType, instanceID, deploymentName)
	// Set payload
	var payload map[string]interface{}
	switch deploymentType {
	case "terraform":
		if deploymentAction == "" {
			err = fmt.Errorf("deployment action is required for terraform deployment type")
			return
		}

		// walk through the directory and read all files
		var patchedFileContents map[string][]byte
		patchedFileContents, err = getDirectoryContents(patchedFilePath)
		if err != nil {
			err = errors.Wrap(err, "failed to read terraform patched files")
			return
		}

		payload = map[string]interface{}{
			"token":           token,
			"name":            deploymentName,
			"instanceID":      instanceID,
			"filesContents":   patchedFileContents,
			"terraformAction": deploymentAction,
		}
	default:
		return fmt.Errorf("unsupported deployment type: %s", deploymentType)
	}

	// Convert payload to JSON bytes
	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return
	}

	// Create new request with the JSON payload
	request, err := http.NewRequest(http.MethodPatch, urlPath, bytes.NewBuffer(jsonPayload))
	if err != nil {
		return
	}

	request = request.WithContext(ctx)
	request.Header.Add("Authorization", token)
	request.Header.Set("Content-Type", "application/json")

	var response *http.Response
	defer func() {
		if response != nil {
			_ = response.Body.Close()
		}
	}()

	response, err = httpClient.Do(request)
	if err != nil {
		err = errors.Wrap(err, "failed to patch instance deployment entity, you need to run it on dataplane agent")
		return err
	}

	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to patch instance deployment entity: %s", response.Status)
	}

	return nil
}

func getDirectoryContents(dirPath string) (map[string][]byte, error) {
	contents := make(map[string][]byte)

	err := filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Read file contents
		fileContent, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("failed to read file %s: %v", path, err)
		}

		// Get relative path
		relPath, err := filepath.Rel(dirPath, path)
		if err != nil {
			return fmt.Errorf("failed to get relative path for %s: %v", path, err)
		}

		// Store in map
		contents[relPath] = fileContent

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to walk directory: %v", err)
	}

	return contents, nil
}
