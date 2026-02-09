package dataaccess

import (
	"context"
	"fmt"

	openapiclient "github.com/omnistrate-oss/omnistrate-sdk-go/v1"
)

const (
	NextStepsAfterBuildMsgTemplate = `
Next steps:
- Customize domain name for SaaS offer: check 'omnistrate-ctl create domain' command
- Update the service configuration: check 'omnistrate-ctl build' command`
)

func PrintNextStepsAfterBuildMsg() {
	fmt.Println(NextStepsAfterBuildMsgTemplate)
}

func ListServices(ctx context.Context, token string) (*openapiclient.ListServiceResult, error) {
	ctxWithToken := context.WithValue(ctx, openapiclient.ContextAccessToken, token)

	apiClient := getV1Client()
	resp, r, err := apiClient.ServiceApiAPI.ServiceApiListService(ctxWithToken).Execute()

	err = handleV1Error(err)
	if err != nil {
		return nil, err
	}

	r.Body.Close()
	return resp, nil
}

func DescribeService(ctx context.Context, token, serviceID string) (*openapiclient.DescribeServiceResult, error) {
	ctxWithToken := context.WithValue(ctx, openapiclient.ContextAccessToken, token)

	apiClient := getV1Client()
	resp, r, err := apiClient.ServiceApiAPI.ServiceApiDescribeService(ctxWithToken, serviceID).Execute()

	err = handleV1Error(err)
	if err != nil {
		return nil, err
	}

	r.Body.Close()
	return resp, nil
}

func DeleteService(ctx context.Context, token, serviceID string) error {
	ctxWithToken := context.WithValue(ctx, openapiclient.ContextAccessToken, token)

	apiClient := getV1Client()
	r, err := apiClient.ServiceApiAPI.ServiceApiDeleteService(ctxWithToken, serviceID).Execute()

	err = handleV1Error(err)
	if err != nil {
		return err
	}
	r.Body.Close()

	return nil
}

func BuildServiceFromServicePlanSpec(ctx context.Context, token string, request openapiclient.BuildServiceFromServicePlanSpecRequest2) (*openapiclient.BuildServiceFromServicePlanSpecResult, error) {
	ctxWithToken := context.WithValue(ctx, openapiclient.ContextAccessToken, token)
	apiClient := getV1Client()

	resp, r, err := apiClient.ServiceApiAPI.ServiceApiBuildServiceFromServicePlanSpec(ctxWithToken).
		BuildServiceFromServicePlanSpecRequest2(request).
		Execute()
	defer func() {
		if r != nil {
			_ = r.Body.Close()
		}
	}()
	if err != nil {
		return nil, handleV1Error(err)
	}

	return resp, nil
}

func BuildServiceFromComposeSpec(ctx context.Context, token string, request openapiclient.BuildServiceFromComposeSpecRequest2) (*openapiclient.BuildServiceFromComposeSpecResult, error) {
	ctxWithToken := context.WithValue(ctx, openapiclient.ContextAccessToken, token)
	apiClient := getV1Client()

	resp, r, err := apiClient.ServiceApiAPI.ServiceApiBuildServiceFromComposeSpec(ctxWithToken).
		BuildServiceFromComposeSpecRequest2(request).
		Execute()
	defer func() {
		if r != nil {
			_ = r.Body.Close()
		}
	}()
	if err != nil {
		return nil, handleV1Error(err)
	}

	return resp, nil
}

// CreateService creates a new service
func CreateService(ctx context.Context, token, name, description string) (serviceID string, err error) {
ctxWithToken := context.WithValue(ctx, openapiclient.ContextAccessToken, token)
apiClient := getV1Client()

req := openapiclient.CreateServiceRequest2{
Name: name,
}
if description != "" {
req.Description = &description
}

resp, r, err := apiClient.ServiceApiAPI.ServiceApiCreateService(ctxWithToken).
CreateServiceRequest2(req).
Execute()
defer func() {
if r != nil {
_ = r.Body.Close()
}
}()
if err != nil {
return "", handleV1Error(err)
}

if resp == nil || resp.Id == "" {
return "", fmt.Errorf("empty service ID in response")
}

return resp.Id, nil
}
