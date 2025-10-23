package dataaccess

import (
	"context"
	"fmt"
	"net/http"

	"github.com/omnistrate-oss/omnistrate-ctl/internal/utils"

	openapiclientfleet "github.com/omnistrate-oss/omnistrate-sdk-go/fleet"
	openapiclientv1 "github.com/omnistrate-oss/omnistrate-sdk-go/v1"
)

func CreateResourceInstance(ctx context.Context, token string,
	serviceProviderId string, serviceKey string, serviceAPIVersion string, serviceEnvironmentKey string, serviceModelKey string, productTierKey string, resourceKey string,
	request openapiclientfleet.FleetCreateResourceInstanceRequest2) (res *openapiclientfleet.FleetCreateResourceInstanceResult, err error) {
	ctxWithToken := context.WithValue(ctx, openapiclientfleet.ContextAccessToken, token)
	apiClient := getFleetClient()

	req := apiClient.InventoryApiAPI.InventoryApiCreateResourceInstance(
		ctxWithToken,
		serviceProviderId,
		serviceKey,
		serviceAPIVersion,
		serviceEnvironmentKey,
		serviceModelKey,
		productTierKey,
		resourceKey,
	).FleetCreateResourceInstanceRequest2(request)

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

func RestoreResourceInstanceSnapshot(ctx context.Context, token string, serviceID, environmentID, snapshotID string, formattedParams map[string]any, tierVersionOverride string, networkType string) (res *openapiclientfleet.FleetRestoreResourceInstanceResult, err error) {
	ctxWithToken := context.WithValue(ctx, openapiclientfleet.ContextAccessToken, token)
	apiClient := getFleetClient()

	if networkType == "" {
		networkType = "PUBLIC"
	}

	reqBody := openapiclientfleet.FleetRestoreResourceInstanceFromSnapshotRequest2{
		InputParametersOverride: formattedParams,
		NetworkType:             utils.ToPtr(networkType),
	}

	if tierVersionOverride != "" {
		reqBody.ProductTierVersionOverride = &tierVersionOverride
	}

	req := apiClient.InventoryApiAPI.InventoryApiRestoreResourceInstanceFromSnapshot(
		ctxWithToken,
		serviceID,
		environmentID,
		snapshotID,
	).FleetRestoreResourceInstanceFromSnapshotRequest2(reqBody)

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

func DescribeResourceInstanceSnapshot(ctx context.Context, token string, serviceID, environmentID, instanceID, snapshotID string) (res *openapiclientfleet.FleetDescribeInstanceSnapshotResult, err error) {
	ctxWithToken := context.WithValue(ctx, openapiclientfleet.ContextAccessToken, token)
	apiClient := getFleetClient()

	req := apiClient.InventoryApiAPI.InventoryApiDescribeResourceInstanceSnapshot(
		ctxWithToken,
		serviceID,
		environmentID,
		instanceID,
		snapshotID,
	)

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

func ListResourceInstance(ctx context.Context, token string, serviceID, environmentID string, opts ...func(openapiclientfleet.ApiInventoryApiListResourceInstancesRequest) openapiclientfleet.ApiInventoryApiListResourceInstancesRequest) (res *openapiclientfleet.ListFleetResourceInstancesResultInternal, err error) {
	ctxWithToken := context.WithValue(ctx, openapiclientfleet.ContextAccessToken, token)
	apiClient := getFleetClient()

	req := apiClient.InventoryApiAPI.InventoryApiListResourceInstances(
		ctxWithToken,
		serviceID,
		environmentID,
	)

	// Apply optional parameters
	for _, opt := range opts {
		req = opt(req)
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

// WithFilter adds a filter parameter to the ListResourceInstance request
func WithFilter(filter string) func(openapiclientfleet.ApiInventoryApiListResourceInstancesRequest) openapiclientfleet.ApiInventoryApiListResourceInstancesRequest {
	return func(req openapiclientfleet.ApiInventoryApiListResourceInstancesRequest) openapiclientfleet.ApiInventoryApiListResourceInstancesRequest {
		return req.Filter(filter)
	}
}

// WithProductTierId adds a product tier ID parameter
func WithProductTierId(productTierId string) func(openapiclientfleet.ApiInventoryApiListResourceInstancesRequest) openapiclientfleet.ApiInventoryApiListResourceInstancesRequest {
	return func(req openapiclientfleet.ApiInventoryApiListResourceInstancesRequest) openapiclientfleet.ApiInventoryApiListResourceInstancesRequest {
		return req.ProductTierId(productTierId)
	}
}

func ListResourceInstanceSnapshots(ctx context.Context, token string, serviceID, environmentID, instanceID string) (res *openapiclientfleet.FleetListInstanceSnapshotResult, err error) {
	ctxWithToken := context.WithValue(ctx, openapiclientfleet.ContextAccessToken, token)
	apiClient := getFleetClient()

	req := apiClient.InventoryApiAPI.InventoryApiListResourceInstanceSnapshots(
		ctxWithToken,
		serviceID,
		environmentID,
		instanceID,
	)

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

func TriggerResourceInstanceAutoBackup(ctx context.Context, token string, serviceID, environmentID, instanceID string) (res *openapiclientfleet.FleetAutomaticInstanceSnapshotCreationResult, err error) {
	ctxWithToken := context.WithValue(ctx, openapiclientfleet.ContextAccessToken, token)
	apiClient := getFleetClient()

	req := apiClient.InventoryApiAPI.InventoryApiTriggerAutomaticResourceInstanceSnapshotCreation(
		ctxWithToken,
		serviceID,
		environmentID,
		instanceID,
	)

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

func DeleteResourceInstance(ctx context.Context, token, serviceID, environmentID, resourceID, instanceID string) (err error) {
	ctxWithToken := context.WithValue(ctx, openapiclientfleet.ContextAccessToken, token)
	apiClient := getFleetClient()

	req := apiClient.InventoryApiAPI.InventoryApiDeleteResourceInstance(
		ctxWithToken,
		serviceID,
		environmentID,
		instanceID,
	).FleetDeleteResourceInstanceRequest2(openapiclientfleet.FleetDeleteResourceInstanceRequest2{
		ResourceId: resourceID,
	})

	var r *http.Response
	defer func() {
		if r != nil {
			_ = r.Body.Close()
		}
	}()

	r, err = req.Execute()
	if err != nil {
		return handleFleetError(err)
	}
	return
}

func DescribeResourceInstance(ctx context.Context, token string, serviceID, environmentID, instanceID string, detail bool) (resp *openapiclientfleet.ResourceInstance, err error) {
	ctxWithToken := context.WithValue(ctx, openapiclientfleet.ContextAccessToken, token)
	apiClient := getFleetClient()

	req := apiClient.InventoryApiAPI.InventoryApiDescribeResourceInstance(
		ctxWithToken,
		serviceID,
		environmentID,
		instanceID,
	).Detail(detail)

	var r *http.Response
	defer func() {
		if r != nil {
			_ = r.Body.Close()
		}
	}()

	resp, r, err = req.Execute()
	if err != nil {
		return nil, handleFleetError(err)
	}
	return
}

func UpdateResourceInstanceDebugMode(ctx context.Context, token string, serviceID, environmentID, instanceID string, enable bool) (err error) {
	ctxWithToken := context.WithValue(ctx, openapiclientfleet.ContextAccessToken, token)
	apiClient := getFleetClient()

	req := apiClient.InventoryApiAPI.InventoryApiUpdateResourceInstanceDebugMode(
		ctxWithToken,
		serviceID,
		environmentID,
		instanceID,
	).FleetUpdateResourceInstanceDebugModeRequest2(openapiclientfleet.FleetUpdateResourceInstanceDebugModeRequest2{
		Enable: enable,
	})

	var r *http.Response
	defer func() {
		if r != nil {
			_ = r.Body.Close()
		}
	}()

	r, err = req.Execute()
	if err != nil {
		return handleFleetError(err)
	}
	return
}

func DebugResourceInstance(ctx context.Context, token string, serviceID, environmentID, instanceID string) (res *openapiclientfleet.DebugResourceInstanceResult, err error) {
	ctxWithToken := context.WithValue(ctx, openapiclientfleet.ContextAccessToken, token)
	apiClient := getFleetClient()

	req := apiClient.InventoryApiAPI.InventoryApiDebugResourceInstance(
		ctxWithToken,
		serviceID,
		environmentID,
		instanceID,
	)

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

func RestartResourceInstance(ctx context.Context, token string, serviceID, environmentID, resourceID, instanceID string) (err error) {
	ctxWithToken := context.WithValue(ctx, openapiclientfleet.ContextAccessToken, token)
	apiClient := getFleetClient()

	req := apiClient.InventoryApiAPI.InventoryApiRestartResourceInstance(
		ctxWithToken,
		serviceID,
		environmentID,
		instanceID,
	).FleetRestartResourceInstanceRequest2(openapiclientfleet.FleetRestartResourceInstanceRequest2{
		ResourceId: resourceID,
	})

	var r *http.Response
	defer func() {
		if r != nil {
			_ = r.Body.Close()
		}
	}()

	r, err = req.Execute()
	if err != nil {
		return handleFleetError(err)
	}
	return
}

func StartResourceInstance(ctx context.Context, token string, serviceID, environmentID, resourceID, instanceID string) (err error) {
	ctxWithToken := context.WithValue(ctx, openapiclientfleet.ContextAccessToken, token)
	apiClient := getFleetClient()

	req := apiClient.InventoryApiAPI.InventoryApiStartResourceInstance(
		ctxWithToken,
		serviceID,
		environmentID,
		instanceID,
	).FleetStartResourceInstanceRequest2(openapiclientfleet.FleetStartResourceInstanceRequest2{
		ResourceId: resourceID,
	})

	var r *http.Response
	defer func() {
		if r != nil {
			_ = r.Body.Close()
		}
	}()

	r, err = req.Execute()
	if err != nil {
		return handleFleetError(err)
	}
	return
}

func StopResourceInstance(ctx context.Context, token string, serviceID, environmentID, resourceID, instanceID string) (err error) {
	ctxWithToken := context.WithValue(ctx, openapiclientfleet.ContextAccessToken, token)
	apiClient := getFleetClient()

	req := apiClient.InventoryApiAPI.InventoryApiStopResourceInstance(
		ctxWithToken,
		serviceID,
		environmentID,
		instanceID,
	).FleetStopResourceInstanceRequest2(openapiclientfleet.FleetStopResourceInstanceRequest2{
		ResourceId: resourceID,
	})

	var r *http.Response
	defer func() {
		if r != nil {
			_ = r.Body.Close()
		}
	}()

	r, err = req.Execute()
	if err != nil {
		return handleFleetError(err)
	}
	return
}

func UpdateResourceInstance(
	ctx context.Context,
	token string,
	serviceID, environmentID, instanceID string,
	resourceId string,
	networkType *string,
	requestParameters map[string]any,
	customTags []openapiclientfleet.CustomTag,
) (err error) {
	ctxWithToken := context.WithValue(ctx, openapiclientfleet.ContextAccessToken, token)
	apiClient := getFleetClient()

	reqBody := openapiclientfleet.FleetUpdateResourceInstanceRequest2{
		NetworkType:   networkType,
		ResourceId:    resourceId,
		RequestParams: requestParameters,
	}
	if customTags != nil {
		reqBody.CustomTags = customTags
	}

	req := apiClient.InventoryApiAPI.InventoryApiUpdateResourceInstance(
		ctxWithToken,
		serviceID,
		environmentID,
		instanceID,
	).FleetUpdateResourceInstanceRequest2(reqBody)

	var r *http.Response
	defer func() {
		if r != nil {
			_ = r.Body.Close()
		}
	}()

	r, err = req.Execute()
	if err != nil {
		return handleFleetError(err)
	}
	return
}

func AdoptResourceInstance(ctx context.Context, token string, serviceID, servicePlanID, hostClusterID, primaryResourceKey string, request openapiclientfleet.AdoptResourceInstanceRequest2, servicePlanVersion, subscriptionID *string) (res *openapiclientfleet.FleetCreateResourceInstanceResult, err error) {
	ctxWithToken := context.WithValue(ctx, openapiclientfleet.ContextAccessToken, token)
	apiClient := getFleetClient()

	req := apiClient.InventoryApiAPI.InventoryApiAdoptResourceInstance(
		ctxWithToken,
		serviceID,
		servicePlanID,
		hostClusterID,
		primaryResourceKey,
	).AdoptResourceInstanceRequest2(request)

	// Add optional parameters if provided
	if servicePlanVersion != nil && *servicePlanVersion != "" {
		req = req.ServicePlanVersion(*servicePlanVersion)
	}
	if subscriptionID != nil && *subscriptionID != "" {
		req = req.SubscriptionID(*subscriptionID)
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

func OneOffPatchResourceInstance(ctx context.Context, token string, serviceID, environmentID, instanceID string, resourceOverrideConfig map[string]openapiclientfleet.ResourceOneOffPatchConfigurationOverride, targetTierVersion string) (err error) {
	ctxWithToken := context.WithValue(ctx, openapiclientfleet.ContextAccessToken, token)
	apiClient := getFleetClient()

	// Create the request
	request := openapiclientfleet.OneOffPatchResourceInstanceRequest2{
		ResourceOverrideConfiguration: &resourceOverrideConfig,
	}

	// Add target tier version if provided
	if targetTierVersion != "" {
		request.TargetTierVersion = &targetTierVersion
	}

	req := apiClient.InventoryApiAPI.InventoryApiOneOffPatchResourceInstance(
		ctxWithToken,
		serviceID,
		environmentID,
		instanceID,
	).OneOffPatchResourceInstanceRequest2(request)

	var r *http.Response
	defer func() {
		if r != nil {
			_ = r.Body.Close()
		}
	}()

	_, r, err = req.Execute()
	if err != nil {
		return handleFleetError(err)
	}
	return
}

func DescribeResourceInstanceInstaller(ctx context.Context, token string, serviceID, environmentID, instanceID string) (resp *openapiclientfleet.DescribeResourceInstanceInstallerResult, err error) {
	ctxWithToken := context.WithValue(ctx, openapiclientfleet.ContextAccessToken, token)
	apiClient := getFleetClient()

	req := apiClient.InventoryApiAPI.InventoryApiDescribeResourceInstanceInstaller(
		ctxWithToken,
		serviceID,
		environmentID,
		instanceID,
	)

	var r *http.Response
	defer func() {
		if r != nil {
			_ = r.Body.Close()
		}
	}()

	resp, r, err = req.Execute()
	if err != nil {
		return nil, handleFleetError(err)
	}
	return
}

func EvaluateExpression(ctx context.Context, token, serviceID, productTierID, instanceID, resourceKey, expression string, expressionMap map[string]interface{}) (result interface{}, err error) {
	// Validate that either expression or expressionMap is provided
	if expression == "" && expressionMap == nil {
		return nil, fmt.Errorf("either expression or expressionMap is required")
	}

	ctxWithToken := context.WithValue(ctx, openapiclientv1.ContextAccessToken, token)
	apiClient := getV1Client()

	// Create the request
	request := openapiclientv1.ExpressionEvaluatorRequest2{
		ServiceID:   serviceID,
		ResourceKey: resourceKey,
	}

	// Set optional fields
	if productTierID != "" {
		request.ProductTierID = &productTierID
	}

	if instanceID != "" {
		request.InstanceID = &instanceID
	}

	// Set either expression or expressionMap
	if expressionMap != nil {
		request.ExpressionMap = expressionMap
	} else if expression != "" {
		request.Expression = &expression
	}

	req := apiClient.ExpressionEvaluatorApiAPI.ExpressionEvaluatorApiExpressionEvaluator(ctxWithToken).ExpressionEvaluatorRequest2(request)

	var r *http.Response
	defer func() {
		if r != nil {
			_ = r.Body.Close()
		}
	}()

	res, r, err := req.Execute()
	if err != nil {
		return nil, handleV1Error(err)
	}

	// Handle the response
	if res.Error != nil && *res.Error != "" {
		return nil, fmt.Errorf("expression evaluation failed: %s", *res.Error)
	}

	return res.Result, nil
}
