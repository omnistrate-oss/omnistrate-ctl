package dataaccess

import (
	"context"

	openapiclient "github.com/omnistrate-oss/omnistrate-sdk-go/v1"
	"github.com/pkg/errors"
)

func ListVersions(ctx context.Context, token, serviceID, productTierID string) (*openapiclient.ListTierVersionSetsResult, error) {
	ctxWithToken := context.WithValue(ctx, openapiclient.ContextAccessToken, token)

	apiClient := getV1Client()
	res, r, err := apiClient.TierVersionSetApiAPI.TierVersionSetApiListTierVersionSets(
		ctxWithToken,
		serviceID,
		productTierID,
	).Execute()

	err = handleV1Error(err)
	if err != nil {
		return nil, err
	}

	defer r.Body.Close()

	return res, nil
}

func FindLatestVersion(ctx context.Context, token, serviceID, productTierID string) (string, error) {
	ctxWithToken := context.WithValue(ctx, openapiclient.ContextAccessToken, token)

	apiClient := getV1Client()
	res, r, err := apiClient.TierVersionSetApiAPI.TierVersionSetApiListTierVersionSets(
		ctxWithToken,
		serviceID,
		productTierID,
	).Execute()

	err = handleV1Error(err)
	if err != nil {
		return "", err
	}

	defer r.Body.Close()

	if len(res.TierVersionSets) == 0 {
		return "", errors.New("no version found")
	}

	return res.TierVersionSets[0].Version, nil
}

func FindPreferredVersion(ctx context.Context, token, serviceID, productTierID string) (string, error) {
	ctxWithToken := context.WithValue(ctx, openapiclient.ContextAccessToken, token)

	apiClient := getV1Client()
	res, r, err := apiClient.TierVersionSetApiAPI.TierVersionSetApiListTierVersionSets(
		ctxWithToken,
		serviceID,
		productTierID,
	).Execute()

	err = handleV1Error(err)
	if err != nil {
		return "", err
	}

	defer r.Body.Close()

	if len(res.TierVersionSets) == 0 {
		return "", errors.New("no version found")
	}

	for _, versionSet := range res.TierVersionSets {
		if versionSet.Status == "Preferred" {
			return versionSet.Version, nil
		}
	}

	return "", errors.New("no preferred version found")
}

func DescribeLatestVersion(ctx context.Context, token, serviceID, productTierID string) (*openapiclient.TierVersionSet, error) {
	ctxWithToken := context.WithValue(ctx, openapiclient.ContextAccessToken, token)

	apiClient := getV1Client()
	res, r, err := apiClient.TierVersionSetApiAPI.TierVersionSetApiListTierVersionSets(
		ctxWithToken,
		serviceID,
		productTierID,
	).Execute()

	err = handleV1Error(err)
	if err != nil {
		return nil, err
	}

	defer r.Body.Close()

	if len(res.TierVersionSets) == 0 {
		return nil, errors.New("no version found")
	}

	return &res.TierVersionSets[0], nil
}

func DescribeVersionSet(ctx context.Context, token, serviceID, productTierID, version string) (*openapiclient.TierVersionSet, error) {
	ctxWithToken := context.WithValue(ctx, openapiclient.ContextAccessToken, token)

	apiClient := getV1Client()
	res, r, err := apiClient.TierVersionSetApiAPI.TierVersionSetApiDescribeTierVersionSet(
		ctxWithToken,
		serviceID,
		productTierID,
		version,
	).Execute()

	err = handleV1Error(err)
	if err != nil {
		return nil, err
	}

	defer r.Body.Close()
	return res, nil
}

func SetDefaultServicePlan(ctx context.Context, token, serviceID, productTierID, version string) (*openapiclient.TierVersionSet, error) {
	ctxWithToken := context.WithValue(ctx, openapiclient.ContextAccessToken, token)

	apiClient := getV1Client()
	res, r, err := apiClient.TierVersionSetApiAPI.TierVersionSetApiPromoteTierVersionSet(
		ctxWithToken,
		serviceID,
		productTierID,
		version,
	).Execute()

	err = handleV1Error(err)
	if err != nil {
		return nil, err
	}

	defer r.Body.Close()
	return res, nil
}

func UpdateVersionSetName(ctx context.Context, token, serviceID, productTierID, version, newName string) (*openapiclient.TierVersionSet, error) {
	ctxWithToken := context.WithValue(ctx, openapiclient.ContextAccessToken, token)

	apiClient := getV1Client()
	updateRequest := openapiclient.NewUpdateTierVersionSetRequest2(newName)

	res, r, err := apiClient.TierVersionSetApiAPI.TierVersionSetApiUpdateTierVersionSet(
		ctxWithToken,
		serviceID,
		productTierID,
		version,
	).UpdateTierVersionSetRequest2(*updateRequest).Execute()

	err = handleV1Error(err)
	if err != nil {
		return nil, err
	}

	defer r.Body.Close()
	return res, nil
}
