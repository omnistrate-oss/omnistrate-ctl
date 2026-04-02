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
			name: "requires mutable fields",
			params: UpdateCloudAccountParams{
				ID: "ac-123",
			},
			wantErr: "at least one of --name, --description, or --nebius-bindings-file must be provided",
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

	params, err := buildUpdateAccountParams(cmd, "ac-123", "updated-name", "", bindingsPath)
	require.NoError(t, err)
	require.Equal(t, "ac-123", params.ID)
	require.NotNil(t, params.Name)
	assert.Equal(t, "updated-name", *params.Name)
	require.Len(t, params.NebiusBindings, 1)
	assert.Equal(t, "project-1", params.NebiusBindings[0].ProjectID)
	assert.Equal(t, "PEM DATA", params.NebiusBindings[0].PrivateKeyPEM)
}
