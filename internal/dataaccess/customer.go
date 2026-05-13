package dataaccess

import (
	"context"
	"strings"

	openapiclientfleet "github.com/omnistrate-oss/omnistrate-sdk-go/fleet"
)

type CustomerUserCreateRequest struct {
	Email                  string
	Name                   string
	password               string
	LegalCompanyName       string
	CompanyURL             string
	EnableAutoVerification bool
	Attributes             map[string]string
}

type CustomerUserUpdateRequest struct {
	Attributes map[string]string
}

type CustomerUserListOptions struct {
	NextPageToken string
	PageSize      int64
	ExcludeStats  bool
}

func NewCustomerUserCreateRequest(email, name, password, legalCompanyName, companyURL string, enableAutoVerification bool, attributes map[string]string) CustomerUserCreateRequest {
	return CustomerUserCreateRequest{
		Email:                  email,
		Name:                   name,
		password:               password,
		LegalCompanyName:       legalCompanyName,
		CompanyURL:             companyURL,
		EnableAutoVerification: enableAutoVerification,
		Attributes:             attributes,
	}
}

func CreateCustomerUser(ctx context.Context, token string, req CustomerUserCreateRequest) (string, error) {
	ctxWithToken := context.WithValue(ctx, openapiclientfleet.ContextAccessToken, token)
	apiClient := getFleetClient()

	body := openapiclientfleet.FleetCreateConsumptionUserRequest2{
		Email:                  req.Email,
		EnableAutoVerification: req.EnableAutoVerification,
		LegalCompanyName:       req.LegalCompanyName,
		Name:                   req.Name,
		Password:               req.password,
	}
	if strings.TrimSpace(req.CompanyURL) != "" {
		body.CompanyUrl = &req.CompanyURL
	}
	if len(req.Attributes) > 0 {
		body.Attributes = &req.Attributes
	}

	resp, r, err := apiClient.InventoryApiAPI.InventoryApiCreateConsumptionUser(ctxWithToken).
		FleetCreateConsumptionUserRequest2(body).
		Execute()
	defer func() {
		if r != nil {
			_ = r.Body.Close()
		}
	}()
	if err != nil {
		return "", handleFleetError(err)
	}
	return cleanupId(resp), nil
}

func ListCustomerUsers(ctx context.Context, token string, opts CustomerUserListOptions) (*openapiclientfleet.FleetListAllUsersResult, error) {
	ctxWithToken := context.WithValue(ctx, openapiclientfleet.ContextAccessToken, token)
	apiClient := getFleetClient()

	req := apiClient.InventoryApiAPI.InventoryApiListAllUsers(ctxWithToken).
		ExcludeStats(opts.ExcludeStats)
	if opts.NextPageToken != "" {
		req = req.NextPageToken(opts.NextPageToken)
	}
	if opts.PageSize > 0 {
		req = req.PageSize(opts.PageSize)
	}

	resp, r, err := req.Execute()
	defer func() {
		if r != nil {
			_ = r.Body.Close()
		}
	}()
	if err != nil {
		return nil, handleFleetError(err)
	}
	return resp, nil
}

func DescribeCustomerUser(ctx context.Context, token, userID string) (*openapiclientfleet.FleetDescribeUserResult, error) {
	ctxWithToken := context.WithValue(ctx, openapiclientfleet.ContextAccessToken, token)
	apiClient := getFleetClient()

	resp, r, err := apiClient.InventoryApiAPI.InventoryApiDescribeOrgUser(ctxWithToken, userID).Execute()
	defer func() {
		if r != nil {
			_ = r.Body.Close()
		}
	}()
	if err != nil {
		return nil, handleFleetError(err)
	}
	return resp, nil
}

func UpdateCustomerUser(ctx context.Context, token, userID string, req CustomerUserUpdateRequest) error {
	ctxWithToken := context.WithValue(ctx, openapiclientfleet.ContextAccessToken, token)
	apiClient := getFleetClient()

	body := openapiclientfleet.FleetUpdateConsumptionUserRequest2{}
	if len(req.Attributes) > 0 {
		body.Attributes = &req.Attributes
	}

	r, err := apiClient.InventoryApiAPI.InventoryApiUpdateConsumptionUser(ctxWithToken, userID).
		FleetUpdateConsumptionUserRequest2(body).
		Execute()
	defer func() {
		if r != nil {
			_ = r.Body.Close()
		}
	}()
	if err != nil {
		return handleFleetError(err)
	}
	return nil
}

func DeleteCustomerUser(ctx context.Context, token, userID string) error {
	ctxWithToken := context.WithValue(ctx, openapiclientfleet.ContextAccessToken, token)
	apiClient := getFleetClient()

	r, err := apiClient.InventoryApiAPI.InventoryApiDeleteUser(ctxWithToken, userID).Execute()
	defer func() {
		if r != nil {
			_ = r.Body.Close()
		}
	}()
	if err != nil {
		return handleFleetError(err)
	}
	return nil
}

func SendCustomerUserVerificationEmail(ctx context.Context, token, userID string) error {
	ctxWithToken := context.WithValue(ctx, openapiclientfleet.ContextAccessToken, token)
	apiClient := getFleetClient()

	r, err := apiClient.InventoryApiAPI.InventoryApiResendVerificationEmail(ctxWithToken, userID).Execute()
	defer func() {
		if r != nil {
			_ = r.Body.Close()
		}
	}()
	if err != nil {
		return handleFleetError(err)
	}
	return nil
}

func SuspendCustomerUser(ctx context.Context, token, userID string) error {
	ctxWithToken := context.WithValue(ctx, openapiclientfleet.ContextAccessToken, token)
	apiClient := getFleetClient()

	r, err := apiClient.InventoryApiAPI.InventoryApiSuspendUser(ctxWithToken, userID).Execute()
	defer func() {
		if r != nil {
			_ = r.Body.Close()
		}
	}()
	if err != nil {
		return handleFleetError(err)
	}
	return nil
}

func UnsuspendCustomerUser(ctx context.Context, token, userID string) error {
	ctxWithToken := context.WithValue(ctx, openapiclientfleet.ContextAccessToken, token)
	apiClient := getFleetClient()

	r, err := apiClient.InventoryApiAPI.InventoryApiUnsuspendUser(ctxWithToken, userID).Execute()
	defer func() {
		if r != nil {
			_ = r.Body.Close()
		}
	}()
	if err != nil {
		return handleFleetError(err)
	}
	return nil
}
