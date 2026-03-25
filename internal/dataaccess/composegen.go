package dataaccess

import (
	"context"
	"net/http"

	openapiclient "github.com/omnistrate-oss/omnistrate-sdk-go/v1"
)

func CheckIfContainerImageAccessible(ctx context.Context, token string, imageRegistry, image string, userName, password *string) (res *openapiclient.CheckIfContainerImageAccessibleResult, err error) {
	ctxWithToken := context.WithValue(ctx, openapiclient.ContextAccessToken, token)

	apiClient := getV1Client()
	req := apiClient.ComposeGenApiAPI.ComposeGenApiCheckIfContainerImageAccessible(ctxWithToken).
		Image(image).
		ImageRegistry(imageRegistry)
	if userName != nil {
		req = req.Username(*userName)
	}
	if password != nil {
		req = req.Password(*password)
	}

	var r *http.Response
	res, r, err = req.Execute()
	if err != nil {
		return nil, handleV1Error(err)
	}

	r.Body.Close()
	return
}

func GenerateComposeSpecFromContainerImage(ctx context.Context, token string, request openapiclient.GenerateComposeSpecFromContainerImageRequest2) (res *openapiclient.GenerateComposeSpecFromContainerImageResult, err error) {
	ctxWithToken := context.WithValue(ctx, openapiclient.ContextAccessToken, token)

	apiClient := getV1Client()
	var r *http.Response
	res, r, err = apiClient.ComposeGenApiAPI.ComposeGenApiGenerateComposeSpecFromContainerImage(ctxWithToken).GenerateComposeSpecFromContainerImageRequest2(request).Execute()
	if err != nil {
		return nil, handleV1Error(err)
	}

	r.Body.Close()
	return
}
