package dataaccess

import (
	"context"
	"net/http"

	openapiclientfleet "github.com/omnistrate-oss/omnistrate-sdk-go/fleet"
	openapiclient "github.com/omnistrate-oss/omnistrate-sdk-go/v1"
)

func SaveHelmChart(
	ctx context.Context,
	token string,
	chartName string,
	chartVersion string,
	namespace string,
	repoName string,
	repoURL string,
	values map[string]any,
) (
	helmPackage openapiclient.HelmPackage,
	err error,
) {
	ctxWithToken := context.WithValue(ctx, openapiclient.ContextAccessToken, token)

	apiClient := getV1Client()

	helmPackage = openapiclient.HelmPackage{
		ChartName:     chartName,
		ChartVersion:  chartVersion,
		Namespace:     namespace,
		ChartRepoName: repoName,
		ChartRepoUrl:  repoURL,
		ChartValues:   values,
	}

	r, err := apiClient.HelmPackageApiAPI.
		HelmPackageApiSaveHelmPackage(ctxWithToken).
		SaveHelmPackageRequest2(openapiclient.SaveHelmPackageRequest2{
			HelmPackage: helmPackage,
		}).Execute()

	if err != nil {
		return helmPackage, handleV1Error(err)
	}

	r.Body.Close()
	return
}

func ListHelmCharts(ctx context.Context, token string) (helmPackages *openapiclient.ListHelmPackagesResult, err error) {
	ctxWithToken := context.WithValue(ctx, openapiclient.ContextAccessToken, token)

	apiClient := getV1Client()

	var r *http.Response
	helmPackages, r, err = apiClient.HelmPackageApiAPI.HelmPackageApiListHelmPackages(ctxWithToken).Execute()
	if err != nil {
		return nil, handleV1Error(err)
	}

	r.Body.Close()
	return
}

func DescribeHelmChart(ctx context.Context, token, chartName, chartVersion string) (helmPackage *openapiclient.HelmPackage, err error) {
	ctxWithToken := context.WithValue(ctx, openapiclient.ContextAccessToken, token)

	apiClient := getV1Client()

	var r *http.Response
	helmPackage, r, err = apiClient.HelmPackageApiAPI.HelmPackageApiDescribeHelmPackage(ctxWithToken, chartName, chartVersion).Execute()
	if err != nil {
		return nil, handleV1Error(err)
	}

	r.Body.Close()
	return
}

func ListHelmChartInstallations(ctx context.Context, token string, hostClusterID string) (helmPackageInstallations *openapiclientfleet.ListHelmPackageInstallationsResult, err error) {
	ctxWithToken := context.WithValue(ctx, openapiclientfleet.ContextAccessToken, token)

	apiClient := getFleetClient()

	req := apiClient.HelmPackageApiAPI.HelmPackageApiListHelmPackageInstallations(ctxWithToken)
	if len(hostClusterID) > 0 {
		req = req.HostClusterID(hostClusterID)
	}

	var r *http.Response
	helmPackageInstallations, r, err = req.Execute()
	if err != nil {
		return nil, handleFleetError(err)
	}

	r.Body.Close()
	return
}

func DeleteHelmChart(ctx context.Context, token, chartName, chartVersion string) (err error) {
	ctxWithToken := context.WithValue(ctx, openapiclient.ContextAccessToken, token)

	apiClient := getV1Client()
	var r *http.Response
	r, err = apiClient.HelmPackageApiAPI.HelmPackageApiDeleteHelmPackage(ctxWithToken, chartName, chartVersion).Execute()
	if err != nil {
		return handleV1Error(err)
	}

	r.Body.Close()
	return
}
