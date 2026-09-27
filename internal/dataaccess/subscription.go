package dataaccess

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/omnistrate-oss/omnistrate-ctl/internal/utils"
	openapiclientfleet "github.com/omnistrate-oss/omnistrate-sdk-go/fleet"
	"github.com/pkg/errors"
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

// CustomerSubscriptionLookup identifies the customer and the plan a subscription is being
// resolved for. EnvironmentType decides whether the org-scoped email fallback applies,
// because org-scoped customer identities only exist in production environments.
type CustomerSubscriptionLookup struct {
	ServiceID       string
	EnvironmentID   string
	EnvironmentType string
	PlanID          string
	CustomerEmail   string
}

func GetSubscriptionByCustomerEmail(ctx context.Context, token string, serviceID string, planID string, customerEmail string) (resp *openapiclientfleet.FleetDescribeSubscriptionResult, err error) {
	// Describe the service offering for this service and plan (product tier) ID to get the environment ID
	serviceOfferingResult, err := DescribeServiceOffering(ctx, token, serviceID, planID, "")
	if err != nil {
		return nil, err
	}

	for _, offering := range serviceOfferingResult.ConsumptionDescribeServiceOfferingResult.Offerings {
		if offering.ProductTierID == planID {
			return GetSubscriptionByCustomerEmailInEnvironment(ctx, token, CustomerSubscriptionLookup{
				ServiceID:       serviceID,
				EnvironmentID:   offering.ServiceEnvironmentID,
				EnvironmentType: offering.ServiceEnvironmentType,
				PlanID:          planID,
				CustomerEmail:   customerEmail,
			})
		}
	}

	err = errors.New("no subscription found for the given customer email or the plan does not exist")
	return
}

// GetSubscriptionByCustomerEmailInEnvironment resolves the subscription a customer owns for
// a plan, creating one when the customer has none.
//
// The listing deliberately leaves IncludeInactive off. Terminating a subscription
// soft-deletes its row, and only the inactive listing returns soft-deleted rows, so a
// terminated subscription is correctly invisible here and the customer can be resubscribed.
// A suspended subscription stays live and is reported rather than silently duplicated.
func GetSubscriptionByCustomerEmailInEnvironment(
	ctx context.Context,
	token string,
	lookup CustomerSubscriptionLookup,
) (resp *openapiclientfleet.FleetDescribeSubscriptionResult, err error) {
	customerEmail := strings.TrimSpace(lookup.CustomerEmail)
	if customerEmail == "" {
		return nil, errors.New("customer email is required to look up a subscription")
	}

	subscriptions, err := ListAllSubscriptions(ctx, token, lookup.ServiceID, lookup.EnvironmentID, &ListSubscriptionsOptions{
		ProductTierId: &lookup.PlanID,
	})
	if err != nil {
		return nil, err
	}

	candidates := matchSubscriptionsByEmail(subscriptions, lookup.PlanID, customerEmail)

	// In production the platform stores an end customer's identity scoped to the service
	// provider's organization, so a customer with no match under the plain address may still
	// own a subscription under <local>+<orgID>@<domain>.
	if len(candidates) == 0 &&
		utils.IsProductionEnvironmentType(lookup.EnvironmentType) &&
		!utils.EmailHasScopedOrg(customerEmail) {
		var scopedEmail string
		if scopedEmail, err = scopedCustomerEmail(ctx, token, customerEmail); err != nil {
			return nil, err
		}
		candidates = matchSubscriptionsByEmail(subscriptions, lookup.PlanID, scopedEmail)
	}

	selected, err := selectCustomerSubscription(candidates, customerEmail)
	if err != nil {
		return nil, err
	}
	if selected != nil {
		return selected, nil
	}

	createResp, err := CreateSubscriptionOnBehalf(ctx, token, lookup.ServiceID, lookup.EnvironmentID, &CreateSubscriptionOnBehalfOptions{
		ProductTierID:           lookup.PlanID,
		OnBehalfOfCustomerEmail: customerEmail,
	})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to create subscription for user %s", customerEmail)
	}

	subscriptionID := strings.TrimSpace(createResp.GetId())
	if subscriptionID == "" {
		return nil, errors.Errorf("subscription creation for user %s returned an empty subscription ID", customerEmail)
	}

	resp, err = DescribeSubscription(ctx, token, lookup.ServiceID, lookup.EnvironmentID, subscriptionID)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to describe newly created subscription for user %s", customerEmail)
	}
	if resp == nil {
		return nil, errors.Errorf("describing newly created subscription for user %s returned an empty response", customerEmail)
	}

	// A newly created subscription goes through the same status gate, so every "unusable
	// subscription" message in this command comes from one place.
	if _, err = selectCustomerSubscription([]openapiclientfleet.FleetDescribeSubscriptionResult{*resp}, customerEmail); err != nil {
		return nil, err
	}

	return resp, nil
}

// scopedCustomerEmail builds the org-scoped form of a customer address using the calling
// service provider's organization ID.
func scopedCustomerEmail(ctx context.Context, token, customerEmail string) (string, error) {
	user, err := DescribeUser(ctx, token)
	if err != nil {
		return "", errors.Wrap(err, "failed to describe the current user to resolve the provider organization ID")
	}
	if user == nil || user.OrgId == nil || strings.TrimSpace(*user.OrgId) == "" {
		return "", errors.New("describe user returned an empty organization ID; cannot check for an org-scoped customer subscription")
	}

	scopedEmail, err := utils.FormatEmailWithScopedOrg(customerEmail, strings.TrimSpace(*user.OrgId))
	if err != nil {
		return "", errors.Wrapf(err, "failed to build the org-scoped form of %s", customerEmail)
	}
	return scopedEmail, nil
}

type ListSubscriptionsOptions struct {
	ProductTierId   *string
	IncludeInactive *bool
	ExcludePricing  *bool
	ExcludeStats    *bool
	NextPageToken   string
	PageSize        *int64
}

func ListSubscriptions(ctx context.Context, token string, serviceID, environmentID string) (resp *openapiclientfleet.FleetListSubscriptionsResult, err error) {
	return ListSubscriptionsWithOptions(ctx, token, serviceID, environmentID, nil)
}

func ListSubscriptionsWithOptions(ctx context.Context, token string, serviceID, environmentID string, opts *ListSubscriptionsOptions) (resp *openapiclientfleet.FleetListSubscriptionsResult, err error) {
	ctxWithToken := context.WithValue(ctx, openapiclientfleet.ContextAccessToken, token)
	apiClient := getFleetClient()

	req := apiClient.InventoryApiAPI.InventoryApiListSubscription(
		ctxWithToken,
		serviceID,
		environmentID,
	)

	if opts != nil {
		if opts.ProductTierId != nil {
			req = req.ProductTierId(*opts.ProductTierId)
		}
		if opts.IncludeInactive != nil {
			req = req.IncludeInactive(*opts.IncludeInactive)
		}
		if opts.ExcludePricing != nil {
			req = req.ExcludePricing(*opts.ExcludePricing)
		}
		if opts.ExcludeStats != nil {
			req = req.ExcludeStats(*opts.ExcludeStats)
		}
		if opts.NextPageToken != "" {
			req = req.NextPageToken(opts.NextPageToken)
		}
		if opts.PageSize != nil {
			req = req.PageSize(*opts.PageSize)
		}
	}

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

func ListAllSubscriptions(ctx context.Context, token string, serviceID, environmentID string, opts *ListSubscriptionsOptions) (subscriptions []openapiclientfleet.FleetDescribeSubscriptionResult, err error) {
	var nextPageToken string
	seen := map[string]struct{}{}

	for {
		pageOptions := ListSubscriptionsOptions{}
		if opts != nil {
			pageOptions = *opts
		}
		pageOptions.NextPageToken = nextPageToken

		res, err := ListSubscriptionsWithOptions(ctx, token, serviceID, environmentID, &pageOptions)
		if err != nil {
			return nil, err
		}
		subscriptions = append(subscriptions, res.GetSubscriptions()...)

		nextPageToken = res.GetNextPageToken()
		if nextPageToken == "" {
			return subscriptions, nil
		}
		// A server that echoes back a page token it already handed out would otherwise send
		// this into an infinite loop, re-accumulating the same page forever. Returning what
		// was collected so far would be worse than failing: a caller resolving a customer
		// email would read the truncated list as "no subscription" and create a duplicate,
		// which is the very bug this lookup exists to prevent.
		if _, alreadySeen := seen[nextPageToken]; alreadySeen {
			return nil, errors.Errorf(
				"subscription listing for service %s environment %s repeated page token %q; refusing to return a partial list",
				serviceID,
				environmentID,
				nextPageToken,
			)
		}
		seen[nextPageToken] = struct{}{}
	}
}

type ListUsersOptions struct {
	NextPageToken  string
	PageSize       *int64
	SubscriptionId *string
	ExcludeStats   *bool
}

func ListUsers(ctx context.Context, token string, serviceID, environmentID string, opts *ListUsersOptions) (resp *openapiclientfleet.FleetListUsersResult, err error) {
	ctxWithToken := context.WithValue(ctx, openapiclientfleet.ContextAccessToken, token)
	apiClient := getFleetClient()

	req := apiClient.InventoryApiAPI.InventoryApiListUsers(
		ctxWithToken,
		serviceID,
		environmentID,
	)

	if opts != nil {
		if opts.NextPageToken != "" {
			req = req.NextPageToken(opts.NextPageToken)
		}
		if opts.PageSize != nil {
			req = req.PageSize(*opts.PageSize)
		}
		if opts.SubscriptionId != nil {
			req = req.SubscriptionId(*opts.SubscriptionId)
		}
		if opts.ExcludeStats != nil {
			req = req.ExcludeStats(*opts.ExcludeStats)
		}
	}

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

func ListAllUsers(ctx context.Context, token string, serviceID, environmentID string, opts *ListUsersOptions) (users []openapiclientfleet.User, err error) {
	var nextPageToken string

	for {
		pageOptions := ListUsersOptions{}
		if opts != nil {
			pageOptions = *opts
		}
		pageOptions.NextPageToken = nextPageToken

		res, err := ListUsers(ctx, token, serviceID, environmentID, &pageOptions)
		if err != nil {
			return nil, err
		}
		users = append(users, res.GetUsers()...)

		nextPageToken = res.GetNextPageToken()
		if nextPageToken == "" {
			return users, nil
		}
	}
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
		listUsersRes, r, listErr := apiClient.InventoryApiAPI.InventoryApiListAllUsers(ctxWithToken).Execute()
		if r != nil {
			_ = r.Body.Close()
		}
		if listErr != nil {
			return nil, handleFleetError(errors.Wrap(listErr, "failed to list users"))
		}

		customerUserID = matchUserIDByEmail(listUsersRes.Users, opts.OnBehalfOfCustomerEmail)
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

// subscriptionStatusActive is the only subscription status that may back a new instance.
// The platform's full vocabulary is ACTIVE, SUSPENDED, CANCELLED and TERMINATED.
const subscriptionStatusActive = "ACTIVE"

const subscriptionStatusSuspended = "SUSPENDED"

// matchUserIDByEmail returns the ID of the user with the given address, or "" when there
// is none. The fleet users API reports unscoped addresses, so a plain case-insensitive
// comparison is the correct match.
func matchUserIDByEmail(users []openapiclientfleet.AccessSideUser, email string) string {
	for _, user := range users {
		if user.Email == nil || user.UserId == nil {
			continue
		}
		if strings.EqualFold(*user.Email, email) {
			return *user.UserId
		}
	}
	return ""
}

// matchSubscriptionsByEmail returns the subscriptions on a plan whose root user is the
// given address. The plan filter is redundant with the server-side query parameter and is
// kept so the matching is correct on its own terms.
func matchSubscriptionsByEmail(
	subscriptions []openapiclientfleet.FleetDescribeSubscriptionResult,
	planID string,
	email string,
) []openapiclientfleet.FleetDescribeSubscriptionResult {
	matches := make([]openapiclientfleet.FleetDescribeSubscriptionResult, 0, 1)
	for _, subscription := range subscriptions {
		if planID != "" && !strings.EqualFold(subscription.ProductTierId, planID) {
			continue
		}
		if !strings.EqualFold(subscription.RootUserEmail, email) {
			continue
		}
		matches = append(matches, subscription)
	}
	return matches
}

// selectCustomerSubscription picks the one subscription that may back a new instance.
// It returns (nil, nil) when there is nothing to choose from, which tells the caller to
// create a subscription instead.
func selectCustomerSubscription(
	candidates []openapiclientfleet.FleetDescribeSubscriptionResult,
	customerEmail string,
) (*openapiclientfleet.FleetDescribeSubscriptionResult, error) {
	if len(candidates) == 0 {
		return nil, nil
	}

	active := make([]openapiclientfleet.FleetDescribeSubscriptionResult, 0, len(candidates))
	for _, candidate := range candidates {
		if strings.EqualFold(candidate.Status, subscriptionStatusActive) {
			active = append(active, candidate)
		}
	}

	switch len(active) {
	case 1:
		selected := active[0]
		return &selected, nil
	case 0:
		return nil, unusableSubscriptionError(candidates, customerEmail)
	default:
		ids := make([]string, 0, len(active))
		for _, candidate := range active {
			ids = append(ids, candidate.Id)
		}
		return nil, errors.Errorf(
			"found %d active subscriptions for customer %s on plan %s (%s). Pass --subscription-id to choose one",
			len(active),
			customerEmail,
			active[0].ProductTierName,
			strings.Join(ids, ", "),
		)
	}
}

// unusableSubscriptionError explains why the subscriptions a customer does own cannot back
// a new instance, and names the remedy when there is one.
func unusableSubscriptionError(
	candidates []openapiclientfleet.FleetDescribeSubscriptionResult,
	customerEmail string,
) error {
	if len(candidates) == 1 {
		candidate := candidates[0]
		if strings.EqualFold(candidate.Status, subscriptionStatusSuspended) {
			return errors.Errorf(
				"subscription %s for customer %s on plan %s is SUSPENDED and cannot be used to create an instance. "+
					"Resume it with 'omnistrate-ctl subscription resume %s', or pass --subscription-id to use a different subscription",
				candidate.Id,
				customerEmail,
				candidate.ProductTierName,
				candidate.Id,
			)
		}
		return errors.Errorf(
			"subscription %s for customer %s on plan %s is in status %s, expected ACTIVE, so it cannot be used to create an instance. "+
				"Pass --subscription-id to use a different subscription",
			candidate.Id,
			customerEmail,
			candidate.ProductTierName,
			candidate.Status,
		)
	}

	described := make([]string, 0, len(candidates))
	for _, candidate := range candidates {
		described = append(described, fmt.Sprintf("%s (%s)", candidate.Id, candidate.Status))
	}

	hasSuspended := false
	for _, candidate := range candidates {
		if strings.EqualFold(candidate.Status, subscriptionStatusSuspended) {
			hasSuspended = true
			break
		}
	}

	msg := fmt.Sprintf(
		"customer %s has no active subscription on plan %s; found %s",
		customerEmail,
		candidates[0].ProductTierName,
		strings.Join(described, ", "),
	)

	if hasSuspended {
		msg += ". Resume a suspended subscription with 'omnistrate-ctl subscription resume', or pass --subscription-id"
	} else {
		msg += ". Pass --subscription-id"
	}

	return errors.New(msg)
}
