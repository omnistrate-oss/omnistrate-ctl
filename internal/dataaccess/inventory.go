package dataaccess

import (
	"context"

	openapiclientfleet "github.com/omnistrate-oss/omnistrate-sdk-go/fleet"
)

func SearchInventory(ctx context.Context, token, query string, filters ...any) (*openapiclientfleet.SearchInventoryResult, error) {
	ctxWithToken := context.WithValue(ctx, openapiclientfleet.ContextAccessToken, token)

	req := newSearchInventoryRequest(query, filters...)

	apiClient := getFleetClient()
	res, r, err := apiClient.InventoryApiAPI.
		InventoryApiSearchInventory(ctxWithToken).
		SearchInventoryRequest2(req).
		Execute()

	err = handleFleetError(err)
	if err != nil {
		return nil, err
	}

	r.Body.Close()
	return res, nil
}

func newSearchInventoryRequest(query string, filters ...any) openapiclientfleet.SearchInventoryRequest2 {
	req := openapiclientfleet.SearchInventoryRequest2{
		Query: query,
	}
	if len(filters) > 0 && filters[0] != nil {
		req.AdditionalProperties = map[string]interface{}{
			"filters": filters[0],
		}
	}
	return req
}
