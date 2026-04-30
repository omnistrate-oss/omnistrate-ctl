package account

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/omnistrate-oss/omnistrate-ctl/internal/model"
	openapiclient "github.com/omnistrate-oss/omnistrate-sdk-go/v1"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateCloudAccountParams_Nebius(t *testing.T) {
	validBinding := openapiclient.NebiusAccountBindingInput{
		PrivateKeyPEM:    "pem-data",
		ProjectID:        "project-1",
		PublicKeyID:      "public-key-1",
		ServiceAccountID: "service-account-1",
	}

	tests := []struct {
		name    string
		params  CloudAccountParams
		wantErr string
	}{
		{
			name: "valid Nebius params",
			params: CloudAccountParams{
				Name:           "nebius-account",
				NebiusTenantID: "tenant-1",
				NebiusBindings: []openapiclient.NebiusAccountBindingInput{validBinding},
			},
		},
		{
			name: "missing Nebius bindings",
			params: CloudAccountParams{
				Name:           "nebius-account",
				NebiusTenantID: "tenant-1",
			},
			wantErr: "both --nebius-tenant-id and --nebius-bindings-file must be provided together",
		},
		{
			name: "mixed providers are rejected",
			params: CloudAccountParams{
				Name:           "mixed-account",
				AwsAccountID:   "123456789012",
				NebiusTenantID: "tenant-1",
				NebiusBindings: []openapiclient.NebiusAccountBindingInput{validBinding},
			},
			wantErr: "only one of --aws-account-id, --gcp-project-id, --azure-subscription-id, or --nebius-tenant-id can be used at a time",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateCloudAccountParams(tt.params)
			if tt.wantErr == "" {
				require.NoError(t, err)
				return
			}
			require.Error(t, err)
			assert.Equal(t, tt.wantErr, err.Error())
		})
	}
}

func TestParseNebiusBindingsFile(t *testing.T) {
	tempDir := t.TempDir()
	privateKeyPath := filepath.Join(tempDir, "private.pem")
	require.NoError(t, os.WriteFile(privateKeyPath, []byte("PEM DATA"), 0600))

	bindingsPath := filepath.Join(tempDir, "bindings.yaml")
	require.NoError(t, os.WriteFile(bindingsPath, []byte(`
bindings:
  - projectId: project-1
    serviceAccountId: service-account-1
    publicKeyId: public-key-1
    privateKeyPEMFile: private.pem
`), 0600))

	bindings, err := parseNebiusBindingsFile(bindingsPath)
	require.NoError(t, err)
	require.Len(t, bindings, 1)

	assert.Equal(t, "project-1", bindings[0].ProjectID)
	assert.Equal(t, "service-account-1", bindings[0].ServiceAccountID)
	assert.Equal(t, "public-key-1", bindings[0].PublicKeyID)
	assert.Equal(t, "PEM DATA", bindings[0].PrivateKeyPEM)
}

func TestParseNebiusBindingsFile_IgnoresLegacyRegionField(t *testing.T) {
	tempDir := t.TempDir()
	privateKeyPath := filepath.Join(tempDir, "private.pem")
	require.NoError(t, os.WriteFile(privateKeyPath, []byte("PEM DATA"), 0600))

	bindingsPath := filepath.Join(tempDir, "bindings.yaml")
	require.NoError(t, os.WriteFile(bindingsPath, []byte(`
bindings:
  - projectId: project-1
    region: eu-north1
    serviceAccountId: service-account-1
    publicKeyId: public-key-1
    privateKeyPEMFile: private.pem
`), 0600))

	bindings, err := parseNebiusBindingsFile(bindingsPath)
	require.NoError(t, err)
	require.Len(t, bindings, 1)
	assert.Equal(t, "project-1", bindings[0].ProjectID)
	assert.Equal(t, "service-account-1", bindings[0].ServiceAccountID)
}

func TestParseNebiusBindingsFile_Errors(t *testing.T) {
	tempDir := t.TempDir()
	privateKeyPath := filepath.Join(tempDir, "private.pem")
	require.NoError(t, os.WriteFile(privateKeyPath, []byte("PEM DATA"), 0600))

	tests := []struct {
		name     string
		contents string
		wantErr  string
	}{
		{
			name: "duplicate project id",
			contents: `
bindings:
  - projectId: project-1
    serviceAccountId: service-account-1
    publicKeyId: public-key-1
    privateKeyPEMFile: private.pem
  - projectId: project-1
    serviceAccountId: service-account-2
    publicKeyId: public-key-2
    privateKeyPEMFile: private.pem
`,
			wantErr: `duplicate Nebius binding for project "project-1"`,
		},
		{
			name: "missing private key source",
			contents: `
bindings:
  - projectId: project-1
    serviceAccountId: service-account-1
    publicKeyId: public-key-1
`,
			wantErr: "privateKeyPEMFile is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bindingsPath := filepath.Join(tempDir, tt.name+".yaml")
			require.NoError(t, os.WriteFile(bindingsPath, []byte(tt.contents), 0600))

			_, err := parseNebiusBindingsFile(bindingsPath)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantErr)
		})
	}
}

func TestFormatAccount_Nebius(t *testing.T) {
	account := &openapiclient.DescribeAccountConfigResult{
		Id:             "ac-123",
		Name:           "nebius-account",
		Status:         "READY",
		NebiusTenantID: ptr("tenant-1"),
		NebiusBindings: []openapiclient.NebiusAccountBindingResult{
			{ProjectID: "project-1", Region: "eu-north1", ServiceAccountID: "sa-1", PublicKeyID: "pk-1"},
			{ProjectID: "project-2", Region: "eu-west1", ServiceAccountID: "sa-2", PublicKeyID: "pk-2"},
		},
	}

	formatted, err := formatAccount(account)
	require.NoError(t, err)
	assert.Equal(t, "Nebius", formatted.CloudProvider)
	assert.Equal(t, "tenant-1 (2 bindings)", formatted.TargetAccountID)
}

func TestBuildCreateAccountOutput(t *testing.T) {
	account := &openapiclient.DescribeAccountConfigResult{
		Id:             "ac-123",
		Name:           "nebius-account",
		Status:         "PENDING",
		NebiusTenantID: ptr("tenant-1"),
		NebiusBindings: []openapiclient.NebiusAccountBindingResult{
			{ProjectID: "project-1", ServiceAccountID: "sa-1", PublicKeyID: "pk-1"},
		},
	}

	t.Run("table uses formatted account summary", func(t *testing.T) {
		output, err := buildCreateAccountOutput("table", account)
		require.NoError(t, err)

		formatted, ok := output.(model.Account)
		require.True(t, ok)
		assert.Equal(t, model.Account{
			ID:              "ac-123",
			Name:            "nebius-account",
			Status:          "PENDING",
			CloudProvider:   "Nebius",
			TargetAccountID: "tenant-1 (1 bindings)",
		}, formatted)
	})

	t.Run("json keeps raw describe payload", func(t *testing.T) {
		output, err := buildCreateAccountOutput("json", account)
		require.NoError(t, err)
		require.Same(t, account, output)
	})
}

func TestPrivateLinkFlagParsing(t *testing.T) {
	cmd := &cobra.Command{}
	addCloudAccountProviderFlags(cmd)
	cmd.Flags().Bool(privateLinkFlag, false, "")
	cmd.Flags().Bool(allowCreateNewFlag, false, "")

	// Default is false
	require.NoError(t, cmd.Flags().Set(awsAccountIDFlag, "123456789012"))
	params, err := cloudAccountParamsFromFlags(cmd, "test-account")
	require.NoError(t, err)
	assert.False(t, params.PrivateLink)

	// Set to true
	require.NoError(t, cmd.Flags().Set(privateLinkFlag, "true"))
	params, err = cloudAccountParamsFromFlags(cmd, "test-account")
	require.NoError(t, err)
	assert.True(t, params.PrivateLink)
}

func TestPrivateLinkFlagRegistered(t *testing.T) {
	// --private-link is BYOA-customer-only; provider create should NOT expose it.
	assert.Nil(t, createCmd.Flags().Lookup(privateLinkFlag),
		"--private-link must not be registered on provider account create (it is ignored by CreateCloudAccount)")
	assert.Nil(t, createCmd.Flags().Lookup(allowCreateNewFlag),
		"--allow-create-new-cloud-native-network must not be registered on provider account create")

	// Customer create owns these flags.
	assert.NotNil(t, customerCreateCmd.Flags().Lookup(privateLinkFlag))
	assert.NotNil(t, customerCreateCmd.Flags().Lookup(allowCreateNewFlag))
}

func ptr[T any](v T) *T {
	return &v
}
