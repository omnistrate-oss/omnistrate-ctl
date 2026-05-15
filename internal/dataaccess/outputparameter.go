package dataaccess

import (
	"context"
	"net/http"

	openapiclient "github.com/omnistrate-oss/omnistrate-sdk-go/v1"
)

func ListOutputParameters(ctx context.Context, token, serviceID, resourceID, productTierID, productTierVersion string) (resp *openapiclient.ListOutputParametersResult, err error) {
	ctxWithToken := context.WithValue(ctx, openapiclient.ContextAccessToken, token)
	apiClient := getV1Client()

	req := apiClient.OutputParameterApiAPI.OutputParameterApiListOutputParameter(
		ctxWithToken,
		serviceID,
		resourceID,
	)
	if productTierID != "" {
		req = req.ProductTierId(productTierID)
	}
	if productTierVersion != "" {
		req = req.ProductTierVersion(productTierVersion)
	}

	var r *http.Response
	defer func() {
		if r != nil {
			_ = r.Body.Close()
		}
	}()

	resp, r, err = req.Execute()
	if err != nil {
		return nil, handleV1Error(err)
	}
	return
}
