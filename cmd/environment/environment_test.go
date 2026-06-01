package environment

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestEnvironmentCommands(t *testing.T) {
	require.Equal(t, "environment [operation] [flags]", Cmd.Use)
	require.Equal(t, "Manage Service Environments for your service", Cmd.Short)
	require.Contains(t, Cmd.Long, "manage the environments")

	expectedCommands := []string{"create", "list", "describe", "delete", "promote"}
	actualCommands := make([]string, 0, len(Cmd.Commands()))
	for _, cmd := range Cmd.Commands() {
		actualCommands = append(actualCommands, cmd.Name())
	}

	for _, expected := range expectedCommands {
		require.Contains(t, actualCommands, expected)
	}
}

func TestPromoteCommandFlags(t *testing.T) {
	require.Equal(t, "promote [service-name] [environment-name] [flags]", promoteCmd.Use)
	require.Equal(t, "Promote a environment", promoteCmd.Short)

	tests := []struct {
		name      string
		flagType  string
		defValue  string
		shorthand string
	}{
		{name: "service-id", flagType: "string", defValue: "", shorthand: ""},
		{name: "environment-id", flagType: "string", defValue: "", shorthand: ""},
		{name: "product-tier-id", flagType: "string", defValue: "", shorthand: "p"},
		{name: "source-version", flagType: "string", defValue: "", shorthand: ""},
	}

	for _, test := range tests {
		flag := promoteCmd.Flags().Lookup(test.name)
		require.NotNil(t, flag, "expected flag %s not found", test.name)
		require.Equal(t, test.flagType, flag.Value.Type())
		require.Equal(t, test.defValue, flag.DefValue)
		require.Equal(t, test.shorthand, flag.Shorthand)
	}
}

func TestValidatePromoteArguments(t *testing.T) {
	tests := []struct {
		name          string
		args          []string
		serviceID     string
		environmentID string
		productTierID string
		sourceVersion string
		wantErr       string
	}{
		{
			name:    "valid names",
			args:    []string{"service-name", "dev"},
			wantErr: "",
		},
		{
			name:          "valid ids",
			serviceID:     "svc-123",
			environmentID: "env-123",
			wantErr:       "",
		},
		{
			name:          "valid source version with product tier id",
			serviceID:     "svc-123",
			environmentID: "env-123",
			productTierID: "pt-123",
			sourceVersion: "1.2.3",
			wantErr:       "",
		},
		{
			name:    "missing required identifiers",
			wantErr: "please provide the service name and environment name or the service ID and environment ID",
		},
		{
			name:    "partial positional args",
			args:    []string{"service-name"},
			wantErr: "invalid arguments: service-name. Need 2 arguments: [service-name] [environment-name]",
		},
		{
			name:          "source version without product tier id",
			serviceID:     "svc-123",
			environmentID: "env-123",
			sourceVersion: "1.2.3",
			wantErr:       "source version can only be provided when product tier ID is provided",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := validatePromoteArguments(test.args, test.serviceID, test.environmentID, test.productTierID, test.sourceVersion)
			if test.wantErr == "" {
				require.NoError(t, err)
				return
			}
			require.EqualError(t, err, test.wantErr)
		})
	}
}
