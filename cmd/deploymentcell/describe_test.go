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
	require.Contains(t, command.Long, "current managed artifact version")
	require.NotNil(t, command.RunE)
	require.True(t, command.SilenceUsage)

	require.NoError(t, command.ValidateArgs([]string{"hc-12345678"}))
	require.Error(t, command.ValidateArgs(nil))
	require.Error(t, command.ValidateArgs([]string{"hc-1", "hc-2"}))
}

func TestFormatDeploymentCellIncludesCurrentManagedArtifactVersion(t *testing.T) {
	version := "r0000020"
	cluster := openapiclientfleet.HostCluster{
		Id:                            "hc-12345678",
		CurrentManagedArtifactVersion: &version,
	}

	formatted := formatDeploymentCell(&cluster)
	require.Equal(t, &version, formatted.CurrentManagedArtifactVersion)
	require.Equal(t, &version, formatted.ToTableView().CurrentManagedArtifactVersion)
}
