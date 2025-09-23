package deploy

import (
	"context"
	"encoding/base64"

	"github.com/omnistrate-oss/omnistrate-ctl/cmd/build"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/dataaccess"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/utils"

	openapiclient "github.com/omnistrate-oss/omnistrate-sdk-go/v1"
	"github.com/pkg/errors"
)

// buildServiceSpec builds a service in DEV environment with release-as-preferred from spec file
func buildServiceSpec(ctx context.Context, fileData []byte, token, name, specType string, releaseDescription *string) (serviceID string, environmentID string, productTierID string, undefinedResources map[string]string, err error) {
	if name == "" {
		return "", "", "", make(map[string]string), errors.New("name is required")
	}

	if specType == "" {
		return "", "", "", make(map[string]string), errors.New("specType is required")
	}

	switch specType {
	case build.ServicePlanSpecType:
		request := openapiclient.BuildServiceFromServicePlanSpecRequest2{
			Name:               name,
			Description:        nil,
			ServiceLogoURL:     nil,
			Environment:        nil,
			EnvironmentType:    nil,
			FileContent:        base64.StdEncoding.EncodeToString(fileData),
			Release:            utils.ToPtr(true),
			ReleaseAsPreferred: utils.ToPtr(true),
			ReleaseVersionName: releaseDescription,
			Dryrun:             utils.ToPtr(false),
		}

		buildRes, err := dataaccess.BuildServiceFromServicePlanSpec(ctx, token, request)
		if err != nil {
			return "", "", "", make(map[string]string), err
		}
		if buildRes == nil {
			return "", "", "", make(map[string]string), errors.New("empty response from server")
		}

		undefinedResources := make(map[string]string)
		if buildRes.UndefinedResources != nil {
			undefinedResources = *buildRes.UndefinedResources
		}

		return buildRes.ServiceID, buildRes.ServiceEnvironmentID, buildRes.ProductTierID, undefinedResources, nil

	case build.DockerComposeSpecType:
		request := openapiclient.BuildServiceFromComposeSpecRequest2{
			Name:               name,
			Description:        nil,
			ServiceLogoURL:     nil,
			Environment:        nil,
			EnvironmentType:    nil,
			FileContent:        base64.StdEncoding.EncodeToString(fileData),
			Release:            utils.ToPtr(true),
			ReleaseAsPreferred: utils.ToPtr(true),
			ReleaseVersionName: releaseDescription,
			Dryrun:             utils.ToPtr(false),
		}

		buildRes, err := dataaccess.BuildServiceFromComposeSpec(ctx, token, request)
		if err != nil {
			return "", "", "", make(map[string]string), err
		}
		if buildRes == nil {
			return "", "", "", make(map[string]string), errors.New("empty response from server")
		}

		undefinedResources = make(map[string]string)
		if buildRes.UndefinedResources != nil {
			undefinedResources = *buildRes.UndefinedResources
		}

		return buildRes.ServiceID, buildRes.ServiceEnvironmentID, buildRes.ProductTierID, undefinedResources, nil

	default:
		return "", "", "", make(map[string]string), errors.Errorf("unsupported spec type: %s (supported: %s, %s)", specType, build.ServicePlanSpecType, build.DockerComposeSpecType)
	}
}