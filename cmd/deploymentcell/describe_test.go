package deploymentcell

import (
	"encoding/json"
	"testing"

	openapiclientfleet "github.com/omnistrate-oss/omnistrate-sdk-go/fleet"
	"github.com/stretchr/testify/require"
)

func TestDescribeCommand(t *testing.T) {
	command, _, err := Cmd.Find([]string{"describe"})
	require.NoError(t, err)
	require.Same(t, describeCmd, command)

	require.NoError(t, command.ValidateArgs([]string{"hc-12345678"}))
	require.Error(t, command.ValidateArgs(nil))
	require.Error(t, command.ValidateArgs([]string{"hc-1", "hc-2"}))
}

func TestFormatDeploymentCellIncludesManagedArtifactVersions(t *testing.T) {
	currentVersion := "r0000020"
	effectiveVersion := "r0000021"
	var cluster openapiclientfleet.HostCluster
	err := json.Unmarshal([]byte(`{
		"accountConfigId":"ac-1",
		"accountID":"123456789012",
		"cloudProvider":"aws",
		"currentManagedArtifactVersion":"r0000020",
		"currentNumberOfDeployments":0,
		"description":"test",
		"effectiveManagedArtifactVersion":"r0000021",
		"id":"hc-12345678",
		"isCustomDeployment":false,
		"key":"dataplane-test",
		"region":"us-east-1",
		"regionId":"region-1",
		"status":"RUNNING",
		"type":"HOSTED"
	}`), &cluster)
	require.NoError(t, err)

	formatted := formatDeploymentCell(&cluster)
	require.Equal(t, &currentVersion, formatted.CurrentManagedArtifactVersion)
	require.Equal(t, &effectiveVersion, formatted.EffectiveManagedArtifactVersion)
}

func TestResolveDeploymentCellID(t *testing.T) {
	hostClusters := []openapiclientfleet.HostCluster{
		{Id: "hc-managed", Key: "dataplane-managed"},
		{Id: "hc-adopted", Key: "my-adopted-cell"},
	}

	require.Equal(t, "hc-managed", resolveDeploymentCellID("hc-managed", hostClusters))
	require.Equal(t, "hc-adopted", resolveDeploymentCellID("my-adopted-cell", hostClusters))
	require.Equal(t, "unknown", resolveDeploymentCellID("unknown", hostClusters))
}
