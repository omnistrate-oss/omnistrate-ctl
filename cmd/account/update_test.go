package account

import (
	"os"
	"path/filepath"
	"testing"

	openapiclient "github.com/omnistrate-oss/omnistrate-sdk-go/v1"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateUpdateAccountParams(t *testing.T) {
	validBinding := openapiclient.NebiusAccountBindingInput{
		PrivateKeyPEM:    "pem-data",
		ProjectID:        "project-1",
		PublicKeyID:      "public-key-1",
		ServiceAccountID: "service-account-1",
	}

	tests := []struct {
		name    string
		params  UpdateCloudAccountParams
		wantErr string
	}{
		{
			name: "metadata only is allowed",
			params: UpdateCloudAccountParams{
				ID:   "ac-123",
				Name: ptr("updated-name"),
			},
		},
		{
			name: "nebius binding replacement is allowed",
			params: UpdateCloudAccountParams{
				ID:             "ac-123",
				NebiusBindings: []openapiclient.NebiusAccountBindingInput{validBinding},
			},
		},
		{
			name: "custom tags replacement is allowed",
			params: UpdateCloudAccountParams{
				ID:         "ac-123",
				CustomTags: []openapiclient.CustomTag{{Key: "env", Value: "prod"}},
			},
		},
		{
			name: "private artifact publication setting is allowed",
			params: UpdateCloudAccountParams{
				ID:                                "ac-123",
				PrivateArtifactPublicationEnabled: ptr(false),
			},
		},
		{
			name: "requires mutable fields",
			params: UpdateCloudAccountParams{
				ID: "ac-123",
			},
			wantErr: "at least one of --name, --description, --nebius-bindings-file, --tags, or --private-artifact-publication-enabled must be provided",
		},
		{
			name: "rejects empty name",
			params: UpdateCloudAccountParams{
				ID:   "ac-123",
				Name: ptr(""),
			},
			wantErr: "name cannot be empty",
		},
		{
			name: "rejects empty description",
			params: UpdateCloudAccountParams{
				ID:          "ac-123",
				Description: ptr(""),
			},
			wantErr: "description cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateUpdateAccountParams(tt.params)
			if tt.wantErr == "" {
				require.NoError(t, err)
				return
			}

			require.EqualError(t, err, tt.wantErr)
		})
	}
}

func TestBuildUpdateAccountParams_UsesBindingsParser(t *testing.T) {
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

	cmd := &cobra.Command{}
	cmd.Flags().String("name", "", "")
	cmd.Flags().String("description", "", "")
	require.NoError(t, cmd.Flags().Set("name", "updated-name"))

	publicationEnabled := false
	params, err := buildUpdateAccountParams(cmd, "ac-123", "updated-name", "", bindingsPath, &publicationEnabled)
	require.NoError(t, err)
	require.Equal(t, "ac-123", params.ID)
	require.NotNil(t, params.Name)
	assert.Equal(t, "updated-name", *params.Name)
	require.Len(t, params.NebiusBindings, 1)
	assert.Equal(t, "project-1", params.NebiusBindings[0].ProjectID)
	assert.Equal(t, "PEM DATA", params.NebiusBindings[0].PrivateKeyPEM)
	require.NotNil(t, params.PrivateArtifactPublicationEnabled)
	assert.False(t, *params.PrivateArtifactPublicationEnabled)
}

func TestOptionalBoolFlag(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want *bool
	}{
		{name: "unset flag is omitted"},
		{name: "explicit true is sent", args: []string{"--enabled"}, want: ptr(true)},
		{name: "explicit false is sent", args: []string{"--enabled=false"}, want: ptr(false)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := &cobra.Command{}
			cmd.Flags().Bool("enabled", false, "")
			require.NoError(t, cmd.Flags().Parse(tt.args))

			got, err := optionalBoolFlag(cmd, "enabled")
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestGetPrivateArtifactPublicationState(t *testing.T) {
	applied := func(enabled bool) *openapiclient.AccountConfigSettings {
		return &openapiclient.AccountConfigSettings{
			AccountGovernanceControls: &openapiclient.AccountGovernanceControls{
				AdditionalProperties: map[string]interface{}{"privateArtifactPublicationEnabled": enabled},
			},
		}
	}

	tests := []struct {
		name     string
		account  openapiclient.DescribeAccountConfigResult
		disabled bool
		want     privateArtifactPublicationState
		wantErr  string
	}{
		{
			name:     "ready without observed value is not verified",
			account:  openapiclient.DescribeAccountConfigResult{Status: "READY"},
			disabled: true,
			want:     privateArtifactPublicationPending,
		},
		{
			name:     "ready with observed disabled value is applied",
			account:  openapiclient.DescribeAccountConfigResult{Status: "READY", AppliedSettings: applied(false)},
			disabled: true,
			want:     privateArtifactPublicationApplied,
		},
		{
			name:    "ready with requested value applied",
			account: openapiclient.DescribeAccountConfigResult{Status: "READY", AppliedSettings: applied(true)},
			want:    privateArtifactPublicationApplied,
		},
		{
			name:    "stale ready with previous value is pending",
			account: openapiclient.DescribeAccountConfigResult{Status: "READY", AppliedSettings: applied(false)},
			want:    privateArtifactPublicationPending,
		},
		{
			name:    "ready without applied settings is pending",
			account: openapiclient.DescribeAccountConfigResult{Status: "READY"},
			want:    privateArtifactPublicationPending,
		},
		{
			name: "verified account waiting for stack update",
			account: openapiclient.DescribeAccountConfigResult{
				Status:        "PENDING_UPDATE",
				StatusMessage: accountSettingsPendingStatusMessage,
			},
			want: privateArtifactPublicationStackUpdateRequired,
		},
		{
			name: "update before verification is pending",
			account: openapiclient.DescribeAccountConfigResult{
				Status:        "PENDING_UPDATE",
				StatusMessage: "update CloudFormation account stack to match requested settings",
			},
			want: privateArtifactPublicationPending,
		},
		{
			name:    "verifying is pending",
			account: openapiclient.DescribeAccountConfigResult{Status: "VERIFYING"},
			want:    privateArtifactPublicationPending,
		},
		{
			name:    "failed verification returns an error",
			account: openapiclient.DescribeAccountConfigResult{Status: "FAILED", StatusMessage: "access denied"},
			wantErr: "account settings update failed: access denied",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := getPrivateArtifactPublicationState(&tt.account, !tt.disabled)
			if tt.wantErr != "" {
				require.EqualError(t, err, tt.wantErr)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestPrintPrivateArtifactPublicationResult_ReturnsFailure(t *testing.T) {
	t.Setenv("OMNISTRATE_DRY_RUN", "true")

	err := printPrivateArtifactPublicationResult("json", &openapiclient.DescribeAccountConfigResult{
		Id:            "ac-123",
		Status:        "FAILED",
		StatusMessage: "access denied",
	}, true)

	require.EqualError(t, err,
		"account update was submitted, but the private artifact publication setting was not applied: account settings update failed: access denied")
}

func TestValidateUpdateTargetAccount(t *testing.T) {
	publicationParams := UpdateCloudAccountParams{ID: "ac-123", PrivateArtifactPublicationEnabled: ptr(true)}

	tests := []struct {
		name    string
		params  UpdateCloudAccountParams
		account openapiclient.DescribeAccountConfigResult
		wantErr string
	}{
		{
			name:    "AWS non-BYOA account accepts private artifact publication",
			params:  publicationParams,
			account: openapiclient.DescribeAccountConfigResult{AwsAccountID: ptr("123456789012")},
		},
		{
			name:    "non-AWS account rejects private artifact publication",
			params:  publicationParams,
			account: openapiclient.DescribeAccountConfigResult{GcpProjectID: ptr("project-1")},
			wantErr: "--private-artifact-publication-enabled is only supported for AWS accounts",
		},
		{
			name:   "BYOA account rejects private artifact publication",
			params: publicationParams,
			account: openapiclient.DescribeAccountConfigResult{
				AwsAccountID:    ptr("123456789012"),
				ByoaInstanceIDs: []string{"instance-1"},
			},
			wantErr: "--private-artifact-publication-enabled is not supported for BYOA accounts",
		},
		{
			name: "non-Nebius account rejects Nebius bindings",
			params: UpdateCloudAccountParams{
				ID:             "ac-123",
				NebiusBindings: []openapiclient.NebiusAccountBindingInput{{ProjectID: "project-1"}},
			},
			account: openapiclient.DescribeAccountConfigResult{AwsAccountID: ptr("123456789012")},
			wantErr: "Nebius bindings can only be updated for Nebius accounts",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateUpdateTargetAccount(tt.params, &tt.account)
			if tt.wantErr == "" {
				require.NoError(t, err)
				return
			}

			require.EqualError(t, err, tt.wantErr)
		})
	}
}
