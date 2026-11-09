package deploymentcell

import (
	"testing"

	openapiclientfleet "github.com/omnistrate-oss/omnistrate-sdk-go/fleet"
	"github.com/stretchr/testify/require"
)

func TestDescribeCommand(t *testing.T) {
	command, _, err := Cmd.Find([]string{"describe"})
	require.NoError(t, err)
	require.Same(t, describeCmd, command)
	require.Equal(t, "describe [deployment-cell-id]", command.Use)
	require.NotEmpty(t, command.Short)
	require.Contains(t, command.Long, "current and effective managed artifact versions")
	require.NotNil(t, command.RunE)
	require.True(t, command.SilenceUsage)

	require.NoError(t, command.ValidateArgs([]string{"hc-12345678"}))
	require.Error(t, command.ValidateArgs(nil))
	require.Error(t, command.ValidateArgs([]string{"hc-1", "hc-2"}))
}

func TestFormatDeploymentCellIncludesManagedArtifactVersions(t *testing.T) {
	currentVersion := "r0000020"
	effectiveVersion := "r0000021"
	cluster := openapiclientfleet.HostCluster{
		Id:                              "hc-12345678",
		CurrentManagedArtifactVersion:   &currentVersion,
		EffectiveManagedArtifactVersion: &effectiveVersion,
	}

	formatted := formatDeploymentCell(&cluster)
	require.Equal(t, &currentVersion, formatted.CurrentManagedArtifactVersion)
	require.Equal(t, &effectiveVersion, formatted.EffectiveManagedArtifactVersion)
	require.Equal(t, &currentVersion, formatted.ToTableView().CurrentManagedArtifactVersion)
	require.Equal(t, &effectiveVersion, formatted.ToTableView().EffectiveManagedArtifactVersion)
}
