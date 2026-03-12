package instance

import (
	"testing"

	"github.com/omnistrate-oss/omnistrate-ctl/internal/dataaccess"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/model"
	"github.com/stretchr/testify/require"
)

func TestBuildWorkflowEventsDebugCommand(t *testing.T) {
	require := require.New(t)

	result := WorkflowMonitorResult{
		WorkflowID:        "wf-123",
		ServiceID:         "svc-123",
		EnvironmentID:     "env-123",
		FailedResourceKey: "writer",
	}

	cmd := buildWorkflowEventsDebugCommand(result)
	require.Equal(
		"omnistrate-ctl workflow events wf-123 -s svc-123 -e env-123 --detail --resource-key writer",
		cmd,
	)
}

func TestBuildWorkflowEventsDebugCommand_MissingIDs(t *testing.T) {
	require := require.New(t)

	cmd := buildWorkflowEventsDebugCommand(WorkflowMonitorResult{
		WorkflowID:    "wf-123",
		ServiceID:     "",
		EnvironmentID: "env-123",
	})
	require.Equal("", cmd)
}

func TestGetFailedStepAndMessage(t *testing.T) {
	require := require.New(t)

	events := &dataaccess.DebugEventsByWorkflowSteps{
		Compute: []dataaccess.DebugEvent{
			{EventType: string(model.WorkflowStepStarted), Message: "Compute step started"},
			{EventType: string(model.WorkflowStepFailed), Message: "Container failed health check"},
		},
	}

	step, reason := getFailedStepAndMessage(events)
	require.Equal(string(model.WorkflowStepCompute), step)
	require.Equal("Container failed health check", reason)
}

