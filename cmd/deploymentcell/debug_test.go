package deploymentcell

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDeploymentCellDebugAccessReasonIsAutomatic(t *testing.T) {
	require.Nil(t, debugCmd.Flags().Lookup("reason"))
	require.Equal(t, "cluster-reader", deploymentCellDebugAccessRole)
	require.Equal(t,
		"Debug deployment cell hc-test using omnistrate-ctl",
		deploymentCellDebugAccessReason("hc-test"))
}
