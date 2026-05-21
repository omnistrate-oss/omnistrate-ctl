package instance

import (
	"encoding/json"
	"testing"

	openapiclientfleet "github.com/omnistrate-oss/omnistrate-sdk-go/fleet"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

func TestCustomWorkflowCommands(t *testing.T) {
	require := require.New(t)

	require.Equal("custom-workflow [list|describe|trigger]", customWorkflowCmd.Use)
	require.NotNil(customWorkflowCmd.Commands())
	require.NotNil(customWorkflowTriggerCmd.Flag("param"))
	require.NotNil(customWorkflowTriggerCmd.Flag("param-file"))
	require.NotNil(customWorkflowTriggerCmd.Flag("resource-id"))
	require.NotNil(customWorkflowTriggerCmd.Flag("capacity"))
	require.NotNil(customWorkflowTriggerCmd.Flag("failed-replica-id"))
	require.NotNil(customWorkflowTriggerCmd.Flag("failed-replica-action"))
	require.NotNil(customWorkflowTriggerCmd.Flag("yes"))

	commandNames := make([]string, 0, len(Cmd.Commands()))
	for _, command := range Cmd.Commands() {
		commandNames = append(commandNames, command.Name())
	}
	require.Contains(commandNames, "custom-workflow")
}

func TestFindSupportedCustomWorkflow(t *testing.T) {
	customWorkflowID := "cwt-123"
	customWorkflows := []openapiclientfleet.ResourceInstanceSupportedOperation{
		{
			Id:     &customWorkflowID,
			Name:   "Collect status",
			Source: customWorkflowSourceCustomWorkflow,
			Verb:   "collectStatus",
		},
		{
			Name:   "Backup",
			Source: customWorkflowSourceSystemWorkflow,
			Verb:   customWorkflowVerbBackup,
		},
	}

	byID, err := findSupportedCustomWorkflow(customWorkflows, customWorkflowID)
	require.NoError(t, err)
	require.Equal(t, "Collect status", byID.Name)

	byVerb, err := findSupportedCustomWorkflow(customWorkflows, "backup")
	require.NoError(t, err)
	require.Equal(t, customWorkflowVerbBackup, byVerb.Verb)

	byName, err := findSupportedCustomWorkflow(customWorkflows, "collect status")
	require.NoError(t, err)
	require.Equal(t, customWorkflowID, *byName.Id)

	_, err = findSupportedCustomWorkflow(customWorkflows, "missing")
	require.ErrorContains(t, err, `custom workflow "missing" not found`)
}

func TestCustomWorkflowParams(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.Flags().Int64("capacity", 0, "")
	cmd.Flags().String("failed-replica-id", "", "")
	cmd.Flags().String("failed-replica-action", "", "")

	capacity, err := capacityParam(cmd, map[string]any{"capacityToBeAdded": float64(3)}, "capacityToBeAdded")
	require.NoError(t, err)
	require.Equal(t, int64(3), capacity)

	capacity, err = capacityParam(cmd, map[string]any{"capacity": json.Number("4")}, "capacityToBeRemoved")
	require.NoError(t, err)
	require.Equal(t, int64(4), capacity)

	require.NoError(t, cmd.Flags().Set("capacity", "5"))
	capacity, err = capacityParam(cmd, nil, "capacityToBeAdded")
	require.NoError(t, err)
	require.Equal(t, int64(5), capacity)

	require.NoError(t, cmd.Flags().Set("failed-replica-id", "replica-1"))
	require.NoError(t, cmd.Flags().Set("failed-replica-action", "delete"))
	failedReplicaID, failedReplicaAction, err := failoverParams(cmd, nil)
	require.NoError(t, err)
	require.Equal(t, "replica-1", failedReplicaID)
	require.Equal(t, "delete", failedReplicaAction)
}

func TestCustomWorkflowParamErrors(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.Flags().Int64("capacity", 0, "")

	_, err := capacityParam(cmd, nil, "capacityToBeAdded")
	require.ErrorContains(t, err, "capacity requires --capacity or request param capacityToBeAdded")
}
