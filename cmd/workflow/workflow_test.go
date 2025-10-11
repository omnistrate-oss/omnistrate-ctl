package workflow

import (
	"testing"

	"github.com/omnistrate-oss/omnistrate-sdk-go/fleet"
	"github.com/stretchr/testify/require"
)

func TestWorkflowCommands(t *testing.T) {
	require := require.New(t)

	// Test that the workflow command has all expected subcommands
	expectedCommands := []string{"list", "describe", "summary", "events", "terminate"}

	require.Equal("workflow [operation] [flags]", Cmd.Use)
	require.Contains(Cmd.Short, "Manage service workflows")

	actualCommands := make([]string, 0)
	for _, cmd := range Cmd.Commands() {
		actualCommands = append(actualCommands, cmd.Name())
	}

	for _, expected := range expectedCommands {
		require.Contains(actualCommands, expected, "Expected command %s not found", expected)
	}
}

func TestListCommandFlags(t *testing.T) {
	require := require.New(t)

	// Test that list command has expected flags
	require.NotNil(listCmd.Flag("service-id"))
	require.NotNil(listCmd.Flag("environment-id"))
	require.NotNil(listCmd.Flag("instance-id"))
	require.NotNil(listCmd.Flag("limit"))
	require.NotNil(listCmd.Flag("start-date"))
	require.NotNil(listCmd.Flag("end-date"))
	require.NotNil(listCmd.Flag("next-page-token"))

	// Test limit flag default value
	limitFlag := listCmd.Flag("limit")
	require.Equal("10", limitFlag.DefValue, "limit flag should default to 10")

	// Test limit flag type
	require.Equal("int", limitFlag.Value.Type(), "limit flag should be int type")
}

func TestDescribeCommandFlags(t *testing.T) {
	require := require.New(t)

	// Test that describe command has expected flags
	require.NotNil(describeCmd.Flag("service-id"))
	require.NotNil(describeCmd.Flag("environment-id"))
}

func TestEventsCommandFlags(t *testing.T) {
	require := require.New(t)

	// Test that events command has expected flags
	require.NotNil(eventsCmd.Flag("service-id"))
	require.NotNil(eventsCmd.Flag("environment-id"))
	require.NotNil(eventsCmd.Flag("resource-id"))
	require.NotNil(eventsCmd.Flag("resource-key"))
	require.NotNil(eventsCmd.Flag("step-names"))
	require.NotNil(eventsCmd.Flag("detail"))
	require.NotNil(eventsCmd.Flag("since"))
	require.NotNil(eventsCmd.Flag("until"))

	// Test detail flag default value
	detailFlag := eventsCmd.Flag("detail")
	require.Equal("false", detailFlag.DefValue, "detail flag should default to false")

	// Test detail flag type
	require.Equal("bool", detailFlag.Value.Type(), "detail flag should be bool type")

	// Test step-names flag type
	stepNamesFlag := eventsCmd.Flag("step-names")
	require.Equal("stringSlice", stepNamesFlag.Value.Type(), "step-names flag should be stringSlice type")
}

func TestTerminateCommandFlags(t *testing.T) {
	require := require.New(t)

	// Test that terminate command has expected flags
	require.NotNil(terminateCmd.Flag("service-id"))
	require.NotNil(terminateCmd.Flag("environment-id"))
	require.NotNil(terminateCmd.Flag("confirm"))
}

func TestSummaryCommandFlags(t *testing.T) {
	require := require.New(t)

	// Test that summary command has expected flags
	require.NotNil(summaryCmd.Flag("service-id"))
	require.NotNil(summaryCmd.Flag("environment-id"))
}

func TestDetermineStepStatus(t *testing.T) {
	require := require.New(t)

	// Test empty events
	require.Equal("unknown", determineStepStatus([]fleet.WorkflowEvent{}))

	// Test completed events
	completedEvents := []fleet.WorkflowEvent{
		{EventType: "WorkflowStepStarted", Message: "{\"message\":\"workflow step Bootstrap started.\"}"},
		{EventType: "WorkflowStepCompleted", Message: "{\"message\":\"workflow step Bootstrap completed.\"}"},
	}
	require.Equal("success", determineStepStatus(completedEvents))

	// Test failed events (should override completed)
	failedEvents := []fleet.WorkflowEvent{
		{EventType: "WorkflowStepStarted", Message: "{\"message\":\"workflow step Deployment started.\"}"},
		{EventType: "WorkflowStepCompleted", Message: "{\"message\":\"workflow step Network completed.\"}"},
		{EventType: "WorkflowStepFailed", Message: "{\"message\":\"workflow step Deployment failed.\"}"},
	}
	require.Equal("failed", determineStepStatus(failedEvents))

	// Test in-progress events (started but not completed)
	inProgressEvents := []fleet.WorkflowEvent{
		{EventType: "WorkflowStepStarted", Message: "{\"message\":\"workflow step Bootstrap started.\"}"},
		{EventType: "WorkflowStepDebug", Message: "{\"action\":\"CreatePod\",\"actionStatus\":\"Running\"}"},
	}
	require.Equal("in-progress", determineStepStatus(inProgressEvents))

	// Test unknown status (no workflow event types)
	unknownEvents := []fleet.WorkflowEvent{
		{EventType: "INFO", Message: "Some generic message"},
		{EventType: "DEBUG", Message: "Debug information"},
	}
	require.Equal("unknown", determineStepStatus(unknownEvents))
}

func TestMatchesResourceFilter(t *testing.T) {
	require := require.New(t)

	resource := fleet.EventsPerResource{
		ResourceId:   "res-123",
		ResourceKey:  "mydb",
		ResourceName: "database",
	}

	// Test matching resource ID
	require.True(matchesResourceFilter(resource, "res-123", ""))

	// Test matching resource key
	require.True(matchesResourceFilter(resource, "", "mydb"))

	// Test non-matching resource ID
	require.False(matchesResourceFilter(resource, "res-456", ""))

	// Test non-matching resource key
	require.False(matchesResourceFilter(resource, "", "other"))

	// Test no filters (should match)
	require.True(matchesResourceFilter(resource, "", ""))
}

func TestMatchesStepNameFilter(t *testing.T) {
	require := require.New(t)

	// Test no filter (should match all)
	require.True(matchesStepNameFilter("Bootstrap", []string{}))

	// Test matching filter
	require.True(matchesStepNameFilter("Bootstrap", []string{"Bootstrap", "Deployment"}))

	// Test non-matching filter
	require.False(matchesStepNameFilter("Bootstrap", []string{"Network", "Storage"}))

	// Test case insensitive matching
	require.True(matchesStepNameFilter("bootstrap", []string{"Bootstrap"}))
}
