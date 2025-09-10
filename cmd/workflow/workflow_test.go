package workflow

import (
	"testing"

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
	require.NotNil(listCmd.Flag("start-date"))
	require.NotNil(listCmd.Flag("end-date"))
	require.NotNil(listCmd.Flag("page-size"))
	require.NotNil(listCmd.Flag("next-page-token"))
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
