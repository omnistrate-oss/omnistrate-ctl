package dataaccess

import (
	"context"

	openapiclient "github.com/omnistrate-oss/omnistrate-sdk-go/v1"
)

func DescribeUser(ctx context.Context, token string) (*openapiclient.DescribeUserResult, error) {
	ctxWithToken := context.WithValue(ctx, openapiclient.ContextAccessToken, token)

	apiClient := getV1Client()
	resp, r, err := apiClient.UsersApiAPI.UsersApiDescribeUser(ctxWithToken).Execute()

	err = handleV1Error(err)
	if err != nil {
		return nil, err
	}

	r.Body.Close()
	return resp, nil
}
