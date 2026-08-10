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
	require.False(t, Cmd.DisableFlagParsing)
	require.Contains(t, resourceDeleteCmd.Use, "--service-id")
	require.Contains(t, resourceDeleteCmd.Use, "--resource-id")

	serviceIDFlag := resourceDeleteCmd.Flags().Lookup("service-id")
	require.NotNil(t, serviceIDFlag)
	require.Equal(t, "", serviceIDFlag.DefValue)

	serviceNameFlag := resourceDeleteCmd.Flags().Lookup("service-name")
	require.Nil(t, serviceNameFlag)

	resourceIDFlag := resourceDeleteCmd.Flags().Lookup("resource-id")
	require.NotNil(t, resourceIDFlag)
	require.Equal(t, "", resourceIDFlag.DefValue)

	dryRunFlag := resourceDeleteCmd.Flags().Lookup("dry-run")
	require.Nil(t, dryRunFlag)
}

func TestValidateResourceDeleteArguments(t *testing.T) {
	tests := []struct {
		name        string
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
			name:        "missing service ID",
			resourceID:  "resource-id",
			expectedErr: "service ID must be provided",
		},
		{
			name:        "missing resource ID",
			serviceID:   "service-id",
			expectedErr: "resource ID must be provided",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateResourceDeleteArguments(tt.serviceID, tt.resourceID)
			if tt.expectedErr == "" {
				require.NoError(t, err)
				return
			}
			require.EqualError(t, err, tt.expectedErr)
		})
	}
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
