package dataaccess

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/omnistrate-oss/omnistrate-ctl/internal/dataaccess"

	openapiclient "github.com/omnistrate-oss/omnistrate-sdk-go/v1"
)

func captureOutput(f func()) string {
	// Capture stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	f()

	// Restore stdout
	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, err := io.Copy(&buf, r)
	if err != nil {
		panic(err)
	}
	return buf.String()
}

func ptr[T any](v T) *T {
	return &v
}

func TestPrintNextStepVerifyAccountMsg(t *testing.T) {
	tests := []struct {
		name     string
		account  *openapiclient.DescribeAccountConfigResult
		expected []string
	}{
		{
			name: "AWS account",
			account: &openapiclient.DescribeAccountConfigResult{
				Name:                         "test-aws",
				AwsAccountID:                 ptr("123456789012"),
				AwsCloudFormationTemplateURL: ptr("https://example.com/template.yml"),
			},
			expected: []string{
				"Next step:",
				"Verify your account",
				"For AWS CloudFormation users:",
				"Please create your CloudFormation Stack using the provided template at https://example.com/template.yml",
				"Watch the CloudFormation guide",
			},
		},
		{
			name: "GCP account",
			account: &openapiclient.DescribeAccountConfigResult{
				Name:                     "test-gcp",
				GcpProjectID:             ptr("my-gcp-project"),
				GcpProjectNumber:         ptr("123456789"),
				GcpBootstrapShellCommand: ptr("bash -c (curl -fsSL https://api.omnistrate.dev/2022-09-01-00/account-setup/gcp-bootstrap.sh?account_config_id=ac-TT9j74qKTT)"),
			},
			expected: []string{
				"Next step:",
				"Verify your account",
				"1. Open Google Cloud Shell",
				"2. Execute the following command:",
				"bash -c (curl -fsSL https://api.omnistrate.dev/2022-09-01-00/account-setup/gcp-bootstrap.sh?account_config_id=ac-TT9j74qKTT)",
			},
		},
		{
			name: "Azure account",
			account: &openapiclient.DescribeAccountConfigResult{
				Name:                       "test-azure",
				AzureSubscriptionID:        ptr("azure-sub-123"),
				AzureTenantID:              ptr("azure-tenant-456"),
				AzureBootstrapShellCommand: ptr("bash -c (curl -fsSL https://api.omnistrate.dev/2022-09-01-00/account-setup/azure-bootstrap.sh?account_config_id=ac-TT9j74qKDT)"),
			},
			expected: []string{
				"Next step:",
				"Verify your account",
				"1. Open Azure Cloud Shell",
				"2. Execute the following command:",
				"bash -c (curl -fsSL https://api.omnistrate.dev/2022-09-01-00/account-setup/azure-bootstrap.sh?account_config_id=ac-TT9j74qKDT)",
			},
		},
		{
			name: "Unnamed AWS account",
			account: &openapiclient.DescribeAccountConfigResult{
				AwsAccountID: ptr("123456789012"),
			},
			expected: []string{
				"Next step:",
				"Verify your account",
				"For AWS CloudFormation users:",
			},
		},
		{
			name: "Unknown provider",
			account: &openapiclient.DescribeAccountConfigResult{
				Name: "test-unknown",
			},
			expected: []string{}, // No message should be printed for unknown provider
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := captureOutput(func() {
				dataaccess.PrintNextStepVerifyAccountMsg(tt.account)
			})

			for _, expectedPhrase := range tt.expected {
				if !strings.Contains(output, expectedPhrase) {
					t.Errorf("\nTest case: %s\nExpected phrase: %q\nNot found in output:\n%v",
						tt.name, expectedPhrase, output)
				}
			}

			// For unknown provider, verify no output is generated
			if tt.name == "Unknown provider" && output != "" {
				t.Errorf("\nTest case: %s\nExpected no output, but got:\n%v", tt.name, output)
			}
		})
	}
}

func TestPrintAccountNotVerifiedWarning(t *testing.T) {
	tests := []struct {
		name     string
		account  *openapiclient.DescribeAccountConfigResult
		expected []string
	}{
		{
			name: "AWS account",
			account: &openapiclient.DescribeAccountConfigResult{
				Name:                         "test-aws",
				AwsAccountID:                 ptr("123456789012"),
				AwsCloudFormationTemplateURL: ptr("https://example.com/template.yml"),
			},
			expected: []string{
				"WARNING! Account test-aws (ID: 123456789012)",
				"For AWS CloudFormation users:",
				"Create your CloudFormation Stack using the template at: https://example.com/template.yml",
				"For AWS Terraform users:",
				"Execute the Terraform scripts",
			},
		},
		{
			name: "GCP account",
			account: &openapiclient.DescribeAccountConfigResult{
				Name:                     "test-gcp",
				GcpProjectID:             ptr("my-gcp-project"),
				GcpProjectNumber:         ptr("123456789"),
				GcpBootstrapShellCommand: ptr("bash -c (curl -fsSL https://api.omnistrate.dev/2022-09-01-00/account-setup/gcp-bootstrap.sh?account_config_id=ac-TT9j74qKTT)"),
			},
			expected: []string{
				"WARNING! Account test-gcp (Project ID: my-gcp-project,Project Number: 123456789)",
				"Open Google Cloud Shell at: https://shell.cloud.google.com/",
				"Execute the following command:",
				"bash -c (curl -fsSL https://api.omnistrate.dev/2022-09-01-00/account-setup/gcp-bootstrap.sh?account_config_id=ac-TT9j74qKTT)",
			},
		},
		{
			name: "Azure account",
			account: &openapiclient.DescribeAccountConfigResult{
				Name:                       "test-azure",
				AzureSubscriptionID:        ptr("azure-sub-123"),
				AzureTenantID:              ptr("azure-tenant-456"),
				AzureBootstrapShellCommand: ptr("bash -c (curl -fsSL https://api.omnistrate.dev/2022-09-01-00/account-setup/azure-bootstrap.sh?account_config_id=ac-TT9j74qKDT)"),
			},
			expected: []string{
				"WARNING! Account test-azure (Subscription ID: azure-sub-123, Tenant ID: azure-tenant-456)",
				"Open Azure Cloud Shell at: https://portal.azure.com/#cloudshell/",
				"Execute the following command:",
				"bash -c (curl -fsSL https://api.omnistrate.dev/2022-09-01-00/account-setup/azure-bootstrap.sh?account_config_id=ac-TT9j74qKDT)",
			},
		},
		{
			name: "Unnamed AWS account",
			account: &openapiclient.DescribeAccountConfigResult{
				AwsAccountID: ptr("123456789012"),
			},
			expected: []string{
				"WARNING! Account Unnamed Account (ID: 123456789012)",
			},
		},
		{
			name: "Unknown provider",
			account: &openapiclient.DescribeAccountConfigResult{
				Name: "test-unknown",
			},
			expected: []string{}, // Empty since no message will be printed for unknown provider
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := captureOutput(func() {
				dataaccess.PrintAccountNotVerifiedWarning(tt.account)
			})

			for _, expectedPhrase := range tt.expected {
				if !strings.Contains(output, expectedPhrase) {
					t.Errorf("\nTest case: %s\nExpected phrase: %q\nNot found in output:\n%v",
						tt.name, expectedPhrase, output)
				}
			}

			// For unknown provider, verify no output is generated
			if tt.name == "Unknown provider" && output != "" {
				t.Errorf("\nTest case: %s\nExpected no output, but got:\n%v", tt.name, output)
			}
		})
	}
}
