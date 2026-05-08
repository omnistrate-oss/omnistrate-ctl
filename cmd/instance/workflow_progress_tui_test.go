package instance

import (
	"fmt"
	"strings"
	"testing"

	bubbleSpinner "github.com/charmbracelet/bubbles/spinner"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/dataaccess"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/model"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/utils"
	"github.com/stretchr/testify/require"
)

func TestBuildWorkflowProgressSectionsTracksCanonicalSegments(t *testing.T) {
	events := &dataaccess.DebugEventsByWorkflowSteps{
		Bootstrap: []dataaccess.DebugEvent{
			{EventType: string(model.WorkflowStepCompleted), Message: "bootstrap complete"},
		},
		Compute: []dataaccess.DebugEvent{
			{EventType: string(model.WorkflowStepStarted), Message: "compute started"},
		},
	}

	sections := buildWorkflowProgressSections(events, "running")

	require.Len(t, sections, 6)
	require.Equal(t, "Bootstrap", sections[0].Name)
	require.Equal(t, "completed", sections[0].Status)
	require.Equal(t, "Compute", sections[3].Name)
	require.Equal(t, "running", sections[3].Status)
	require.Equal(t, "Deployment", sections[4].Name)
	require.Equal(t, "pending", sections[4].Status)
}

func TestBuildWorkflowProgressSectionsCompletesAllOnSuccessfulWorkflow(t *testing.T) {
	sections := buildWorkflowProgressSections(&dataaccess.DebugEventsByWorkflowSteps{}, "success")

	require.Len(t, sections, 6)
	for _, section := range sections {
		require.Equal(t, "completed", section.Status, section.Name)
	}
}

func TestBuildWorkflowProgressSectionsMarksLastActiveStepOnFailure(t *testing.T) {
	events := &dataaccess.DebugEventsByWorkflowSteps{
		Network: []dataaccess.DebugEvent{
			{EventType: string(model.WorkflowStepStarted), Message: "network started"},
		},
	}

	sections := buildWorkflowProgressSections(events, "failed")

	require.Equal(t, "failed", sections[2].Status)
	require.Equal(t, "pending", sections[3].Status)
}

func TestBuildWorkflowProgressResourceComputesSegmentProgress(t *testing.T) {
	events := &dataaccess.DebugEventsByWorkflowSteps{
		Bootstrap: []dataaccess.DebugEvent{{EventType: string(model.WorkflowStepCompleted)}},
		Compute:   []dataaccess.DebugEvent{{EventType: string(model.WorkflowStepStarted)}},
	}
	workflowInfo := &dataaccess.WorkflowInfo{WorkflowStatus: "running"}

	resource := buildWorkflowProgressResource(dataaccess.ResourceWorkflowDebugEvents{
		ResourceID:           "r-1",
		ResourceKey:          "db",
		ResourceName:         "database",
		EventsByWorkflowStep: events,
	}, workflowInfo)

	require.Equal(t, "database", workflowProgressResourceName(resource))
	require.Equal(t, "running", resource.Status)
	require.Equal(t, 1, resource.CompletedSteps)
	require.Equal(t, 6, resource.TotalSteps)
	require.Equal(t, 25, resource.Percent)
}

func TestBuildWorkflowProgressResourceCompletesWhenAllSectionsComplete(t *testing.T) {
	events := &dataaccess.DebugEventsByWorkflowSteps{
		Bootstrap:  []dataaccess.DebugEvent{{EventType: string(model.WorkflowStepCompleted)}},
		Storage:    []dataaccess.DebugEvent{{EventType: string(model.WorkflowStepCompleted)}},
		Network:    []dataaccess.DebugEvent{{EventType: string(model.WorkflowStepCompleted)}},
		Compute:    []dataaccess.DebugEvent{{EventType: string(model.WorkflowStepCompleted)}},
		Deployment: []dataaccess.DebugEvent{{EventType: string(model.WorkflowStepCompleted)}},
		Monitoring: []dataaccess.DebugEvent{{EventType: string(model.WorkflowStepCompleted)}},
	}
	workflowInfo := &dataaccess.WorkflowInfo{WorkflowStatus: "running"}

	resource := buildWorkflowProgressResource(dataaccess.ResourceWorkflowDebugEvents{
		ResourceID:           "r-1",
		ResourceName:         "db",
		EventsByWorkflowStep: events,
	}, workflowInfo)

	require.Equal(t, "completed", resource.Status)
	require.Equal(t, 6, resource.CompletedSteps)
	require.Equal(t, 6, resource.TotalSteps)
	require.Equal(t, 100, resource.Percent)
}

func TestWorkflowProgressFailureMessagePrefersResourceName(t *testing.T) {
	snapshot := workflowProgressSnapshot{
		WorkflowStatus: "failed",
		Resources: []workflowProgressResource{
			{Name: "database", Status: "failed"},
		},
	}

	require.Equal(t, "for resource database", workflowProgressFailureMessage(snapshot))
}

func TestWorkflowProgressViewRendersProgressAndSections(t *testing.T) {
	m := newWorkflowProgressModel("inst-1", "create", "aws", "us-east-1")
	m.loading = false
	m.events = map[string][]workflowProgressEvent{
		"database": {
			{
				ResourceName: "database",
				Section:      "Compute",
				Action:       "BatchCreatePods",
				Message:      "pod is not ready: Pending",
			},
		},
	}
	m.snapshot = workflowProgressSnapshot{
		InstanceID:         "inst-1",
		WorkflowID:         "submit-create-1",
		WorkflowStatus:     "running",
		OverallPercent:     25,
		TotalResources:     1,
		CompletedResources: 0,
		Resources: []workflowProgressResource{
			{
				Name:           "database",
				Status:         "running",
				Percent:        25,
				CompletedSteps: 1,
				TotalSteps:     6,
				Events: []workflowProgressEvent{
					{ResourceName: "database", Section: "Compute", Action: "BatchCreatePods", Message: "pod is not ready: Pending"},
				},
				Sections: []workflowProgressSection{
					{Name: "Bootstrap", Status: "completed"},
					{Name: "Compute", Status: "running"},
				},
			},
		},
	}

	view := m.View()

	require.Contains(t, view, "omnistrate-ctl deploy")
	require.NotContains(t, view, "Global target")
	require.Contains(t, view, "aws")
	require.Contains(t, view, "us-east-1")
	require.Contains(t, view, "Progress")
	require.Contains(t, view, "database")
	require.Contains(t, view, "Bootstrap")
	require.Contains(t, view, "Compute")
	require.Contains(t, view, "Events")
	require.Contains(t, view, "Compute: BatchCreatePods")
	require.Contains(t, view, strings.Repeat(".", 30))
}

func TestWorkflowProgressViewRendersEventsUnderMatchingResource(t *testing.T) {
	m := newWorkflowProgressModel("inst-1", "create")
	m.loading = false
	m.events = map[string][]workflowProgressEvent{
		"r-db": {
			{ResourceID: "r-db", ResourceName: "database", Section: "Storage", Action: "CreateVolume", Message: "volume ready"},
		},
		"r-web": {
			{ResourceID: "r-web", ResourceName: "wordpress", Section: "Deployment", Action: "BatchCreatePods", Message: "pod is not ready: Pending"},
		},
	}
	m.snapshot = workflowProgressSnapshot{
		InstanceID:         "inst-1",
		WorkflowID:         "submit-create-1",
		WorkflowStatus:     "running",
		OverallPercent:     40,
		TotalResources:     2,
		CompletedResources: 0,
		Resources: []workflowProgressResource{
			{
				ID:             "r-db",
				Name:           "database",
				Status:         "running",
				Percent:        50,
				CompletedSteps: 2,
				TotalSteps:     6,
				Sections:       []workflowProgressSection{{Name: "Storage", Status: "running"}},
			},
			{
				ID:             "r-web",
				Name:           "wordpress",
				Status:         "running",
				Percent:        25,
				CompletedSteps: 1,
				TotalSteps:     6,
				Sections:       []workflowProgressSection{{Name: "Deployment", Status: "running"}},
			},
		},
	}

	view := m.View()

	dbIndex := strings.Index(view, "database")
	dbEventIndex := strings.Index(view, "Storage: CreateVolume")
	webIndex := strings.Index(view, "wordpress")
	webEventIndex := strings.Index(view, "Deployment: BatchCreatePods")
	require.NotEqual(t, -1, dbIndex)
	require.NotEqual(t, -1, dbEventIndex)
	require.NotEqual(t, -1, webIndex)
	require.NotEqual(t, -1, webEventIndex)
	require.Less(t, dbIndex, dbEventIndex)
	require.Less(t, dbEventIndex, webIndex)
	require.Less(t, webIndex, webEventIndex)
	require.NotContains(t, view, "volume ready")
	require.NotContains(t, view, "pod is not ready")
}

func TestWorkflowProgressViewPulsesRegionMarkerOnTick(t *testing.T) {
	model := newWorkflowProgressModel("inst-1", "create", "us-east-1")

	view := model.View()
	require.Contains(t, view, "◉")

	for i := 0; i < utils.RegionGlobePulseFrames; i++ {
		updated, _ := model.Update(bubbleSpinner.TickMsg{})
		model = updated.(workflowProgressModel)
	}
	view = model.View()

	require.Contains(t, view, "●")
	require.NotContains(t, view, "◉")
}

func TestWorkflowProgressEventMsgMaintainsSendMsgStyleWindow(t *testing.T) {
	model := newWorkflowProgressModel("inst-1", "create")

	updated, _ := model.Update(workflowProgressEventMsg{
		event: workflowProgressEvent{
			ResourceID: "r-db",
			Action:     "CreateVolume",
			Message:    "volume ready",
		},
	})
	model = updated.(workflowProgressModel)

	events := model.events["r-db"]
	require.Len(t, events, workflowProgressMaxEvents)
	for i := 0; i < workflowProgressMaxEvents-1; i++ {
		require.Empty(t, events[i].Action)
		require.Empty(t, events[i].Message)
	}
	require.Equal(t, "CreateVolume", events[workflowProgressMaxEvents-1].String())

	for i := 0; i < workflowProgressMaxEvents; i++ {
		updated, _ = model.Update(workflowProgressEventMsg{
			event: workflowProgressEvent{
				ResourceID: "r-db",
				Action:     fmt.Sprintf("Action-%d", i),
				Message:    fmt.Sprintf("event-%d", i),
			},
		})
		model = updated.(workflowProgressModel)
	}

	events = model.events["r-db"]
	require.Len(t, events, workflowProgressMaxEvents)
	require.Equal(t, "Action-0", events[0].String())
	require.Equal(t, "Action-4", events[workflowProgressMaxEvents-1].String())
}

func TestWorkflowProgressActionMessageParsesJSONPayload(t *testing.T) {
	event := dataaccess.DebugEvent{
		EventType: "WorkflowStepDebug",
		Message:   `{"action":"BatchCreatePods","actionStatus":"Running","message":"pod is not ready: Pending. Details: {\"podEvents\":[{\"message\":\"waiting\"}]}"}`,
	}

	action, status, message := workflowProgressActionMessage(event)

	require.Equal(t, "BatchCreatePods", action)
	require.Equal(t, "Running", status)
	require.Equal(t, "pod is not ready: Pending", message)
	require.NotContains(t, message, "podEvents")
}
