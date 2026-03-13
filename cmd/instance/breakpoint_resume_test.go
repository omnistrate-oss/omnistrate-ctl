package instance

import (
	"testing"

	openapiclientfleet "github.com/omnistrate-oss/omnistrate-sdk-go/fleet"
	"github.com/stretchr/testify/require"
)

func TestSummarizeBreakpointForResumeFormatsResourceKeyAndID(t *testing.T) {
	resourceSummaries := []openapiclientfleet.ResourceVersionSummary{
		{
			ResourceName: new("writer"),
			ResourceId:   new("res-writer"),
		},
		{
			ResourceName: new("reader"),
			ResourceId:   new("res-reader"),
		},
	}

	t.Run("hit breakpoint by resource key", func(t *testing.T) {
		activeBreakpoints := []openapiclientfleet.WorkflowBreakpointWithStatus{
			{
				Id:     "writer",
				Status: "hit",
			},
		}

		summary := summarizeBreakpointForResume(activeBreakpoints, resourceSummaries)
		require.Equal(t, "writer [res-writer]", summary)
	})

	t.Run("hit breakpoint by resource id", func(t *testing.T) {
		activeBreakpoints := []openapiclientfleet.WorkflowBreakpointWithStatus{
			{
				Id:     "res-reader",
				Status: "hit",
			},
		}

		summary := summarizeBreakpointForResume(activeBreakpoints, resourceSummaries)
		require.Equal(t, "reader [res-reader]", summary)
	})

	t.Run("multiple hit breakpoints", func(t *testing.T) {
		activeBreakpoints := []openapiclientfleet.WorkflowBreakpointWithStatus{
			{
				Id:     "writer",
				Status: "hit",
			},
			{
				Id:     "res-reader",
				Status: "hit",
			},
		}

		summary := summarizeBreakpointForResume(activeBreakpoints, resourceSummaries)
		require.Equal(t, "writer [res-writer], reader [res-reader] (multiple hit breakpoints)", summary)
	})
}
