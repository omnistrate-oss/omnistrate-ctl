package service

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestResourceDeleteNestedCommandRegistered(t *testing.T) {
	resourceDeleteCmd, _, err := Cmd.Find([]string{"resource", "delete"})

	require.NoError(t, err)
	require.NotNil(t, resourceDeleteCmd)
	require.Equal(t, "delete", resourceDeleteCmd.Name())
	require.NotEmpty(t, resourceDeleteCmd.Example)
	require.True(t, resourceDeleteCmd.SilenceUsage)

	serviceIDFlag := resourceDeleteCmd.Flags().Lookup("service-id")
	require.NotNil(t, serviceIDFlag)
	require.Equal(t, "", serviceIDFlag.DefValue)

	serviceNameFlag := resourceDeleteCmd.Flags().Lookup("service-name")
	require.NotNil(t, serviceNameFlag)
	require.Equal(t, "", serviceNameFlag.DefValue)

	resourceIDFlag := resourceDeleteCmd.Flags().Lookup("resource-id")
	require.NotNil(t, resourceIDFlag)
	require.Equal(t, "", resourceIDFlag.DefValue)

	dryRunFlag := resourceDeleteCmd.Flags().Lookup("dry-run")
	require.NotNil(t, dryRunFlag)
	require.Equal(t, "false", dryRunFlag.DefValue)
}

func TestNormalizeServiceResourceDeleteArgs(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		wantArgs    []string
		wantRewrote bool
	}{
		{
			name:        "rewrites positional service name or ID resource delete",
			args:        []string{"service-selector", "resource", "delete", "--resource-id", "resource-id"},
			wantArgs:    []string{"resource", "delete", "--service-name", "service-selector", "--resource-id", "resource-id"},
			wantRewrote: true,
		},
		{
			name:        "preserves canonical resource delete",
			args:        []string{"resource", "delete", "--service-id", "service-id", "--resource-id", "resource-id"},
			wantArgs:    []string{"resource", "delete", "--service-id", "service-id", "--resource-id", "resource-id"},
			wantRewrote: false,
		},
		{
			name:        "preserves existing service delete",
			args:        []string{"delete", "--id", "service-id"},
			wantArgs:    []string{"delete", "--id", "service-id"},
			wantRewrote: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotArgs, gotRewrote := normalizeServiceResourceDeleteArgs(tt.args)

			require.Equal(t, tt.wantArgs, gotArgs)
			require.Equal(t, tt.wantRewrote, gotRewrote)
		})
	}
}

func TestValidateResourceDeleteArguments(t *testing.T) {
	tests := []struct {
		name        string
		serviceName string
		serviceID   string
		resourceID  string
		expectedErr string
	}{
		{
			name:       "valid with service ID",
			serviceID:  "service-id",
			resourceID: "resource-id",
		},
		{
			name:        "valid with service name",
			serviceName: "service-name",
			resourceID:  "resource-id",
		},
		{
			name:        "missing service name or ID",
			resourceID:  "resource-id",
			expectedErr: "service name or ID must be provided",
		},
		{
			name:        "both service name and ID",
			serviceName: "service-name",
			serviceID:   "service-id",
			resourceID:  "resource-id",
			expectedErr: "only one of service name or ID can be provided",
		},
		{
			name:        "missing resource ID",
			serviceID:   "service-id",
			expectedErr: "resource ID must be provided",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateResourceDeleteArguments(tt.serviceName, tt.serviceID, tt.resourceID)
			if tt.expectedErr == "" {
				require.NoError(t, err)
				return
			}
			require.EqualError(t, err, tt.expectedErr)
		})
	}
}

func TestParseServiceResourceDeletePath(t *testing.T) {
	serviceName, serviceID, resourceID, dryRun, output, err := parseServiceResourceDeletePath([]string{
		"resource",
		"delete",
		"--service-name",
		"service-selector",
		"--resource-id",
		"resource-id",
		"--dry-run",
		"--output",
		"json",
	})

	require.NoError(t, err)
	require.Equal(t, "service-selector", serviceName)
	require.Empty(t, serviceID)
	require.Equal(t, "resource-id", resourceID)
	require.True(t, dryRun)
	require.Equal(t, "json", output)
}

func TestRunServiceResourceDeletePathHelp(t *testing.T) {
	handled, err := runServiceResourceDeletePath(Cmd, []string{
		"service-name",
		"resource",
		"delete",
		"--help",
	})

	require.True(t, handled)
	require.NoError(t, err)
}

func TestServicePlanNestedCommandRegistered(t *testing.T) {
	planCmd, _, err := Cmd.Find([]string{"plan"})

	require.NoError(t, err)
	require.NotNil(t, planCmd)
	require.Equal(t, "plan", planCmd.Name())

	expected := []string{
		"delete",
		"describe",
		"describe-version",
		"disable-feature",
		"enable-feature",
		"list",
		"list-versions",
		"release",
		"set-default",
		"update",
	}
	actual := make([]string, 0, len(planCmd.Commands()))
	for _, command := range planCmd.Commands() {
		actual = append(actual, command.Name())
	}
	require.ElementsMatch(t, expected, actual)
}
