package managedartifact

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestManagedArtifactCommands(t *testing.T) {
	require.Equal(t, "managed-artifact [operation] [flags]", Cmd.Use)

	for _, commandName := range []string{"policy", "release", "sync"} {
		command, _, err := Cmd.Find([]string{commandName})
		require.NoError(t, err)
		require.Equal(t, commandName, command.Name())
	}

	for _, command := range []*struct {
		name string
		cmd  string
	}{
		{name: "release list", cmd: "list"},
		{name: "release describe", cmd: "describe"},
	} {
		found, _, err := releaseCmd.Find([]string{command.cmd})
		require.NoError(t, err, command.name)
		require.Equal(t, command.cmd, found.Name())
	}

	require.NotNil(t, policyDescribeCmd.Flags().Lookup("environment-type"))
	require.NotNil(t, policyDescribeCmd.Flags().Lookup("cloud-provider"))
	require.NotNil(t, policyUpdateCmd.Flags().Lookup("auto-upgrade"))
	require.NotNil(t, policyUpdateCmd.Flags().Lookup("preferred-bundle-version"))
	require.NotNil(t, syncListCmd.Flags().Lookup("status"))
	require.NotNil(t, syncDescribeCmd.Flags().Lookup("id"))
}

func TestManagedArtifactValidation(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		check   func(string) error
		wantErr bool
	}{
		{name: "valid bundle", input: "r0000020", check: validateBundleVersion},
		{name: "invalid bundle", input: "v20", check: validateBundleVersion, wantErr: true},
		{name: "valid sync", input: "spabs-test-123", check: validateSyncID},
		{name: "invalid sync", input: "sync-test", check: validateSyncID, wantErr: true},
		{name: "valid timestamp", input: "2026-09-18T12:00:00Z", check: func(value string) error { return validateRFC3339Flag("updated-after", value) }},
		{name: "invalid timestamp", input: "yesterday", check: func(value string) error { return validateRFC3339Flag("updated-after", value) }, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.check(tt.input)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestManagedArtifactNormalization(t *testing.T) {
	environmentType, err := normalizeEnvironmentType("prod")
	require.NoError(t, err)
	require.Equal(t, "PROD", environmentType)

	cloudProvider, err := normalizeCloudProvider("AWS")
	require.NoError(t, err)
	require.Equal(t, "aws", cloudProvider)

	status, err := normalizeSyncStatus("in_progress")
	require.NoError(t, err)
	require.Equal(t, "IN_PROGRESS", status)

	_, err = normalizeSyncStatus("cancelled")
	require.Error(t, err)
}
