package dataaccess

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/pkg/errors"

	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

func GetInstanceDeploymentEntity(ctx context.Context, token string, instanceID string, deploymentType string, deploymentName string) (output string, err error) {
	httpClient := getRetryableHttpClient()

	request, err := newLocalDeploymentAgentRequest(ctx, http.MethodGet, token, nil, deploymentType, instanceID, deploymentName)
	if err != nil {
		return "", err
	}

	var response *http.Response
	defer func() {
		if response != nil {
			_ = response.Body.Close()
		}
	}()

	response, err = doLocalDeploymentAgentRequest(httpClient, request)
	if err != nil {
		err = errors.Wrap(err, "Could not retrieve instance deployment information. Please try executing the command within the dataplane agent pod.")
		return "", err
	}

	if response.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to get instance deployment entity: %s", response.Status)
	}

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return "", err
	}

	return string(body), nil
}

func PauseInstanceDeploymentEntity(ctx context.Context, token string, instanceID string, deploymentType string, deploymentName string) (err error) {
	httpClient := getRetryableHttpClient()

	request, err := newLocalDeploymentAgentRequest(ctx, http.MethodPost, token, nil, deploymentType, "pause", instanceID, deploymentName)
	if err != nil {
		return err
	}

	var response *http.Response
	defer func() {
		if response != nil {
			_ = response.Body.Close()
		}
	}()

	response, err = doLocalDeploymentAgentRequest(httpClient, request)
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
	request, err := newLocalDeploymentAgentRequest(ctx, http.MethodPost, token, bytes.NewBuffer(jsonPayload), deploymentType, "resume", instanceID, deploymentName)
	if err != nil {
		return
	}
	request.Header.Set("Content-Type", "application/json")

	var response *http.Response
	defer func() {
		if response != nil {
			_ = response.Body.Close()
		}
	}()

	response, err = doLocalDeploymentAgentRequest(httpClient, request)
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
	request, err := newLocalDeploymentAgentRequest(ctx, http.MethodPatch, token, bytes.NewBuffer(jsonPayload), deploymentType, instanceID, deploymentName)
	if err != nil {
		return
	}
	request.Header.Set("Content-Type", "application/json")

	var response *http.Response
	defer func() {
		if response != nil {
			_ = response.Body.Close()
		}
	}()

	response, err = doLocalDeploymentAgentRequest(httpClient, request)
	if err != nil {
		err = errors.Wrap(err, "failed to patch instance deployment entity, you need to run it on dataplane agent")
		return err
	}

	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to patch instance deployment entity: %s", response.Status)
	}

	return nil
}

func newLocalDeploymentAgentRequest(ctx context.Context, method, token string, body io.Reader, pathSegments ...string) (*http.Request, error) {
	request, err := http.NewRequestWithContext(ctx, method, localDeploymentAgentURL(pathSegments...), body)
	if err != nil {
		return nil, err
	}

	request.Header.Add("Authorization", token)
	return request, nil
}

func localDeploymentAgentURL(pathSegments ...string) string {
	escapedSegments := make([]string, len(pathSegments))
	for i, pathSegment := range pathSegments {
		escapedSegments[i] = url.PathEscape(pathSegment)
	}

	return "http://localhost:80/2022-09-01-00/" + strings.Join(escapedSegments, "/")
}

func doLocalDeploymentAgentRequest(httpClient *http.Client, request *http.Request) (*http.Response, error) {
	if request == nil || request.URL == nil {
		return nil, fmt.Errorf("local deployment agent request is missing a URL")
	}

	if request.URL.Scheme != "http" || request.URL.Host != "localhost:80" {
		return nil, fmt.Errorf("refusing to contact non-local deployment agent URL %q", request.URL.Redacted())
	}

	return httpClient.Do(request) //nolint:gosec // request host is validated to the local dataplane agent
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
		fileContent, err := os.ReadFile(filepath.Clean(path)) //nolint:gosec // path comes from filepath.Walk on a local directory
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
