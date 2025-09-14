package dataaccess

import (
	"context"
	"net/http"
	"time"

	openapiclientfleet "github.com/omnistrate-oss/omnistrate-sdk-go/fleet"
)

type DeploymentCellHealthOptions struct {
	HostClusterID        string
	ServiceID            string
	ServiceEnvironmentID string
}

func GetDeploymentCellHealth(ctx context.Context, token string, opts *DeploymentCellHealthOptions) (res *openapiclientfleet.DeploymentCellHealthDetail, err error) {
	ctxWithToken := context.WithValue(ctx, openapiclientfleet.ContextAccessToken, token)
	apiClient := getFleetClient()

	req := apiClient.OperationsApiAPI.OperationsApiDeploymentCellHealth(ctxWithToken)

	if opts != nil {
		if opts.HostClusterID != "" {
			req = req.HostClusterID(opts.HostClusterID)
		}
		if opts.ServiceID != "" {
			req = req.ServiceID(opts.ServiceID)
		}
		if opts.ServiceEnvironmentID != "" {
			req = req.ServiceEnvironmentID(opts.ServiceEnvironmentID)
		}
	}

	var r *http.Response
	defer func() {
		if r != nil {
			_ = r.Body.Close()
		}
	}()

	res, r, err = req.Execute()
	if err != nil {
		return nil, handleFleetError(err)
	}
	return
}

func GetServiceHealth(ctx context.Context, token string, serviceID, serviceEnvironmentID string) (res *openapiclientfleet.ServiceHealthSummary, err error) {
	ctxWithToken := context.WithValue(ctx, openapiclientfleet.ContextAccessToken, token)
	apiClient := getFleetClient()

	req := apiClient.OperationsApiAPI.OperationsApiServiceHealth(ctxWithToken).
		ServiceID(serviceID).
		ServiceEnvironmentID(serviceEnvironmentID)

	var r *http.Response
	defer func() {
		if r != nil {
			_ = r.Body.Close()
		}
	}()

	res, r, err = req.Execute()
	if err != nil {
		return nil, handleFleetError(err)
	}
	return
}

type ListEventsOptions struct {
	NextPageToken        string
	PageSize             *int64
	EnvironmentType      string
	EventTypes           []string
	ServiceID            string
	ServiceEnvironmentID string
	InstanceID           string
	StartDate            *time.Time
	EndDate              *time.Time
	ProductTierID        string
}

func ListOperationsEvents(ctx context.Context, token string, opts *ListEventsOptions) (res *openapiclientfleet.ListServiceProviderEventsResult, err error) {
	ctxWithToken := context.WithValue(ctx, openapiclientfleet.ContextAccessToken, token)
	apiClient := getFleetClient()

	req := apiClient.OperationsApiAPI.OperationsApiListEvents(ctxWithToken)

	if opts != nil {
		if opts.NextPageToken != "" {
			req = req.NextPageToken(opts.NextPageToken)
		}
		if opts.PageSize != nil {
			req = req.PageSize(*opts.PageSize)
		}
		if opts.EnvironmentType != "" {
			req = req.EnvironmentType(opts.EnvironmentType)
		}
		if len(opts.EventTypes) > 0 {
			req = req.EventTypes(opts.EventTypes)
		}
		if opts.ServiceID != "" {
			req = req.ServiceID(opts.ServiceID)
		}
		if opts.ServiceEnvironmentID != "" {
			req = req.ServiceEnvironmentID(opts.ServiceEnvironmentID)
		}
		if opts.InstanceID != "" {
			req = req.InstanceID(opts.InstanceID)
		}
		if opts.StartDate != nil {
			req = req.StartDate(*opts.StartDate)
		}
		if opts.EndDate != nil {
			req = req.EndDate(*opts.EndDate)
		}
		if opts.ProductTierID != "" {
			req = req.ProductTierID(opts.ProductTierID)
		}
	}

	var r *http.Response
	defer func() {
		if r != nil {
			_ = r.Body.Close()
		}
	}()

	res, r, err = req.Execute()
	if err != nil {
		return nil, handleFleetError(err)
	}
	return
}
