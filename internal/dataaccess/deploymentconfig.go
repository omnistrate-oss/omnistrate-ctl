package dataaccess

import (
	"context"

	openapiclient "github.com/omnistrate-oss/omnistrate-sdk-go/v1"
)

func GetDefaultDeploymentConfigID(ctx context.Context, token string) (string, error) {
	ctxWithToken := context.WithValue(ctx, openapiclient.ContextAccessToken, token)

	apiClient := getV1Client()
	res, r, err := apiClient.DeploymentConfigApiAPI.DeploymentConfigApiDescribeDeploymentConfig(
		ctxWithToken,
		"default",
	).Execute()

	err = handleV1Error(err)
	if err != nil {
		return "nil", err
	}

	r.Body.Close()
	return res.Id, nil
}
