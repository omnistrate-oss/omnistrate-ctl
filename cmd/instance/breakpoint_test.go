package instance

import (
	"testing"

	openapiclientfleet "github.com/omnistrate-oss/omnistrate-sdk-go/fleet"
	"github.com/stretchr/testify/require"
)

func TestBreakpointCommands(t *testing.T) {
	require.NotNil(t, breakpointCmd)
	require.Equal(t, "breakpoint [operation] [flags]", breakpointCmd.Use)
	require.NotNil(t, breakpointListCmd)
	require.Equal(t, "list [instance-id]", breakpointListCmd.Use)

	found := false
	for _, cmd := range breakpointCmd.Commands() {
		if cmd.Name() == "list" {
			found = true
			break
		}
	}
	require.True(t, found, "expected breakpoint list subcommand to be registered")
}

func TestFormatBreakpointListItemsIncludesEvent(t *testing.T) {
	event := "StartTerraformPlan"
	items := formatBreakpointListItems(
		[]openapiclientfleet.WorkflowBreakpointWithStatus{
			{
				Id:     "terraform",
				Event:  &event,
				Status: "hit",
			},
		},
		[]openapiclientfleet.ResourceVersionSummary{
			{
				ResourceName: openapiclientfleet.PtrString("terraform"),
				ResourceId:   openapiclientfleet.PtrString("res-terraform"),
			},
		},
	)

	require.Len(t, items, 1)
	require.Equal(t, "terraform", items[0].Key)
	require.Equal(t, "res-terraform", items[0].ID)
	require.Equal(t, "StartTerraformPlan", items[0].Event)
	require.Equal(t, "hit", items[0].Status)
}
