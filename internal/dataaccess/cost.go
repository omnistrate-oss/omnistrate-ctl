package dataaccess

import (
	"context"
	"net/http"
	"time"

	openapiclientfleet "github.com/omnistrate-oss/omnistrate-sdk-go/fleet"
)

type CostOptions struct {
	StartDate                time.Time
	EndDate                  time.Time
	EnvironmentType          string
	Frequency                string
	IncludeCloudProviderIDs  *string
	ExcludeCloudProviderIDs  *string
	IncludeRegionIDs         *string
	ExcludeRegionIDs         *string
	IncludeDeploymentCellIDs *string
	ExcludeDeploymentCellIDs *string
	IncludeInstanceIDs       *string
	ExcludeInstanceIDs       *string
	IncludeUserIDs           *string
	ExcludeUserIDs           *string
	TopNInstances            *int64
	TopNUsers                *int64
}

func DescribeCloudProviderCost(ctx context.Context, token string, opts CostOptions) (res *openapiclientfleet.DescribeCloudProviderCostResult, err error) {
	ctxWithToken := context.WithValue(ctx, openapiclientfleet.ContextAccessToken, token)
	apiClient := getFleetClient()

	req := apiClient.CostApiAPI.CostApiDescribeCloudProviderCost(ctxWithToken).
		StartDate(opts.StartDate).
		EndDate(opts.EndDate).
		EnvironmentType(opts.EnvironmentType).
		Frequency(opts.Frequency)

	if opts.IncludeCloudProviderIDs != nil {
		req = req.IncludeCloudProviderIDs(*opts.IncludeCloudProviderIDs)
	}
	if opts.ExcludeCloudProviderIDs != nil {
		req = req.ExcludeCloudProviderIDs(*opts.ExcludeCloudProviderIDs)
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

func DescribeDeploymentCellCost(ctx context.Context, token string, opts CostOptions) (res *openapiclientfleet.DescribeDeploymentCellCostResult, err error) {
	ctxWithToken := context.WithValue(ctx, openapiclientfleet.ContextAccessToken, token)
	apiClient := getFleetClient()

	req := apiClient.CostApiAPI.CostApiDescribeDeploymentCellCost(ctxWithToken).
		StartDate(opts.StartDate).
		EndDate(opts.EndDate).
		EnvironmentType(opts.EnvironmentType).
		Frequency(opts.Frequency)

	if opts.IncludeCloudProviderIDs != nil {
		req = req.IncludeCloudProviderIDs(*opts.IncludeCloudProviderIDs)
	}
	if opts.ExcludeCloudProviderIDs != nil {
		req = req.ExcludeCloudProviderIDs(*opts.ExcludeCloudProviderIDs)
	}
	if opts.IncludeDeploymentCellIDs != nil {
		req = req.IncludeDeploymentCellIDs(*opts.IncludeDeploymentCellIDs)
	}
	if opts.ExcludeDeploymentCellIDs != nil {
		req = req.ExcludeDeploymentCellIDs(*opts.ExcludeDeploymentCellIDs)
	}
	if opts.IncludeInstanceIDs != nil {
		req = req.IncludeInstanceIDs(*opts.IncludeInstanceIDs)
	}
	if opts.ExcludeInstanceIDs != nil {
		req = req.ExcludeInstanceIDs(*opts.ExcludeInstanceIDs)
	}
	if opts.TopNInstances != nil {
		req = req.TopNInstances(*opts.TopNInstances)
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

func DescribeRegionCost(ctx context.Context, token string, opts CostOptions) (res *openapiclientfleet.DescribeRegionCostResult, err error) {
	ctxWithToken := context.WithValue(ctx, openapiclientfleet.ContextAccessToken, token)
	apiClient := getFleetClient()

	req := apiClient.CostApiAPI.CostApiDescribeRegionCost(ctxWithToken).
		StartDate(opts.StartDate).
		EndDate(opts.EndDate).
		EnvironmentType(opts.EnvironmentType).
		Frequency(opts.Frequency)

	if opts.IncludeCloudProviderIDs != nil {
		req = req.IncludeCloudProviderIDs(*opts.IncludeCloudProviderIDs)
	}
	if opts.ExcludeCloudProviderIDs != nil {
		req = req.ExcludeCloudProviderIDs(*opts.ExcludeCloudProviderIDs)
	}
	if opts.IncludeRegionIDs != nil {
		req = req.IncludeRegionIDs(*opts.IncludeRegionIDs)
	}
	if opts.ExcludeRegionIDs != nil {
		req = req.ExcludeRegionIDs(*opts.ExcludeRegionIDs)
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

func DescribeUserCost(ctx context.Context, token string, opts CostOptions) (res *openapiclientfleet.DescribeUserCostResult, err error) {
	ctxWithToken := context.WithValue(ctx, openapiclientfleet.ContextAccessToken, token)
	apiClient := getFleetClient()

	req := apiClient.CostApiAPI.CostApiDescribeUserCost(ctxWithToken).
		StartDate(opts.StartDate).
		EndDate(opts.EndDate).
		EnvironmentType(opts.EnvironmentType)

	if opts.IncludeUserIDs != nil {
		req = req.IncludeUserIDs(*opts.IncludeUserIDs)
	}
	if opts.ExcludeUserIDs != nil {
		req = req.ExcludeUserIDs(*opts.ExcludeUserIDs)
	}
	if opts.TopNUsers != nil {
		req = req.TopNUsers(*opts.TopNUsers)
	}
	if opts.TopNInstances != nil {
		req = req.TopNInstances(*opts.TopNInstances)
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
