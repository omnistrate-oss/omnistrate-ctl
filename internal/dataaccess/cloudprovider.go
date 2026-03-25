package dataaccess

import (
	"context"

	openapiclient "github.com/omnistrate-oss/omnistrate-sdk-go/v1"
)

func GetCloudProviderByName(ctx context.Context, token string, cloudProvider string) (string, error) {
	ctxWithToken := context.WithValue(ctx, openapiclient.ContextAccessToken, token)

	apiClient := getV1Client()
	res, r, err := apiClient.CloudProviderApiAPI.CloudProviderApiGetCloudProviderByName(
		ctxWithToken,
		cloudProvider,
	).Execute()

	err = handleV1Error(err)
	if err != nil {
		return "", err
	}

	defer r.Body.Close()
	return res, nil
}
