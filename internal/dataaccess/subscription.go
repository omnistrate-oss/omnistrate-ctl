package dataaccess

import (
	"context"
	"github.com/pkg/errors"
	"net/http"

	openapiclientfleet "github.com/omnistrate-oss/omnistrate-sdk-go/fleet"
)

func DescribeSubscription(ctx context.Context, token string, serviceID, environmentID, instanceID string) (resp *openapiclientfleet.FleetDescribeSubscriptionResult, err error) {
	ctxWithToken := context.WithValue(ctx, openapiclientfleet.ContextAccessToken, token)
	apiClient := getFleetClient()

	req := apiClient.InventoryApiAPI.InventoryApiDescribeSubscription(
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

func GetSubscriptionByCustomerEmail(ctx context.Context, token string, serviceID string, planID string, customerEmail string) (resp *openapiclientfleet.FleetDescribeSubscriptionResult, err error) {
	ctxWithToken := context.WithValue(ctx, openapiclientfleet.ContextAccessToken, token)
	apiClient := getFleetClient()

	// Describe the service offering for this service and plan (product tier) ID to get the environment ID
	serviceOfferingResult, err := DescribeServiceOffering(ctx, token, serviceID, planID, "")
	if err != nil {
		return nil, handleFleetError(err)
	}

	var r *http.Response
	defer func() {
		if r != nil {
			_ = r.Body.Close()
		}
	}()

	for _, offering := range serviceOfferingResult.ConsumptionDescribeServiceOfferingResult.Offerings {
		if offering.ProductTierID == planID {
			req := apiClient.InventoryApiAPI.InventoryApiListSubscription(
				ctxWithToken,
				serviceID,
				offering.ServiceEnvironmentID,
			).ProductTierId(planID)

			var listSubscriptionResult *openapiclientfleet.FleetListSubscriptionsResult
			listSubscriptionResult, r, err = req.Execute()
			if err != nil {
				return nil, handleFleetError(err)
			}

			for _, subscription := range listSubscriptionResult.Subscriptions {
				if subscription.RootUserEmail == customerEmail {
					resp = &subscription
					return
				}
			}

			// Search user by email
			listUsersRes, r, err := apiClient.InventoryApiAPI.InventoryApiListAllUsers(ctxWithToken).Execute()
			if err != nil {
				return nil, handleFleetError(errors.Wrap(err, "failed to list users"))
			}
			_ = r.Body.Close()

			userID := ""
			for _, user := range listUsersRes.Users {
				if *user.Email == customerEmail {
					userID = *user.UserId
					break
				}
			}

			if userID == "" {
				return nil, errors.Errorf("no user found with email %s", customerEmail)
			}

			// Subscription not found for the given customer email, create a new one
			createReq := apiClient.InventoryApiAPI.InventoryApiCreateSubscriptionOnBehalfOfCustomer(
				ctxWithToken,
				serviceID,
				offering.ServiceEnvironmentID,
			).FleetCreateSubscriptionOnBehalfOfCustomerRequest2(openapiclientfleet.FleetCreateSubscriptionOnBehalfOfCustomerRequest2{
				ProductTierId:            planID,
				OnBehalfOfCustomerUserId: userID,
			})

			createResp, r, err := createReq.Execute()
			if err != nil {
				return nil, handleFleetError(errors.Wrapf(err, "failed to create subscription for user %s", customerEmail))
			}
			_ = r.Body.Close()

			// Describe the newly created subscription
			resp, err = DescribeSubscription(ctx, token, serviceID, offering.ServiceEnvironmentID, *createResp.Id)
			if err != nil {
				return nil, handleFleetError(errors.Wrapf(err, "failed to describe newly created subscription for user %s", customerEmail))
			}

			return resp, nil
		}
	}

	err = errors.New("no subscription found for the given customer email or the plan does not exist")
	return
}

func ListSubscriptions(ctx context.Context, token string, serviceID, environmentID string) (resp *openapiclientfleet.FleetListSubscriptionsResult, err error) {
	ctxWithToken := context.WithValue(ctx, openapiclientfleet.ContextAccessToken, token)
	apiClient := getFleetClient()

	req := apiClient.InventoryApiAPI.InventoryApiListSubscription(
		ctxWithToken,
		serviceID,
		environmentID,
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

func ListSubscriptionRequests(ctx context.Context, token string, serviceID, environmentID string) (resp *openapiclientfleet.ListSubscriptionRequestsResult, err error) {
	ctxWithToken := context.WithValue(ctx, openapiclientfleet.ContextAccessToken, token)
	apiClient := getFleetClient()

	req := apiClient.InventoryApiAPI.InventoryApiListSubscriptionRequests(
		ctxWithToken,
		serviceID,
		environmentID,
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

func DescribeSubscriptionRequest(ctx context.Context, token string, serviceID, environmentID, requestID string) (resp *openapiclientfleet.DescribeSubscriptionRequestResult, err error) {
	ctxWithToken := context.WithValue(ctx, openapiclientfleet.ContextAccessToken, token)
	apiClient := getFleetClient()

	req := apiClient.InventoryApiAPI.InventoryApiDescribeSubscriptionRequest(
		ctxWithToken,
		serviceID,
		environmentID,
		requestID,
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

func ApproveSubscriptionRequest(ctx context.Context, token string, serviceID, environmentID, requestID string) (err error) {
	ctxWithToken := context.WithValue(ctx, openapiclientfleet.ContextAccessToken, token)
	apiClient := getFleetClient()

	req := apiClient.InventoryApiAPI.InventoryApiApproveSubscriptionRequest(
		ctxWithToken,
		serviceID,
		environmentID,
		requestID,
	)

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

func DenySubscriptionRequest(ctx context.Context, token string, serviceID, environmentID, requestID string) (err error) {
	ctxWithToken := context.WithValue(ctx, openapiclientfleet.ContextAccessToken, token)
	apiClient := getFleetClient()

	req := apiClient.InventoryApiAPI.InventoryApiDenySubscriptionRequest(
		ctxWithToken,
		serviceID,
		environmentID,
		requestID,
	)

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

type CreateSubscriptionOnBehalfOptions struct {
	ProductTierID                        string
	OnBehalfOfCustomerUserID             string
	OnBehalfOfCustomerEmail              string
	AllowCreatesWhenPaymentNotConfigured *bool
	BillingProvider                      string
	CustomPrice                          *bool
	CustomPricePerUnit                   map[string]interface{}
	ExternalPayerID                      string
	MaxNumberOfInstances                 *int64
	PriceEffectiveDate                   string
}

func CreateSubscriptionOnBehalf(ctx context.Context, token string, serviceID, environmentID string, opts *CreateSubscriptionOnBehalfOptions) (resp *openapiclientfleet.FleetCreateSubscriptionOnBehalfOfCustomerResult, err error) {
	ctxWithToken := context.WithValue(ctx, openapiclientfleet.ContextAccessToken, token)
	apiClient := getFleetClient()

	// If email is provided instead of user ID, resolve it to user ID
	customerUserID := opts.OnBehalfOfCustomerUserID
	if customerUserID == "" && opts.OnBehalfOfCustomerEmail != "" {
		listUsersRes, r, err := apiClient.InventoryApiAPI.InventoryApiListAllUsers(ctxWithToken).Execute()
		if err != nil {
			if r != nil {
				_ = r.Body.Close()
			}
			return nil, handleFleetError(errors.Wrap(err, "failed to list users"))
		}
		if r != nil {
			_ = r.Body.Close()
		}

		for _, user := range listUsersRes.Users {
			if user.Email != nil && *user.Email == opts.OnBehalfOfCustomerEmail {
				customerUserID = *user.UserId
				break
			}
		}

		if customerUserID == "" {
			return nil, errors.Errorf("no user found with email %s", opts.OnBehalfOfCustomerEmail)
		}
	}

	requestBody := openapiclientfleet.FleetCreateSubscriptionOnBehalfOfCustomerRequest2{
		ProductTierId:            opts.ProductTierID,
		OnBehalfOfCustomerUserId: customerUserID,
	}

	if opts.AllowCreatesWhenPaymentNotConfigured != nil {
		requestBody.AllowCreatesWhenPaymentNotConfigured = opts.AllowCreatesWhenPaymentNotConfigured
	}
	if opts.BillingProvider != "" {
		requestBody.BillingProvider = &opts.BillingProvider
	}
	if opts.CustomPrice != nil {
		requestBody.CustomPrice = opts.CustomPrice
	}
	if opts.CustomPricePerUnit != nil {
		requestBody.CustomPricePerUnit = opts.CustomPricePerUnit
	}
	if opts.ExternalPayerID != "" {
		requestBody.ExternalPayerId = &opts.ExternalPayerID
	}
	if opts.MaxNumberOfInstances != nil {
		requestBody.MaxNumberOfInstances = opts.MaxNumberOfInstances
	}
	if opts.PriceEffectiveDate != "" {
		requestBody.PriceEffectiveDate = &opts.PriceEffectiveDate
	}

	req := apiClient.InventoryApiAPI.InventoryApiCreateSubscriptionOnBehalfOfCustomer(
		ctxWithToken,
		serviceID,
		environmentID,
	).FleetCreateSubscriptionOnBehalfOfCustomerRequest2(requestBody)

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

func SuspendSubscription(ctx context.Context, token string, serviceID, environmentID, subscriptionID string) (err error) {
	ctxWithToken := context.WithValue(ctx, openapiclientfleet.ContextAccessToken, token)
	apiClient := getFleetClient()

	req := apiClient.InventoryApiAPI.InventoryApiSuspendSubscription(
		ctxWithToken,
		serviceID,
		environmentID,
		subscriptionID,
	)

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

func ResumeSubscription(ctx context.Context, token string, serviceID, environmentID, subscriptionID string) (err error) {
	ctxWithToken := context.WithValue(ctx, openapiclientfleet.ContextAccessToken, token)
	apiClient := getFleetClient()

	req := apiClient.InventoryApiAPI.InventoryApiResumeSubscription(
		ctxWithToken,
		serviceID,
		environmentID,
		subscriptionID,
	)

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

func TerminateSubscription(ctx context.Context, token string, serviceID, environmentID, subscriptionID string) (err error) {
	ctxWithToken := context.WithValue(ctx, openapiclientfleet.ContextAccessToken, token)
	apiClient := getFleetClient()

	req := apiClient.InventoryApiAPI.InventoryApiTerminateSubscription(
		ctxWithToken,
		serviceID,
		environmentID,
		subscriptionID,
	)

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
