package build

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/omnistrate-oss/omnistrate-ctl/internal/dataaccess"
	openapiclient "github.com/omnistrate-oss/omnistrate-sdk-go/v1"
)

const (
	timeout      = 5 * time.Minute
	pollInterval = 5 * time.Second
	StatusReady  = "READY"
	StatusFailed = "FAILED"
)

// DetectSpecType analyzes YAML content to determine if it contains service plan specifications
// Returns ServicePlanSpecType if plan-specific keys are found, otherwise DockerComposeSpecType
func DetectSpecType(yamlContent map[string]any) string {
	// Improved: Recursively check for plan spec keys at any level
	planKeyGroups := [][]string{
		{"helm", "helmChart", "helmChartConfiguration"},
		{"operator", "operatorCRDConfiguration"},
		{"terraform", "terraformConfigurations"},
		{"kustomize", "kustomizeConfiguration"},
	}

	// Check if any plan-specific keys are found
	for _, keys := range planKeyGroups {
		if ContainsAnyKey(yamlContent, keys) {
			return ServicePlanSpecType
		}
	}

	return DockerComposeSpecType
}

// ContainsOmnistrateKey recursively searches for any x-omnistrate key in a map
func ContainsOmnistrateKey(m map[string]any) bool {
	for k, v := range m {
		// Check for any x-omnistrate key
		if strings.HasPrefix(k, "x-omnistrate-") {
			return true
		}
		// Recurse into nested maps
		if sub, ok := v.(map[string]any); ok {
			if ContainsOmnistrateKey(sub) {
				return true
			}
		}
		// Recurse into slices of maps
		if arr, ok := v.([]any); ok {
			for _, item := range arr {
				if subm, ok := item.(map[string]any); ok {
					if ContainsOmnistrateKey(subm) {
						return true
					}
				}
			}
		}
	}
	return false
}

// ContainsAnyKey recursively searches for any key in keys in a map
func ContainsAnyKey(m map[string]any, keys []string) bool {
	for k, v := range m {
		if slices.Contains(keys, k) {
			return true
		}
		// Recurse into nested maps
		if sub, ok := v.(map[string]any); ok {
			if ContainsAnyKey(sub, keys) {
				return true
			}
		}
		// Recurse into slices of maps
		if arr, ok := v.([]any); ok {
			for _, item := range arr {
				if subm, ok := item.(map[string]any); ok {
					if ContainsAnyKey(subm, keys) {
						return true
					}
				}
			}
		}
	}
	return false
}

// ArchiveArtifactPaths creates tar.gz archives for each artifact path and returns base64 encoded content.
// baseDir is the directory from which relative paths are resolved.
// Returns a map of relative path to base64 encoded tar.gz content.
func ArchiveArtifactPaths(baseDir string, artifactPaths []string) (map[string]string, error) {
	if len(artifactPaths) == 0 {
		return nil, nil
	}

	result := make(map[string]string)

	for _, artifactPath := range artifactPaths {
		// Resolve the artifact path relative to baseDir
		resolvedPath := artifactPath
		if !filepath.IsAbs(artifactPath) {
			resolvedPath = filepath.Join(baseDir, artifactPath)
		}

		// Clean the path
		resolvedPath = filepath.Clean(resolvedPath)

		// Check if the path exists
		info, err := os.Stat(resolvedPath) //nolint:gosec // G703: path is cleaned with filepath.Clean above
		if err != nil {
			return nil, fmt.Errorf("artifact path '%s' does not exist: %w", artifactPath, err)
		}

		if !info.IsDir() {
			// Path is a file - check if it's already a .tar.gz / .tgz file
			if isGzipTarFile(resolvedPath) {
				// Already a tar.gz file, just read and base64 encode it directly
				fileContent, readErr := os.ReadFile(resolvedPath) //nolint:gosec // G703: path is cleaned with filepath.Clean above
				if readErr != nil {
					return nil, fmt.Errorf("failed to read tar.gz file '%s': %w", artifactPath, readErr)
				}
				result[artifactPath] = base64.StdEncoding.EncodeToString(fileContent)
				continue
			}
			return nil, fmt.Errorf("artifact path '%s' is not a directory or a .tar.gz file", artifactPath)
		}

		// Create the tar.gz archive in memory and encode to base64
		base64Content, err := createTarGzBase64(resolvedPath)
		if err != nil {
			return nil, fmt.Errorf("failed to create archive for '%s': %w", artifactPath, err)
		}

		result[artifactPath] = base64Content
	}

	return result, nil
}

// createTarGzBase64 creates a tar.gz archive of a directory and returns base64 encoded content
func createTarGzBase64(sourceDir string) (string, error) {
	// Create a buffer to write the archive to
	var buf bytes.Buffer

	// Create gzip writer
	gzWriter := gzip.NewWriter(&buf)

	// Create tar writer
	tarWriter := tar.NewWriter(gzWriter)

	// Walk through the source directory
	err := filepath.Walk(sourceDir, func(path string, info os.FileInfo, err error) error { //nolint:gosec // G703: sourceDir is validated by caller
		if err != nil {
			return err
		}

		// Get the relative path
		relPath, err := filepath.Rel(sourceDir, path)
		if err != nil {
			return fmt.Errorf("failed to get relative path: %w", err)
		}

		// Skip the root directory itself
		if relPath == "." {
			return nil
		}

		// Create tar header
		header, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return fmt.Errorf("failed to create tar header: %w", err)
		}

		// Use relative path as the name in the archive
		header.Name = relPath

		// Handle symlinks
		if info.Mode()&os.ModeSymlink != 0 {
			link, err := os.Readlink(path)
			if err != nil {
				return fmt.Errorf("failed to read symlink: %w", err)
			}
			header.Linkname = link
		}

		// Write header
		if err := tarWriter.WriteHeader(header); err != nil {
			return fmt.Errorf("failed to write tar header: %w", err)
		}

		// If it's a regular file, write its content
		if info.Mode().IsRegular() {
			file, err := os.Open(filepath.Clean(path)) //nolint:gosec // path comes from filepath.Walk within known directory
			if err != nil {
				return fmt.Errorf("failed to open file: %w", err)
			}
			defer file.Close()

			if _, err := io.Copy(tarWriter, file); err != nil {
				return fmt.Errorf("failed to write file content: %w", err)
			}
		}

		return nil
	})

	if err != nil {
		return "", err
	}

	// Close tar writer first, then gzip writer
	if err := tarWriter.Close(); err != nil {
		return "", fmt.Errorf("failed to close tar writer: %w", err)
	}
	if err := gzWriter.Close(); err != nil {
		return "", fmt.Errorf("failed to close gzip writer: %w", err)
	}

	// Encode to base64 using standard encoding
	base64Content := base64.StdEncoding.EncodeToString(buf.Bytes())

	return base64Content, nil
}

// isGzipTarFile checks whether a file is already in tar.gz (.tar.gz or .tgz) format.
// It checks both the file extension and the gzip magic bytes (0x1f, 0x8b) at the start of the file.
func isGzipTarFile(filePath string) bool {
	lower := strings.ToLower(filePath)
	hasExtension := strings.HasSuffix(lower, ".tar.gz") || strings.HasSuffix(lower, ".tgz")

	// Also verify by reading the first two bytes (gzip magic number)
	f, err := os.Open(filePath) //nolint:gosec // G703: filePath is already cleaned by caller via filepath.Clean
	if err != nil {
		return hasExtension
	}
	defer f.Close()

	magic := make([]byte, 2)
	n, err := f.Read(magic)
	if err != nil || n < 2 {
		return hasExtension
	}

	hasMagic := magic[0] == 0x1f && magic[1] == 0x8b
	return hasExtension || hasMagic
}

// ServiceHierarchyResult holds the result of finding or creating the service hierarchy
type ServiceHierarchyResult struct {
	ServiceID        string
	EnvironmentID    string
	ProductTierID    string
	IsNewProductTier bool
	// ArtifactUploadingTasks contains artifact upload tasks returned by the prepare API.
	// Each task describes one artifact that needs to be uploaded, with all server-resolved parameters.
	ArtifactUploadingTasks []ArtifactUploadingTask
}

// ArtifactUploadingTask represents a single artifact upload task returned by the prepare API.
// It contains all the server-side resolved parameters needed to call the upload artifact API.
type ArtifactUploadingTask struct {
	ArtifactPath    string // The artifact path expected by the server
	AccountConfigID string // The resolved account config ID for this upload
	ServiceName     string // The service name
	ProductTierName string // The product tier name
	EnvironmentType string // The environment type (e.g., "DEV", "PROD")
}

// UniqueArtifactPathsFromTasks returns deduplicated artifact paths from ArtifactUploadingTasks
func UniqueArtifactPathsFromTasks(tasks []ArtifactUploadingTask) []string {
	seen := make(map[string]struct{})
	var paths []string
	for _, t := range tasks {
		if _, ok := seen[t.ArtifactPath]; !ok {
			seen[t.ArtifactPath] = struct{}{}
			paths = append(paths, t.ArtifactPath)
		}
	}
	return paths
}

// FindOrCreateServiceHierarchy uses the prepare service plan spec API to find or create
// the service hierarchy (service -> environment -> product tier) in a single server-side call.
// The response also includes artifact uploading tasks that the caller should follow to upload artifacts.
// environmentName and environmentType are optional - if nil, defaults to "Development" and "DEV" respectively.
func FindOrCreateServiceHierarchy(
	ctx context.Context,
	token string,
	serviceName string,
	fileData []byte,
	environmentName *string,
	environmentType *string,
) (*ServiceHierarchyResult, error) {
	// Set defaults for environment name and type if not provided
	envName := "Development"
	if environmentName != nil && *environmentName != "" {
		envName = *environmentName
	}

	envType := "DEV"
	if environmentType != nil && *environmentType != "" {
		envType = strings.ToUpper(*environmentType)
	}

	request := openapiclient.PrepareServiceFromServicePlanSpecRequest2{
		Name:            serviceName,
		Environment:     envName,
		EnvironmentType: envType,
		FileContent:     base64.StdEncoding.EncodeToString(fileData),
	}

	resp, err := dataaccess.PrepareServiceFromServicePlanSpec(ctx, token, request)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare service hierarchy: %w", err)
	}

	result := &ServiceHierarchyResult{
		ServiceID:        resp.GetServiceID(),
		EnvironmentID:    resp.GetServiceEnvironmentID(),
		ProductTierID:    resp.GetProductTierID(),
		IsNewProductTier: resp.GetIsNewProductTierCreated(),
	}

	// Extract artifact uploading tasks from the prepare response
	if sdkTasks, ok := resp.GetArtifactUploadingTasksOk(); ok && sdkTasks != nil {
		for _, t := range sdkTasks {
			result.ArtifactUploadingTasks = append(result.ArtifactUploadingTasks, ArtifactUploadingTask{
				ArtifactPath:    t.GetArtifactPath(),
				AccountConfigID: t.GetAccountConfigID(),
				ServiceName:     t.GetServiceName(),
				ProductTierName: t.GetProductTierName(),
				EnvironmentType: t.GetEnvironmentType(),
			})
		}
	}

	return result, nil
}

// ArtifactStatusCallback is called when an artifact's status changes during polling.
// artifactID is the artifact ID, status is the new status (e.g., "READY", "FAILED", "UPLOADING").
type ArtifactStatusCallback func(artifactID string, status string)

// waitForArtifactsReady polls the artifact status until all artifacts are in READY status.
// Timeout is 5 minutes. Status can be "READY", "UPLOADING", or "FAILED".
// An optional onStatusChange callback is invoked each time an artifact's status is observed.
func waitForArtifactsReady(ctx context.Context, token string, artifactIDs []string, onStatusChange ArtifactStatusCallback) error {
	deadline := time.Now().Add(timeout)

	pendingArtifacts := make(map[string]bool)
	for _, id := range artifactIDs {
		pendingArtifacts[id] = true
	}

	for time.Now().Before(deadline) {
		for artifactID := range pendingArtifacts {
			result, err := dataaccess.DescribeArtifact(ctx, token, artifactID)
			if err != nil {
				return fmt.Errorf("failed to describe artifact %s: %w", artifactID, err)
			}

			if onStatusChange != nil {
				onStatusChange(artifactID, result.Status)
			}

			switch result.Status {
			case StatusReady:
				delete(pendingArtifacts, artifactID)
			case StatusFailed:
				return fmt.Errorf("artifact %s failed to process", artifactID)
			}
			// If status is UPLOADING, continue waiting
		}

		if len(pendingArtifacts) == 0 {
			return nil
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(pollInterval):
		}
	}

	return fmt.Errorf("timeout after 5 minutes waiting for artifacts to be ready, %d artifacts still pending", len(pendingArtifacts))
}

// isCurrentDirPath returns true if the given artifact path refers to the current working directory.
func isCurrentDirPath(artifactPath string) bool {
	cleaned := filepath.Clean(artifactPath)
	return cleaned == "." || cleaned == "/" || artifactPath == "./"
}
