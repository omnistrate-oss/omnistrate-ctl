package instance

import (
	"encoding/json"
	"testing"

	openapiclientfleet "github.com/omnistrate-oss/omnistrate-sdk-go/fleet"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

func TestOperationCommands(t *testing.T) {
	require := require.New(t)

	require.Equal("operation [list|describe|trigger]", operationCmd.Use)
	require.NotNil(operationCmd.Commands())
	require.NotNil(operationTriggerCmd.Flag("param"))
	require.NotNil(operationTriggerCmd.Flag("param-file"))
	require.NotNil(operationTriggerCmd.Flag("resource-id"))
	require.NotNil(operationTriggerCmd.Flag("capacity"))
	require.NotNil(operationTriggerCmd.Flag("failed-replica-id"))
	require.NotNil(operationTriggerCmd.Flag("failed-replica-action"))
	require.NotNil(operationTriggerCmd.Flag("yes"))

	commandNames := make([]string, 0, len(Cmd.Commands()))
	for _, command := range Cmd.Commands() {
		commandNames = append(commandNames, command.Name())
	}
	require.Contains(commandNames, "operation")
	require.NotContains(commandNames, "custom-workflow")
}

func TestFindSupportedOperation(t *testing.T) {
	customWorkflowID := "cwt-123"
	operations := []openapiclientfleet.ResourceInstanceSupportedOperation{
		{
			Id:     &customWorkflowID,
			Name:   "Collect status",
			Source: operationSourceCustomWorkflow,
			Verb:   "collectStatus",
		},
		{
			Name:   "Backup",
			Source: operationSourceSystemWorkflow,
			Verb:   operationVerbBackup,
		},
	}

	byID, err := findSupportedOperation(operations, customWorkflowID)
	require.NoError(t, err)
	require.Equal(t, "Collect status", byID.Name)

	byVerb, err := findSupportedOperation(operations, "backup")
	require.NoError(t, err)
	require.Equal(t, operationVerbBackup, byVerb.Verb)

	byName, err := findSupportedOperation(operations, "collect status")
	require.NoError(t, err)
	require.Equal(t, customWorkflowID, *byName.Id)

	_, err = findSupportedOperation(operations, "missing")
	require.ErrorContains(t, err, `custom operation "missing" not found`)
}

func TestOperationOperationsFiltersLegacyOperations(t *testing.T) {
	customWorkflowID := "cwt-123"
	operations := []openapiclientfleet.ResourceInstanceSupportedOperation{
		{
			Name:   "Legacy backup",
			Source: "LEGACY",
			Verb:   operationVerbBackup,
		},
		{
			Id:     &customWorkflowID,
			Name:   "Collect status",
			Source: operationSourceCustomWorkflow,
			Verb:   "collectStatus",
		},
		{
			Name:   "System backup",
			Source: operationSourceSystemWorkflow,
			Verb:   operationVerbBackup,
		},
	}

	filteredOperations := operationOperations(operations)
	require.Len(t, filteredOperations, 2)
	require.Equal(t, operationSourceCustomWorkflow, filteredOperations[0].Source)
	require.Equal(t, operationSourceSystemWorkflow, filteredOperations[1].Source)
}

func TestOperationParams(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.Flags().Int64("capacity", 0, "")
	cmd.Flags().String("failed-replica-id", "", "")
	cmd.Flags().String("failed-replica-action", "", "")

	capacity, err := capacityParam(cmd, map[string]any{"capacityToBeAdded": float64(3)}, "capacityToBeAdded", operationVerbAddCapacity)
	require.NoError(t, err)
	require.Equal(t, int64(3), capacity)

	capacity, err = capacityParam(cmd, map[string]any{"capacity": json.Number("4")}, "capacityToBeRemoved", operationVerbRemoveCapacity)
	require.NoError(t, err)
	require.Equal(t, int64(4), capacity)

	require.NoError(t, cmd.Flags().Set("capacity", "5"))
	capacity, err = capacityParam(cmd, nil, "capacityToBeAdded", operationVerbAddCapacity)
	require.NoError(t, err)
	require.Equal(t, int64(5), capacity)

	require.NoError(t, cmd.Flags().Set("failed-replica-id", "replica-1"))
	require.NoError(t, cmd.Flags().Set("failed-replica-action", "delete"))
	failedReplicaID, failedReplicaAction, err := failoverParams(cmd, nil)
	require.NoError(t, err)
	require.Equal(t, "replica-1", failedReplicaID)
	require.Equal(t, "delete", failedReplicaAction)
}

func TestOperationParamErrors(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.Flags().Int64("capacity", 0, "")

	_, err := capacityParam(cmd, nil, "capacityToBeAdded", operationVerbAddCapacity)
	require.ErrorContains(t, err, "ADD_CAPACITY requires --capacity or request param capacityToBeAdded")

	_, err = int64Param(float64(3.7))
	require.ErrorContains(t, err, "capacity value 3.7 is not a whole number")
}
