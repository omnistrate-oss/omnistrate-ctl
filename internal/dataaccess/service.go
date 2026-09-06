package dataaccess

import (
	"context"
	"fmt"
	"net/http"

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

func PrepareServiceFromServicePlanSpec(ctx context.Context, token string, request openapiclient.PrepareServiceFromServicePlanSpecRequest2) (*openapiclient.PrepareServiceFromServicePlanSpecResult, error) {
	ctxWithToken := context.WithValue(ctx, openapiclient.ContextAccessToken, token)
	apiClient := getV1Client()

	resp, r, err := apiClient.ServiceApiAPI.ServiceApiPrepareServiceFromServicePlanSpec(ctxWithToken).
		PrepareServiceFromServicePlanSpecRequest2(request).
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

// ValidationSpecTypeServicePlan and ValidationSpecTypeCompose are the only two
// values the validation endpoint accepts for ValidateServiceSpecRequest2.SpecType.
const (
	ValidationSpecTypeServicePlan = "service-plan"
	ValidationSpecTypeCompose     = "compose"
)

// ValidationStatus values returned by the read-only validation endpoint.
const (
	ValidationStatusValid      = "VALID"
	ValidationStatusInvalid    = "INVALID"
	ValidationStatusIncomplete = "INCOMPLETE"
)

// ValidationArtifactEncoding is the single permitted encoding for artifact
// content supplied to the validation endpoint.
const ValidationArtifactEncoding = "tar+gzip+base64"

// ValidateServiceSpecError wraps a failed ValidateServiceSpec call and preserves
// the HTTP status code, which handleV1Error otherwise discards. The build
// --dry-run route needs the status to distinguish "this server has no read-only
// validation endpoint" (404/405) from a genuine authentication or transport
// failure, without ever falling back to the legacy build endpoints.
type ValidateServiceSpecError struct {
	// StatusCode is the HTTP status, or 0 when the request never got a response.
	StatusCode int
	Err        error
}

func (e *ValidateServiceSpecError) Error() string {
	if e == nil || e.Err == nil {
		return "validate service spec failed"
	}
	return e.Err.Error()
}

func (e *ValidateServiceSpecError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

// ServerLacksValidationEndpoint reports whether the server answered in a way
// that means the read-only validation route is not deployed there.
func (e *ValidateServiceSpecError) ServerLacksValidationEndpoint() bool {
	if e == nil {
		return false
	}
	return e.StatusCode == http.StatusNotFound || e.StatusCode == http.StatusMethodNotAllowed
}

// ValidateServiceSpec performs one read-only specification validation request.
// It creates, updates, releases, promotes and publishes nothing: the endpoint is
// the only server interaction the build --dry-run route is permitted to make.
//
// The request body carries specification bytes, compose configs/secrets and
// local artifact archives, so it is registered as a sensitive-body route in
// client.go and is never dumped into debug logs.
func ValidateServiceSpec(ctx context.Context, token string, request openapiclient.ValidateServiceSpecRequest2) (*openapiclient.ValidateServiceSpecResult, error) {
	ctxWithToken := context.WithValue(ctx, openapiclient.ContextAccessToken, token)
	apiClient := getV1Client()

	resp, r, err := apiClient.ServiceApiAPI.ServiceApiValidateServiceSpec(ctxWithToken).
		ValidateServiceSpecRequest2(request).
		Execute()
	defer func() {
		if r != nil {
			_ = r.Body.Close()
		}
	}()
	if err != nil {
		statusCode := 0
		if r != nil {
			statusCode = r.StatusCode
		}
		return nil, &ValidateServiceSpecError{StatusCode: statusCode, Err: handleV1Error(err)}
	}
	if resp == nil {
		statusCode := 0
		if r != nil {
			statusCode = r.StatusCode
		}
		return nil, &ValidateServiceSpecError{
			StatusCode: statusCode,
			Err:        fmt.Errorf("empty response from the validation endpoint"),
		}
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
