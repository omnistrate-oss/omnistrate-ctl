package dataaccess

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	openapiclient "github.com/omnistrate-oss/omnistrate-sdk-go/v1"
)

// ArtifactUploadResult represents the result of uploading an artifact
type ArtifactUploadResult struct {
	// Unique identifier for the uploaded artifact
	ArtifactID string
}

// ArtifactDescribeResult represents the result of describing an artifact
type ArtifactDescribeResult struct {
	// Unique identifier for the artifact
	ArtifactID string
	// Status of the artifact
	Status string
}

// UploadArtifact uploads a base64 encoded tar.gz artifact to Omnistrate
// base64Content is the base64 encoded tar.gz content
// artifactPath is the path to the deployment artifact
// serviceName is the name of the service
// productTierName is the name of the product tier
// accountConfigID is the account config ID associated with the deployment artifact
// environmentType is the type of environment (e.g., "DEV", "PROD")
func UploadArtifact(
	ctx context.Context,
	token string,
	base64Content string,
	artifactPath string,
	serviceName string,
	productTierName string,
	accountConfigID string,
	environmentType string,
) (*ArtifactUploadResult, error) {
	ctxWithToken := context.WithValue(ctx, openapiclient.ContextAccessToken, token)
	apiClient := getV1Client()

	var r *http.Response
	defer func() {
		if r != nil {
			_ = r.Body.Close()
		}
	}()

	resp, r, err := apiClient.DeploymentArtifactApiAPI.DeploymentArtifactApiUploadDeploymentArtifact(ctxWithToken).
		UploadDeploymentArtifactRequest2(openapiclient.UploadDeploymentArtifactRequest2{
			AccountConfigID:       accountConfigID,
			ArtifactPath:          artifactPath,
			Base64EncodedArtifact: base64Content,
			ProductTierName:       productTierName,
			ServiceName:           serviceName,
			EnvironmentType:       environmentType,
		}).Execute()

	if err != nil {
		return nil, handleV1Error(err)
	}

	// Clean up the response ID (remove surrounding quotes and newlines)
	return &ArtifactUploadResult{
		ArtifactID: strings.Trim(resp, "\"\n\t "),
	}, nil
}

// UploadArtifactParams holds parameters for uploading an artifact
type UploadArtifactParams struct {
	Base64Content   string
	ArtifactPath    string
	ServiceName     string
	ProductTierName string
	AccountConfigID string
	EnvironmentType string
}

// UploadArtifacts uploads multiple artifacts and returns a map of path to artifact ID
// artifacts is a map of path to UploadArtifactParams
func UploadArtifacts(
	ctx context.Context,
	token string,
	artifacts map[string]UploadArtifactParams,
) (map[string]string, error) {
	if len(artifacts) == 0 {
		return nil, nil
	}

	result := make(map[string]string)

	for path, params := range artifacts {
		uploadResult, err := UploadArtifact(
			ctx,
			token,
			params.Base64Content,
			params.ArtifactPath,
			params.ServiceName,
			params.ProductTierName,
			params.AccountConfigID,
			params.EnvironmentType,
		)
		if err != nil {
			return nil, err
		}
		result[path] = uploadResult.ArtifactID
	}

	return result, nil
}

// DescribeArtifact retrieves information about an uploaded artifact
func DescribeArtifact(
	ctx context.Context,
	token string,
	artifactID string,
) (*ArtifactDescribeResult, error) {
	ctxWithToken := context.WithValue(ctx, openapiclient.ContextAccessToken, token)
	apiClient := getV1Client()

	var r *http.Response
	defer func() {
		if r != nil {
			_ = r.Body.Close()
		}
	}()

	resp, r, err := apiClient.DeploymentArtifactApiAPI.DeploymentArtifactApiDescribeDeploymentArtifact(ctxWithToken, artifactID).Execute()
	if err != nil {
		return nil, handleV1Error(err)
	}

	return &ArtifactDescribeResult{
		ArtifactID: resp.GetId(),
		Status:     resp.GetStatus(),
	}, nil
}

// ListArtifactsResult represents the result of listing artifacts
type ListArtifactsResult struct {
	Artifacts []ArtifactUploadResult
}

// ListArtifacts retrieves a list of all artifacts for the current user/organization
func ListArtifacts(
	ctx context.Context,
	token string,
) (*ListArtifactsResult, error) {
	// TODO: Implement when ListDeploymentArtifacts API is available in SDK
	return nil, fmt.Errorf("list artifacts API not yet implemented")
}

// DeleteArtifact deletes an uploaded artifact
func DeleteArtifact(
	ctx context.Context,
	token string,
	artifactID string,
) error {
	// TODO: Implement when DeleteDeploymentArtifact API is available in SDK
	return fmt.Errorf("delete artifact API not yet implemented")
}
