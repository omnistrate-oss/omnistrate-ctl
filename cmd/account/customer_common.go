package account

import (
	"context"
	"fmt"
	"strings"

	"github.com/omnistrate-oss/omnistrate-ctl/internal/dataaccess"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/utils"
	openapiclientfleet "github.com/omnistrate-oss/omnistrate-sdk-go/fleet"
	openapiclient "github.com/omnistrate-oss/omnistrate-sdk-go/v1"
)

const customerAccountSearchQuery = "resourceinstance:i"

type customerAccountListItem struct {
	InstanceID     string `json:"instance_id"`
	Service        string `json:"service"`
	Environment    string `json:"environment"`
	Plan           string `json:"plan"`
	Version        string `json:"version"`
	Resource       string `json:"resource"`
	CloudProvider  string `json:"cloud_provider"`
	Status         string `json:"status"`
	SubscriptionID string `json:"subscription_id"`
}

type customerAccountSummary struct {
	InstanceID         string `json:"instance_id"`
	AccountConfigID    string `json:"account_config_id,omitempty"`
	AccountName        string `json:"account_name,omitempty"`
	AccountDescription string `json:"account_description,omitempty"`
	TargetAccountID    string `json:"target_account_id,omitempty"`
	Service            string `json:"service"`
	Environment        string `json:"environment"`
	Plan               string `json:"plan"`
	Version            string `json:"version"`
	Resource           string `json:"resource"`
	CloudProvider      string `json:"cloud_provider"`
	Region             string `json:"region"`
	InstanceStatus     string `json:"instance_status"`
	AccountStatus      string `json:"account_status,omitempty"`
	AccountStatusMsg   string `json:"account_status_message,omitempty"`
	SubscriptionID     string `json:"subscription_id"`
}

type customerAccountDescribeResult struct {
	Summary  customerAccountSummary                     `json:"summary"`
	Instance *openapiclientfleet.ResourceInstance       `json:"instance"`
	Account  *openapiclient.DescribeAccountConfigResult `json:"account,omitempty"`
}

type customerAccountDeleteResult struct {
	InstanceID      string `json:"instance_id"`
	AccountConfigID string `json:"account_config_id,omitempty"`
	Service         string `json:"service"`
	Environment     string `json:"environment"`
	Plan            string `json:"plan"`
	Resource        string `json:"resource"`
	CloudProvider   string `json:"cloud_provider"`
	SubscriptionID  string `json:"subscription_id"`
	Deleted         bool   `json:"deleted"`
}

type customerAccountInstanceRef struct {
	InstanceID     string
	ServiceID      string
	ServiceName    string
	EnvironmentID  string
	Environment    string
	PlanID         string
	Plan           string
	Version        string
	ResourceID     string
	Resource       string
	CloudProvider  string
	Region         string
	Status         string
	SubscriptionID string
}

var (
	searchInventoryFn          = dataaccess.SearchInventory
	describeResourceInstanceFn = dataaccess.DescribeResourceInstance
	describeAccountFn          = dataaccess.DescribeAccount
	updateAccountFn            = dataaccess.UpdateAccount
	deleteResourceInstanceFn   = dataaccess.DeleteResourceInstance
)

func isCustomerAccountResourceID(resourceID string) bool {
	return strings.HasPrefix(strings.TrimSpace(resourceID), "r-injectedaccountconfig")
}

func customerAccountListItemFromSearchRecord(record openapiclientfleet.ResourceInstanceSearchRecord) (*customerAccountListItem, bool) {
	ref, ok := customerAccountInstanceRefFromSearchRecord(record)
	if !ok {
		return nil, false
	}

	return &customerAccountListItem{
		InstanceID:     ref.InstanceID,
		Service:        ref.ServiceName,
		Environment:    ref.Environment,
		Plan:           ref.Plan,
		Version:        ref.Version,
		Resource:       ref.Resource,
		CloudProvider:  ref.CloudProvider,
		Status:         ref.Status,
		SubscriptionID: ref.SubscriptionID,
	}, true
}

func customerAccountInstanceRefFromSearchRecord(record openapiclientfleet.ResourceInstanceSearchRecord) (*customerAccountInstanceRef, bool) {
	if record.ResourceId == nil || !isCustomerAccountResourceID(*record.ResourceId) {
		return nil, false
	}

	plan := record.ProductTierId
	if record.ProductTierName != nil && strings.TrimSpace(*record.ProductTierName) != "" {
		plan = strings.TrimSpace(*record.ProductTierName)
	}

	version := ""
	if record.ProductTierVersion != nil {
		version = strings.TrimSpace(*record.ProductTierVersion)
	}

	subscriptionID := ""
	if record.SubscriptionId != nil {
		subscriptionID = strings.TrimSpace(*record.SubscriptionId)
	}

	return &customerAccountInstanceRef{
		InstanceID:     strings.TrimSpace(record.Id),
		ServiceID:      strings.TrimSpace(record.ServiceId),
		ServiceName:    strings.TrimSpace(record.ServiceName),
		EnvironmentID:  strings.TrimSpace(record.ServiceEnvironmentId),
		Environment:    strings.TrimSpace(record.ServiceEnvironmentName),
		PlanID:         strings.TrimSpace(record.ProductTierId),
		Plan:           plan,
		Version:        version,
		ResourceID:     strings.TrimSpace(utils.FromPtr(record.ResourceId)),
		Resource:       strings.TrimSpace(record.ResourceName),
		CloudProvider:  strings.TrimSpace(record.CloudProvider),
		Region:         strings.TrimSpace(record.RegionCode),
		Status:         strings.TrimSpace(record.Status),
		SubscriptionID: subscriptionID,
	}, true
}

func resolveCustomerAccountInstanceByID(ctx context.Context, token string, instanceID string) (*customerAccountInstanceRef, error) {
	searchResult, err := searchInventoryFn(ctx, token, fmt.Sprintf("resourceinstance:%s", strings.TrimSpace(instanceID)))
	if err != nil {
		return nil, err
	}

	for _, record := range searchResult.ResourceInstanceResults {
		if !strings.EqualFold(record.Id, instanceID) {
			continue
		}

		ref, ok := customerAccountInstanceRefFromSearchRecord(record)
		if !ok {
			return nil, fmt.Errorf("%s is not a customer BYOA account onboarding instance", instanceID)
		}
		return ref, nil
	}

	return nil, fmt.Errorf("%s not found. Please check the customer account instance ID and try again", instanceID)
}

func describeCustomerAccountByInstanceRef(
	ctx context.Context,
	token string,
	ref *customerAccountInstanceRef,
) (*customerAccountDescribeResult, error) {
	if ref == nil {
		return nil, fmt.Errorf("customer account instance reference is required")
	}

	instance, err := describeResourceInstanceFn(ctx, token, ref.ServiceID, ref.EnvironmentID, ref.InstanceID)
	if err != nil {
		return nil, err
	}

	accountConfigID := extractCustomerAccountConfigID(instance)

	var account *openapiclient.DescribeAccountConfigResult
	if accountConfigID != "" {
		account, err = describeAccountFn(ctx, token, accountConfigID)
		if err != nil {
			return nil, err
		}
	}

	return &customerAccountDescribeResult{
		Summary:  buildCustomerAccountSummary(ref, accountConfigID, account),
		Instance: instance,
		Account:  account,
	}, nil
}

func buildCustomerAccountSummary(
	ref *customerAccountInstanceRef,
	accountConfigID string,
	account *openapiclient.DescribeAccountConfigResult,
) customerAccountSummary {
	summary := customerAccountSummary{
		InstanceID:      ref.InstanceID,
		AccountConfigID: strings.TrimSpace(accountConfigID),
		Service:         ref.ServiceName,
		Environment:     ref.Environment,
		Plan:            ref.Plan,
		Version:         ref.Version,
		Resource:        ref.Resource,
		CloudProvider:   ref.CloudProvider,
		Region:          ref.Region,
		InstanceStatus:  ref.Status,
		SubscriptionID:  ref.SubscriptionID,
	}

	if account == nil {
		return summary
	}

	summary.AccountName = account.Name
	summary.AccountDescription = account.Description
	summary.AccountStatus = account.Status
	summary.AccountStatusMsg = account.StatusMessage

	formattedAccount, err := formatAccount(account)
	if err == nil {
		summary.TargetAccountID = formattedAccount.TargetAccountID
		if summary.CloudProvider == "" {
			summary.CloudProvider = formattedAccount.CloudProvider
		}
	}

	return summary
}
