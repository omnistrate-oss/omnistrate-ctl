package dataaccess

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/omnistrate-oss/omnistrate-ctl/internal/config"

	"github.com/omnistrate-oss/omnistrate-ctl/internal/utils"
	openapiclient "github.com/omnistrate-oss/omnistrate-sdk-go/v1"
)

func DescribeAccount(ctx context.Context, token string, id string) (*openapiclient.DescribeAccountConfigResult, error) {
	ctxWithToken := context.WithValue(ctx, openapiclient.ContextAccessToken, token)

	apiClient := getV1Client()
	res, r, err := apiClient.AccountConfigApiAPI.AccountConfigApiDescribeAccountConfig(
		ctxWithToken,
		id,
	).Execute()

	err = handleV1Error(err)
	if err != nil {
		return nil, err
	}

	r.Body.Close()
	return res, nil
}

func ListAccounts(ctx context.Context, token string, cloudProvider string) (*openapiclient.ListAccountConfigResult, error) {
	ctxWithToken := context.WithValue(ctx, openapiclient.ContextAccessToken, token)

	apiClient := getV1Client()
	res, r, err := apiClient.AccountConfigApiAPI.AccountConfigApiListAccountConfig(
		ctxWithToken,
		cloudProvider,
	).Execute()

	err = handleV1Error(err)
	if err != nil {
		return nil, err
	}

	r.Body.Close()
	return res, nil
}

func DeleteAccount(ctx context.Context, token, accountConfigID string) error {
	ctxWithToken := context.WithValue(ctx, openapiclient.ContextAccessToken, token)

	apiClient := getV1Client()
	r, err := apiClient.AccountConfigApiAPI.AccountConfigApiDeleteAccountConfig(
		ctxWithToken,
		accountConfigID,
	).Execute()

	err = handleV1Error(err)
	if err != nil {
		return err
	}

	r.Body.Close()
	return nil
}

func CreateAccount(ctx context.Context, token string, accountConfig openapiclient.CreateAccountConfigRequest2) (string, error) {
	ctxWithToken := context.WithValue(ctx, openapiclient.ContextAccessToken, token)

	apiClient := getV1Client()
	res, r, err := apiClient.AccountConfigApiAPI.AccountConfigApiCreateAccountConfig(
		ctxWithToken,
	).CreateAccountConfigRequest2(accountConfig).Execute()

	err = handleV1Error(err)
	if err != nil {
		return "", err
	}

	r.Body.Close()
	return strings.Trim(res, "\"\n"), nil
}

type UpdateAccountParams struct {
	AccountConfigID string
	Name            *string
	Description     *string
	NebiusBindings  []openapiclient.NebiusAccountBindingInput
}

type updateAccountConfigRequestBody struct {
	Name           *string                          `json:"name,omitempty"`
	Description    *string                          `json:"description,omitempty"`
	NebiusBindings []updateNebiusAccountBindingBody `json:"nebiusBindings,omitempty"`
}

type updateNebiusAccountBindingBody struct {
	ProjectID        string `json:"projectID"`
	ServiceAccountID string `json:"serviceAccountID"`
	PublicKeyID      string `json:"publicKeyID"`
	PrivateKeyPEM    string `json:"privateKeyPEM"`
}

func UpdateAccount(ctx context.Context, token string, params UpdateAccountParams) (string, error) {
	httpClient := getRetryableHttpClient()

	body := updateAccountConfigRequestBody{
		Name:        params.Name,
		Description: params.Description,
	}
	if len(params.NebiusBindings) > 0 {
		body.NebiusBindings = make([]updateNebiusAccountBindingBody, 0, len(params.NebiusBindings))
		for _, binding := range params.NebiusBindings {
			body.NebiusBindings = append(body.NebiusBindings, updateNebiusAccountBindingBody{
				ProjectID:        binding.ProjectID,
				ServiceAccountID: binding.ServiceAccountID,
				PublicKeyID:      binding.PublicKeyID,
				PrivateKeyPEM:    binding.PrivateKeyPEM,
			})
		}
	}

	jsonPayload, err := json.Marshal(body)
	if err != nil {
		return "", err
	}

	urlPath := fmt.Sprintf(
		"%s://%s/2022-09-01-00/accountconfig/%s",
		config.GetHostScheme(),
		config.GetHost(),
		params.AccountConfigID,
	)
	request, err := http.NewRequest(http.MethodPut, urlPath, bytes.NewBuffer(jsonPayload))
	if err != nil {
		return "", err
	}
	request = request.WithContext(ctx)
	request.Header.Add("Authorization", token)
	request.Header.Set("Content-Type", "application/json")

	var response *http.Response
	defer func() {
		if response != nil {
			_ = response.Body.Close()
		}
	}()

	response, err = httpClient.Do(request)
	if err != nil {
		return "", err
	}

	if response.StatusCode != http.StatusAccepted {
		return "", handleV1HTTPResponseError(response)
	}

	responseBody, err := io.ReadAll(response.Body)
	if err != nil {
		return "", err
	}

	var accountConfigID string
	if err := json.Unmarshal(responseBody, &accountConfigID); err == nil {
		return accountConfigID, nil
	}

	return strings.TrimSpace(strings.Trim(string(responseBody), "\"")), nil
}

func handleV1HTTPResponseError(response *http.Response) error {
	responseBody, readErr := io.ReadAll(response.Body)
	if readErr != nil {
		return fmt.Errorf("request failed: %s", response.Status)
	}

	var apiError openapiclient.Error
	if err := json.Unmarshal(responseBody, &apiError); err == nil && apiError.Message != "" {
		return fmt.Errorf("%s\nDetail: %s", apiError.Name, apiError.Message)
	}

	trimmed := strings.TrimSpace(string(responseBody))
	if trimmed == "" {
		return fmt.Errorf("request failed: %s", response.Status)
	}

	return fmt.Errorf("request failed: %s\nDetail: %s", response.Status, trimmed)
}

const (
	AccountNotVerifiedWarningMsgTemplateAWS = `
WARNING! Account %s (ID: %s) is not verified. To complete the account configuration setup, follow the instructions below:

For AWS CloudFormation users:
- Create your CloudFormation Stack using the template at: %s
- Watch our setup guide at: %s

For AWS Terraform users:
- Execute the Terraform scripts from: %s
- Use your Account Config ID: %s
- Watch our Terraform guide at: %s`

	AccountNotVerifiedWarningMsgTemplateGCP = `
WARNING! Account %s (Project ID: %s,Project Number: %s) is not verified. To complete the account configuration setup, follow the instructions below:

1. Open Google Cloud Shell at: https://shell.cloud.google.com/?cloudshell_ephemeral=true&show=terminal
2. Execute the following command:
   %s

For guidance, watch our GCP setup guide at: https://youtu.be/7A9WbZjuXgQ`

	AccountNotVerifiedWarningMsgTemplateAzure = `
WARNING! Account %s (Subscription ID: %s, Tenant ID: %s) is not verified. To complete the account configuration setup, follow the instructions below:

1. Open Azure Cloud Shell at: https://portal.azure.com/#cloudshell/
2. Execute the following command:
   %s

For guidance, watch our Azure setup guide at: https://youtu.be/isTGi8tQA2w`

	NextStepVerifyAccountMsgTemplateAWS = `
Next step:
Verify your account.
For AWS CloudFormation users:
- Please create your CloudFormation Stack using the provided template at %s
- Watch the CloudFormation guide at %s for help

For AWS Terraform users:
- Execute the Terraform scripts from: %s
- Use your Account Config ID: %s
- Watch our Terraform guide at %s`

	NextStepVerifyAccountMsgTemplateGCP = `
Next step:
Verify your account.

1. Open Google Cloud Shell at: https://shell.cloud.google.com/?cloudshell_ephemeral=true&show=terminal
2. Execute the following command:
   %s

For guidance, watch our GCP setup guide at: https://youtu.be/7A9WbZjuXgQ`

	NextStepVerifyAccountMsgTemplateAzure = `
Next step:
Verify your account.

1. Open Azure Cloud Shell at: https://portal.azure.com/#cloudshell/
2. Execute the following command:
   %s

For guidance, watch our Azure setup guide at: https://youtu.be/isTGi8tQA2w`

	NextStepVerifyAccountMsgTemplateNebius = `
Next step:
Verify your Nebius account.

- Run 'omnistrate-ctl account describe %s' to inspect the per-region binding status.
- Every Nebius binding should become READY before using this account for deployments.`

	AwsCloudFormationGuideURL = "https://youtu.be/Mu-4jppldwk"
	AwsGcpTerraformScriptsURL = "https://github.com/omnistrate-oss/account-setup"
	AwsGcpTerraformGuideURL   = "https://youtu.be/eKktc4QKgaA"
)

func PrintNextStepVerifyAccountMsg(account *openapiclient.DescribeAccountConfigResult) {
	awsCloudFormationTemplateURL := ""
	if account.AwsCloudFormationTemplateURL != nil {
		awsCloudFormationTemplateURL = *account.AwsCloudFormationTemplateURL
	}

	var nextStepMessage string
	name := account.Name
	if name == "" {
		name = "Unnamed Account"
	}

	// Determine cloud provider and set appropriate message
	if account.AwsAccountID != nil {
		targetAccountID := *account.AwsAccountID
		nextStepMessage = fmt.Sprintf("Account: %s\n%s",
			name,
			fmt.Sprintf(NextStepVerifyAccountMsgTemplateAWS,
				awsCloudFormationTemplateURL, AwsCloudFormationGuideURL,
				AwsGcpTerraformScriptsURL, targetAccountID, AwsGcpTerraformGuideURL))
	} else if account.GcpProjectID != nil && account.GcpBootstrapShellCommand != nil {
		nextStepMessage = fmt.Sprintf("Account: %s\n%s",
			name,
			fmt.Sprintf(NextStepVerifyAccountMsgTemplateGCP,
				*account.GcpBootstrapShellCommand))
	} else if account.AzureSubscriptionID != nil && account.AzureBootstrapShellCommand != nil {
		nextStepMessage = fmt.Sprintf("Account: %s\n%s",
			name,
			fmt.Sprintf(NextStepVerifyAccountMsgTemplateAzure,
				*account.AzureBootstrapShellCommand))
	} else if account.NebiusTenantID != nil {
		nextStepMessage = fmt.Sprintf("Account: %s\n%s",
			name,
			fmt.Sprintf(NextStepVerifyAccountMsgTemplateNebius, account.Id))
	}

	if nextStepMessage != "" {
		fmt.Println(nextStepMessage)
	}
}

func PrintAccountNotVerifiedWarning(account *openapiclient.DescribeAccountConfigResult) {
	awsCloudFormationTemplateURL := ""
	if account.AwsCloudFormationTemplateURL != nil {
		awsCloudFormationTemplateURL = *account.AwsCloudFormationTemplateURL
	}

	var targetAccountID string
	var warningMessage string
	name := account.Name
	if name == "" {
		name = "Unnamed Account"
	}

	// Determine cloud provider and set appropriate message
	if account.AwsAccountID != nil {
		warningMessage = fmt.Sprintf(AccountNotVerifiedWarningMsgTemplateAWS, name, *account.AwsAccountID,
			awsCloudFormationTemplateURL, AwsCloudFormationGuideURL,
			AwsGcpTerraformScriptsURL, targetAccountID, AwsGcpTerraformGuideURL)
	} else if account.GcpProjectID != nil && account.GcpProjectNumber != nil && account.GcpBootstrapShellCommand != nil {
		warningMessage = fmt.Sprintf(AccountNotVerifiedWarningMsgTemplateGCP, name, *account.GcpProjectID,
			*account.GcpProjectNumber, *account.GcpBootstrapShellCommand)
	} else if account.AzureSubscriptionID != nil && account.AzureTenantID != nil && account.AzureBootstrapShellCommand != nil {
		warningMessage = fmt.Sprintf(AccountNotVerifiedWarningMsgTemplateAzure, name, *account.AzureSubscriptionID,
			*account.AzureTenantID, *account.AzureBootstrapShellCommand)
	} else if account.NebiusTenantID != nil {
		warningMessage = fmt.Sprintf(
			"WARNING! Account %s (Nebius Tenant ID: %s) is not verified.\n\n%s",
			name,
			*account.NebiusTenantID,
			formatNebiusBindingStatusSummary(account),
		)
	}

	if warningMessage != "" {
		utils.PrintWarning(warningMessage)
	}
}

func formatNebiusBindingStatusSummary(account *openapiclient.DescribeAccountConfigResult) string {
	if len(account.NebiusBindings) == 0 {
		return "No Nebius bindings are configured. Add at least one Nebius binding and then verify the account."
	}

	lines := []string{
		"Per-region binding status:",
	}
	for _, binding := range account.NebiusBindings {
		status := utils.FromPtr(binding.Status)
		if status == "" {
			status = account.Status
		}
		statusMessage := utils.FromPtr(binding.StatusMessage)

		line := fmt.Sprintf("- region=%s project=%s serviceAccountID=%s publicKeyID=%s status=%s",
			binding.Region,
			binding.ProjectID,
			binding.ServiceAccountID,
			binding.PublicKeyID,
			status,
		)
		if statusMessage != "" {
			line += fmt.Sprintf(" message=%s", statusMessage)
		}
		lines = append(lines, line)
	}

	lines = append(lines, fmt.Sprintf("Run 'omnistrate-ctl account describe %s' for the full Nebius verification details.", account.Id))
	return strings.Join(lines, "\n")
}

func AskVerifyAccountIfAny(ctx context.Context) {
	token, err := config.GetToken()
	if err != nil {
		utils.PrintError(err)
		return
	}

	// List all accounts
	listRes, err := ListAccounts(ctx, token, "all")
	if err != nil {
		utils.PrintError(err)
		return
	}

	// Warn if any accounts are not verified
	for _, account := range listRes.AccountConfigs {
		if account.Status != "READY" {
			PrintAccountNotVerifiedWarning(&account)
		}
	}
}
