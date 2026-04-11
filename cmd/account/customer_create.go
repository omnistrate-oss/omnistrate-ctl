package account

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/omnistrate-oss/omnistrate-ctl/cmd/common"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/config"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/dataaccess"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/utils"
	openapiclientfleet "github.com/omnistrate-oss/omnistrate-sdk-go/fleet"
	openapiclient "github.com/omnistrate-oss/omnistrate-sdk-go/v1"
	"github.com/spf13/cobra"
)

const (
	customerCreateExample = `# Onboard a Nebius BYOA account into a service plan
	omnistrate-ctl account customer create \
	  --service=postgres \
	  --environment=prod \
	  --plan=customer-hosted \
	  --customer-email=customer@example.com \
	  --nebius-tenant-id=tenant-xxxx \
	  --nebius-bindings-file=./nebius-bindings.yaml`

	customerAccountResourceName          = "Cloud Provider Account"
	customerAccountResourceKey           = "omnistrateCloudAccountConfig"
	customerAccountResultAccountIDKey    = "cloud_provider_account_config_id"
	customerAccountIacToolName           = "Account Configuration Method"
	customerAccountAWSAccountIDName      = "AWS Account ID"
	customerAccountAWSBootstrapRoleName  = "AWS Bootstrap Role ARN"
	customerAccountGCPProjectIDName      = "GCP Project ID"
	customerAccountGCPProjectNumberName  = "GCP Project Number"
	customerAccountGCPServiceAccountName = "GCP Service Account Email"
	customerAccountAzureSubIDName        = "Azure Subscription ID"
	customerAccountAzureTenantIDName     = "Azure Tenant ID"
	customerAccountNebiusTenantIDName    = "Nebius Tenant ID"
	customerAccountNebiusBindingsName    = "Nebius Bindings"
	customerAccountReadyTimeout          = 10 * time.Minute
	customerAccountReadyPollInterval     = 10 * time.Second
	customerEmailFlag                    = "customer-email"
	customerAccountDefaultVersion        = "preferred"
)

type customerCreateOutput struct {
	InstanceID      string `json:"instanceId"`
	AccountConfigID string `json:"accountConfigId,omitempty"`
	Service         string `json:"service"`
	Environment     string `json:"environment"`
	Plan            string `json:"plan"`
	Version         string `json:"version"`
	Resource        string `json:"resource"`
	CloudProvider   string `json:"cloudProvider"`
	Status          string `json:"status"`
	SubscriptionID  string `json:"subscriptionId,omitempty"`
}

type customerAccountTarget struct {
	ServiceID           string
	ServiceName         string
	ServiceProviderID   string
	ServiceURLKey       string
	EnvironmentID       string
	EnvironmentName     string
	EnvironmentType     string
	EnvironmentKey      string
	ProductTierID       string
	ProductTierName     string
	ProductTierKey      string
	Version             string
	ServiceAPIVersion   string
	ServiceModelKey     string
	ServiceModelType    string
	AccountResourceID   string
	AccountResourceKey  string
	AccountResourceName string
	AccountInputParams  []openapiclient.DescribeInputParameterResult
	SubscriptionID      string
}

var (
	resolveCustomerSubscriptionByEmail = dataaccess.GetSubscriptionByCustomerEmailInEnvironment
	describeCurrentUserFn              = dataaccess.DescribeUser
)

var customerCreateCmd = &cobra.Command{
	Use:          "create --service=[service] --environment=[environment] --plan=[plan] [provider flags]",
	Short:        "Create a customer BYOA account onboarding instance",
	Long:         "This command onboards a customer cloud account into the injected BYOA account-config resource for a specific service plan.",
	Example:      customerCreateExample,
	RunE:         runCustomerCreate,
	SilenceUsage: true,
}

func init() {
	customerCreateCmd.Args = cobra.NoArgs

	addCloudAccountProviderFlags(customerCreateCmd)

	customerCreateCmd.Flags().String("service", "", "Service name or ID")
	customerCreateCmd.Flags().String("environment", "", "Environment name or ID")
	customerCreateCmd.Flags().String("plan", "", "Service plan name or ID")
	customerCreateCmd.Flags().String("subscription-id", "", "Subscription ID to use for the onboarding instance")
	customerCreateCmd.Flags().String(customerEmailFlag, "", "Customer email to onboard on behalf of in production environments")

	_ = customerCreateCmd.MarkFlagRequired("service")
	_ = customerCreateCmd.MarkFlagRequired("environment")
	_ = customerCreateCmd.MarkFlagRequired("plan")
}

func runCustomerCreate(cmd *cobra.Command, args []string) error {
	defer config.CleanupArgsAndFlags(cmd, &args)

	serviceArg, _ := cmd.Flags().GetString("service")
	environmentArg, _ := cmd.Flags().GetString("environment")
	planArg, _ := cmd.Flags().GetString("plan")
	subscriptionID, _ := cmd.Flags().GetString("subscription-id")
	customerEmail, _ := cmd.Flags().GetString(customerEmailFlag)
	output, _ := cmd.Flags().GetString("output")
	skipWait, _ := cmd.Flags().GetBool(skipWaitFlag)

	params, err := cloudAccountParamsFromFlags(cmd, "")
	if err != nil {
		utils.PrintError(err)
		return err
	}

	token, err := common.GetTokenWithLogin()
	if err != nil {
		utils.PrintError(err)
		return err
	}

	var sm utils.SpinnerManager
	var spinner *utils.Spinner
	if output != "json" {
		sm = utils.NewSpinnerManager()
		spinner = sm.AddSpinner("Resolving BYOA account target...")
		sm.Start()
	}

	target, err := resolveCustomerAccountTarget(cmd.Context(), token, serviceArg, environmentArg, planArg)
	if err != nil {
		utils.HandleSpinnerError(spinner, sm, err)
		return err
	}

	utils.HandleSpinnerSuccess(spinner, sm, fmt.Sprintf(
		"Resolved %s / %s / %s (%s)",
		target.ServiceName,
		target.EnvironmentName,
		target.ProductTierName,
		target.Version,
	))
	spinner = nil
	sm = nil

	target.SubscriptionID = strings.TrimSpace(subscriptionID)
	target.SubscriptionID, err = resolveCustomerAccountSubscription(
		cmd.Context(),
		token,
		target,
		subscriptionID,
		customerEmail,
	)
	if err != nil {
		utils.PrintError(err)
		return err
	}

	requestParams, err := buildCustomerAccountRequestParams(cmd.Context(), token, params, target.AccountInputParams)
	if err != nil {
		utils.PrintError(err)
		return err
	}

	if output != "json" {
		sm = utils.NewSpinnerManager()
		spinner = sm.AddSpinner("Creating customer BYOA account onboarding instance...")
		sm.Start()
	}

	request := openapiclientfleet.FleetCreateResourceInstanceRequest2{
		ProductTierVersion: &target.Version,
		RequestParams:      requestParams,
	}
	if target.SubscriptionID != "" {
		request.SubscriptionId = &target.SubscriptionID
	}

	createResult, err := dataaccess.CreateResourceInstance(
		cmd.Context(),
		token,
		target.ServiceProviderID,
		target.ServiceURLKey,
		target.ServiceAPIVersion,
		target.EnvironmentKey,
		target.ServiceModelKey,
		target.ProductTierKey,
		target.AccountResourceKey,
		request,
	)
	if err != nil {
		utils.HandleSpinnerError(spinner, sm, err)
		return err
	}
	if createResult == nil || createResult.Id == nil || strings.TrimSpace(*createResult.Id) == "" {
		err = fmt.Errorf("customer account onboarding returned an empty resource instance ID")
		utils.HandleSpinnerError(spinner, sm, err)
		return err
	}

	instanceID := strings.TrimSpace(*createResult.Id)
	utils.HandleSpinnerSuccess(spinner, sm, "Created customer BYOA account onboarding instance")

	if !skipWait {
		var waitSpinner *utils.Spinner
		if output != "json" {
			fmt.Printf("\n")
			sm = utils.NewSpinnerManager()
			waitSpinner = sm.AddSpinner("Waiting for BYOA account onboarding to become READY...")
			sm.Start()
		}

		if err = waitForCustomerAccountInstanceReady(cmd.Context(), token, target.ServiceID, target.EnvironmentID, instanceID); err != nil {
			utils.HandleSpinnerError(waitSpinner, sm, err)
			utils.PrintError(fmt.Errorf("customer account onboarding did not become READY: %v", err))
			return err
		}

		utils.HandleSpinnerSuccess(waitSpinner, sm, "BYOA account onboarding is now READY")
	}

	instanceDetail, err := dataaccess.DescribeResourceInstance(cmd.Context(), token, target.ServiceID, target.EnvironmentID, instanceID)
	if err != nil {
		utils.PrintError(err)
		return err
	}

	accountConfigID := extractCustomerAccountConfigID(instanceDetail)

	var backingAccount *openapiclient.DescribeAccountConfigResult
	if accountConfigID != "" {
		backingAccount, err = dataaccess.DescribeAccount(cmd.Context(), token, accountConfigID)
		if err != nil && output != "json" {
			utils.PrintWarning(fmt.Sprintf("failed to describe backing account config %s: %v", accountConfigID, err))
		}
	}

	createOutput := buildCustomerCreateOutput(target, params, instanceID, accountConfigID, instanceDetail)
	if err = utils.PrintTextTableJsonOutput(output, createOutput); err != nil {
		utils.PrintError(err)
		return err
	}

	if output != "json" && backingAccount != nil && backingAccount.Status != "READY" {
		dataaccess.PrintNextStepVerifyAccountMsg(backingAccount)
	}

	return nil
}

func resolveCustomerAccountTarget(
	ctx context.Context,
	token string,
	serviceArg string,
	environmentArg string,
	planArg string,
) (*customerAccountTarget, error) {
	serviceID, serviceName, err := resolveServiceByNameOrID(ctx, token, serviceArg)
	if err != nil {
		return nil, err
	}

	environment, err := resolveServiceEnvironmentByNameOrID(ctx, token, serviceID, environmentArg)
	if err != nil {
		return nil, err
	}

	plan, err := resolveServicePlanByNameOrID(ctx, token, serviceID, environment.Id, planArg)
	if err != nil {
		return nil, err
	}

	version, err := resolveCustomerAccountVersion(ctx, token, serviceID, plan.ProductTierId, customerAccountDefaultVersion)
	if err != nil {
		return nil, err
	}

	offeringResult, err := dataaccess.ExternalDescribeServiceOffering(ctx, token, serviceID, environment.Id, plan.ProductTierId)
	if err != nil {
		return nil, err
	}

	offering, err := findCustomerAccountOffering(offeringResult, environment.Id, plan.ProductTierId)
	if err != nil {
		return nil, err
	}

	if !strings.EqualFold(offering.ServiceModelType, "BYOA") {
		return nil, fmt.Errorf(
			"service plan %q in environment %q is not a BYOA model (got %q)",
			offering.ProductTierName,
			offering.ServiceEnvironmentName,
			offering.ServiceModelType,
		)
	}

	accountResource, err := findCustomerAccountResource(offering.ResourceParameters)
	if err != nil {
		return nil, err
	}

	return &customerAccountTarget{
		ServiceID:           serviceID,
		ServiceName:         serviceName,
		ServiceProviderID:   offeringResult.ServiceProviderId,
		ServiceURLKey:       offeringResult.ServiceURLKey,
		EnvironmentID:       environment.Id,
		EnvironmentName:     environment.Name,
		EnvironmentType:     environment.Type,
		EnvironmentKey:      offering.ServiceEnvironmentURLKey,
		ProductTierID:       plan.ProductTierId,
		ProductTierName:     plan.ProductTierName,
		ProductTierKey:      offering.ProductTierURLKey,
		Version:             version,
		ServiceAPIVersion:   offering.ServiceAPIVersion,
		ServiceModelKey:     offering.ServiceModelURLKey,
		ServiceModelType:    offering.ServiceModelType,
		AccountResourceID:   accountResource.ResourceId,
		AccountResourceKey:  accountResource.UrlKey,
		AccountResourceName: accountResource.Name,
		AccountInputParams:  customerAccountInputParameters(),
	}, nil
}

func resolveServiceByNameOrID(ctx context.Context, token string, serviceArg string) (string, string, error) {
	services, err := dataaccess.ListServices(ctx, token)
	if err != nil {
		return "", "", err
	}

	matches := make([]openapiclient.DescribeServiceResult, 0)
	for _, service := range services.Services {
		if service.Id == serviceArg || strings.EqualFold(service.Name, serviceArg) {
			matches = append(matches, service)
		}
	}

	switch len(matches) {
	case 0:
		return "", "", fmt.Errorf("service %q not found", serviceArg)
	case 1:
		return matches[0].Id, matches[0].Name, nil
	default:
		return "", "", fmt.Errorf("multiple services matched %q; use a service ID", serviceArg)
	}
}

func resolveServiceEnvironmentByNameOrID(
	ctx context.Context,
	token string,
	serviceID string,
	environmentArg string,
) (*openapiclient.DescribeServiceEnvironmentResult, error) {
	environments, err := dataaccess.ListServiceEnvironments(ctx, token, serviceID)
	if err != nil {
		return nil, err
	}

	matches := make([]*openapiclient.DescribeServiceEnvironmentResult, 0)
	for _, environmentID := range environments.Ids {
		environment, describeErr := dataaccess.DescribeServiceEnvironment(ctx, token, serviceID, environmentID)
		if describeErr != nil {
			return nil, describeErr
		}
		if environment.Id == environmentArg || strings.EqualFold(environment.Name, environmentArg) {
			matches = append(matches, environment)
		}
	}

	switch len(matches) {
	case 0:
		return nil, fmt.Errorf("environment %q not found for service %s", environmentArg, serviceID)
	case 1:
		return matches[0], nil
	default:
		return nil, fmt.Errorf("multiple environments matched %q; use an environment ID", environmentArg)
	}
}

func resolveServicePlanByNameOrID(
	ctx context.Context,
	token string,
	serviceID string,
	environmentID string,
	planArg string,
) (*openapiclient.GetServicePlanResult, error) {
	plans, err := dataaccess.ListServicePlans(ctx, token, serviceID, environmentID)
	if err != nil {
		return nil, err
	}

	matches := make([]openapiclient.GetServicePlanResult, 0)
	for _, plan := range plans.ServicePlans {
		if plan.ProductTierId == planArg || strings.EqualFold(plan.ProductTierName, planArg) {
			matches = append(matches, plan)
		}
	}

	switch len(matches) {
	case 0:
		return nil, fmt.Errorf("service plan %q not found in environment %s", planArg, environmentID)
	case 1:
		return &matches[0], nil
	default:
		return nil, fmt.Errorf("multiple service plans matched %q; use a product tier ID", planArg)
	}
}

func resolveCustomerAccountVersion(
	ctx context.Context,
	token string,
	serviceID string,
	productTierID string,
	versionArg string,
) (string, error) {
	version := strings.TrimSpace(versionArg)
	var err error

	switch version {
	case "", "preferred":
		version, err = dataaccess.FindPreferredVersion(ctx, token, serviceID, productTierID)
	case "latest":
		version, err = dataaccess.FindLatestVersion(ctx, token, serviceID, productTierID)
	default:
		_, err = dataaccess.DescribeVersionSet(ctx, token, serviceID, productTierID, version)
	}
	if err != nil {
		return "", err
	}

	return version, nil
}

func findCustomerAccountOffering(
	offeringResult *openapiclient.DescribeServiceOfferingResult,
	environmentID string,
	productTierID string,
) (*openapiclient.ServiceOffering, error) {
	if offeringResult == nil {
		return nil, fmt.Errorf("service offering response is empty")
	}

	for _, offering := range offeringResult.Offerings {
		if offering.ServiceEnvironmentID == environmentID && offering.ProductTierID == productTierID {
			return &offering, nil
		}
	}

	return nil, fmt.Errorf(
		"service offering did not expose environment %s and product tier %s",
		environmentID,
		productTierID,
	)
}

func findCustomerAccountResource(resources []openapiclient.ResourceEntity) (*openapiclient.ResourceEntity, error) {
	for _, resource := range resources {
		if resource.UrlKey == customerAccountResourceKey || strings.HasPrefix(resource.ResourceId, "r-injectedaccountconfig") {
			return &resource, nil
		}
	}

	return nil, fmt.Errorf(
		"selected service plan does not expose the injected %q resource required for BYOA account onboarding",
		customerAccountResourceName,
	)
}

func customerAccountInputParameters() []openapiclient.DescribeInputParameterResult {
	return []openapiclient.DescribeInputParameterResult{
		{Name: customerAccountIacToolName, Key: "account_configuration_method"},
		{Name: customerAccountAWSAccountIDName, Key: "aws_account_id"},
		{Name: customerAccountAWSBootstrapRoleName, Key: "aws_bootstrap_role_arn"},
		{Name: customerAccountGCPProjectIDName, Key: "gcp_project_id"},
		{Name: customerAccountGCPProjectNumberName, Key: "gcp_project_number"},
		{Name: customerAccountGCPServiceAccountName, Key: "gcp_service_account_email"},
		{Name: customerAccountAzureSubIDName, Key: "azure_subscription_id"},
		{Name: customerAccountAzureTenantIDName, Key: "azure_tenant_id"},
		{Name: customerAccountNebiusTenantIDName, Key: "nebius_tenant_id"},
		{Name: customerAccountNebiusBindingsName, Key: "nebius_bindings"},
	}
}

func resolveCustomerAccountSubscription(
	ctx context.Context,
	token string,
	target *customerAccountTarget,
	requestedSubscriptionID string,
	requestedCustomerEmail string,
) (string, error) {
	if target == nil {
		return "", fmt.Errorf("customer account target is required")
	}

	subscriptionID := strings.TrimSpace(requestedSubscriptionID)
	customerEmail := strings.TrimSpace(requestedCustomerEmail)

	if customerEmail != "" && subscriptionID != "" {
		return "", fmt.Errorf("cannot specify both --customer-email and --subscription-id")
	}
	if customerEmail != "" {
		if err := utils.ValidateEmail(customerEmail); err != nil {
			return "", fmt.Errorf("invalid --customer-email value: %w", err)
		}
	}

	if isProductionEnvironmentType(target.EnvironmentType) && customerEmail == "" && subscriptionID == "" {
		currentUser, err := describeCurrentUserFn(ctx, token)
		if err != nil {
			return "", fmt.Errorf("failed to resolve the calling user for production subscription lookup: %w", err)
		}

		if currentUser == nil || currentUser.Email == nil || strings.TrimSpace(*currentUser.Email) == "" {
			return "", fmt.Errorf("current user email is unavailable for production subscription lookup")
		}

		customerEmail = strings.TrimSpace(*currentUser.Email)
	}

	if customerEmail == "" {
		return subscriptionID, nil
	}

	subscription, err := resolveCustomerSubscriptionByEmail(
		ctx,
		token,
		target.ServiceID,
		target.EnvironmentID,
		target.ProductTierID,
		customerEmail,
	)
	if err != nil {
		return "", fmt.Errorf(
			"failed to resolve subscription for customer %s in %s/%s: %w",
			customerEmail,
			target.ServiceName,
			target.ProductTierName,
			err,
		)
	}
	if subscription == nil || strings.TrimSpace(subscription.Id) == "" {
		return "", fmt.Errorf("subscription lookup for customer %s returned an empty subscription ID", customerEmail)
	}

	return strings.TrimSpace(subscription.Id), nil
}

func isProductionEnvironmentType(environmentType string) bool {
	switch strings.ToUpper(strings.TrimSpace(environmentType)) {
	case "PROD", "PRODUCTION":
		return true
	default:
		return false
	}
}

func buildCustomerAccountRequestParams(
	ctx context.Context,
	token string,
	params CloudAccountParams,
	inputParameters []openapiclient.DescribeInputParameterResult,
) (map[string]any, error) {
	gcpServiceAccountEmail := ""
	if params.GcpProjectID != "" {
		user, err := dataaccess.DescribeUser(ctx, token)
		if err != nil {
			return nil, err
		}
		if user.OrgId == nil || strings.TrimSpace(*user.OrgId) == "" {
			return nil, fmt.Errorf("describe user returned an empty org ID; cannot derive the GCP bootstrap service account email")
		}

		gcpServiceAccountEmail = fmt.Sprintf("bootstrap-%s@%s.iam.gserviceaccount.com", *user.OrgId, params.GcpProjectID)
	}

	return buildCustomerAccountRequestParamsWithDerivedValues(params, inputParameters, gcpServiceAccountEmail)
}

func buildCustomerAccountRequestParamsWithDerivedValues(
	params CloudAccountParams,
	inputParameters []openapiclient.DescribeInputParameterResult,
	gcpServiceAccountEmail string,
) (map[string]any, error) {
	keysByName := make(map[string]string, len(inputParameters))
	for _, inputParameter := range inputParameters {
		keysByName[inputParameter.Name] = inputParameter.Key
	}

	requestParams := make(map[string]any)
	setParam := func(displayName string, value any) error {
		key, exists := keysByName[displayName]
		if !exists || strings.TrimSpace(key) == "" {
			return fmt.Errorf("BYOA account-config resource is missing input parameter %q", displayName)
		}
		requestParams[key] = value
		return nil
	}

	switch requestedCloudProvider(params) {
	case "aws":
		if err := setParam(customerAccountIacToolName, "CloudFormation"); err != nil {
			return nil, err
		}
		if err := setParam(customerAccountAWSAccountIDName, params.AwsAccountID); err != nil {
			return nil, err
		}
		if err := setParam(
			customerAccountAWSBootstrapRoleName,
			fmt.Sprintf("arn:aws:iam::%s:role/omnistrate-bootstrap-role", params.AwsAccountID),
		); err != nil {
			return nil, err
		}
	case "gcp":
		if err := setParam(customerAccountIacToolName, "Terraform"); err != nil {
			return nil, err
		}
		if err := setParam(customerAccountGCPProjectIDName, params.GcpProjectID); err != nil {
			return nil, err
		}
		if err := setParam(customerAccountGCPProjectNumberName, params.GcpProjectNumber); err != nil {
			return nil, err
		}
		if err := setParam(customerAccountGCPServiceAccountName, gcpServiceAccountEmail); err != nil {
			return nil, err
		}
	case "azure":
		if err := setParam(customerAccountIacToolName, "AzureScript"); err != nil {
			return nil, err
		}
		if err := setParam(customerAccountAzureSubIDName, params.AzureSubscriptionID); err != nil {
			return nil, err
		}
		if err := setParam(customerAccountAzureTenantIDName, params.AzureTenantID); err != nil {
			return nil, err
		}
	case "nebius":
		if err := setParam(customerAccountNebiusTenantIDName, params.NebiusTenantID); err != nil {
			return nil, err
		}
		if err := setParam(customerAccountNebiusBindingsName, toCustomerNebiusBindingParams(params.NebiusBindings)); err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("unsupported cloud provider request")
	}

	return requestParams, nil
}

func toCustomerNebiusBindingParams(bindings []openapiclient.NebiusAccountBindingInput) []map[string]any {
	result := make([]map[string]any, 0, len(bindings))
	for _, binding := range bindings {
		result = append(result, map[string]any{
			"projectID":        binding.ProjectID,
			"serviceAccountID": binding.ServiceAccountID,
			"publicKeyID":      binding.PublicKeyID,
			"privateKeyPEM":    binding.PrivateKeyPEM,
		})
	}
	return result
}

func requestedCloudProvider(params CloudAccountParams) string {
	switch {
	case params.AwsAccountID != "":
		return "aws"
	case params.GcpProjectID != "":
		return "gcp"
	case params.AzureSubscriptionID != "":
		return "azure"
	case params.NebiusTenantID != "":
		return "nebius"
	default:
		return ""
	}
}

func waitForCustomerAccountInstanceReady(
	ctx context.Context,
	token string,
	serviceID string,
	environmentID string,
	instanceID string,
) error {
	timeout := time.After(customerAccountReadyTimeout)
	ticker := time.NewTicker(customerAccountReadyPollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timeout:
			fmt.Fprintf(
				os.Stderr,
				"\nWarning: customer account onboarding %s did not become READY after %s. Check the instance with 'omnistrate-ctl instance describe %s'\n",
				instanceID,
				customerAccountReadyTimeout,
				instanceID,
			)
			return fmt.Errorf("resource instance %s did not become READY after %s", instanceID, customerAccountReadyTimeout)
		case <-ticker.C:
			instance, err := dataaccess.DescribeResourceInstance(ctx, token, serviceID, environmentID, instanceID)
			if err != nil {
				return err
			}

			status := strings.ToUpper(utils.FromPtr(instance.ConsumptionResourceInstanceResult.Status))
			switch status {
			case "READY":
				return nil
			case "FAILED":
				return fmt.Errorf("resource instance %s entered FAILED state", instanceID)
			}
		}
	}
}

func extractCustomerAccountConfigID(instance *openapiclientfleet.ResourceInstance) string {
	if instance == nil {
		return ""
	}

	resultParamsMap, ok := instance.ConsumptionResourceInstanceResult.ResultParams.(map[string]any)
	if !ok {
		return ""
	}

	value, exists := resultParamsMap[customerAccountResultAccountIDKey]
	if !exists {
		return ""
	}

	accountConfigID, ok := value.(string)
	if !ok {
		return ""
	}

	return strings.TrimSpace(accountConfigID)
}

func buildCustomerCreateOutput(
	target *customerAccountTarget,
	params CloudAccountParams,
	instanceID string,
	accountConfigID string,
	instanceDetail *openapiclientfleet.ResourceInstance,
) customerCreateOutput {
	status := ""
	if instanceDetail != nil {
		status = utils.FromPtr(instanceDetail.ConsumptionResourceInstanceResult.Status)
	}

	return customerCreateOutput{
		InstanceID:      instanceID,
		AccountConfigID: accountConfigID,
		Service:         target.ServiceName,
		Environment:     target.EnvironmentName,
		Plan:            target.ProductTierName,
		Version:         target.Version,
		Resource:        target.AccountResourceName,
		CloudProvider:   requestedCloudProvider(params),
		Status:          status,
		SubscriptionID:  target.SubscriptionID,
	}
}
