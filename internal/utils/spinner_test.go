package utils

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/stretchr/testify/require"
)

func TestParseSpinnerStepMessage(t *testing.T) {
	index, total, label, ok := parseSpinnerStepMessage("Step 2/2: Using cloud provider 'aws' and region 'us-east-1'")

	require.True(t, ok)
	require.Equal(t, 2, index)
	require.Equal(t, 2, total)
	require.Equal(t, "Using cloud provider 'aws' and region 'us-east-1'", label)
}

func TestSpinnerStepGroupsAttachDetailsToCurrentPhase(t *testing.T) {
	entries := []*spinnerEntry{
		{message: "Step 1/2: Checking cloud provider accounts...", state: spinnerComplete},
		{message: " - Using AWS Account ID: 123", state: spinnerComplete},
		{message: "Step 2/2: Preparing instance deployment", state: spinnerRunning},
	}

	groups, ok := spinnerStepGroups(entries)

	require.True(t, ok)
	require.Len(t, groups, 2)
	require.Equal(t, "Service creation", groups[0].title)
	require.Len(t, groups[0].entries, 2)
	require.Equal(t, "Using AWS Account ID: 123", groups[0].entries[1].label)
	require.True(t, groups[0].entries[1].detail)
	require.Equal(t, "Instance deployment", groups[1].title)
}

func TestGroupedDeploymentViewIncludesPhaseBars(t *testing.T) {
	mgr := &spinnerMgr{
		entries: []*spinnerEntry{
			{message: "Step 1/2: Service name resolved: app", state: spinnerComplete},
			{message: "Step 2/2: Preparing instance deployment", state: spinnerRunning},
		},
	}
	view, ok := spinnerModel{mgr: mgr, width: 100}.groupedDeploymentView(mgr.entries)

	require.True(t, ok)
	require.Contains(t, view, "omnistrate-ctl deploy")
	require.Contains(t, view, "Service creation")
	require.Contains(t, view, "Instance deployment")
	require.Contains(t, view, "Service name resolved: app")
}

func TestGroupedDeploymentViewWrapsLongLinesWithinFrame(t *testing.T) {
	mgr := &spinnerMgr{
		entries: []*spinnerEntry{
			{message: "Step 1/2: Checking cloud provider accounts...", state: spinnerComplete},
			{message: " - Using Azure Subscription ID: 4a66b749-4fd1-4367-a681-5deecf287e14 and Tenant ID: 4e6c839d-e141-462e-bc65-8cd863580351", state: spinnerComplete},
			{message: "Step 2/2: Instance creation submitted (ID: instance-tsxb27b54)", state: spinnerComplete},
		},
	}

	view, ok := spinnerModel{mgr: mgr, width: 96}.groupedDeploymentView(mgr.entries)

	require.True(t, ok)
	lines := strings.Split(view, "\n")
	maxWidth := lipgloss.Width(lines[0])
	for _, line := range lines {
		require.LessOrEqual(t, lipgloss.Width(line), maxWidth, line)
	}
	require.Contains(t, view, "4e6c839d-e141-462e-bc65-8cd863580351")
}

func TestGroupedDeploymentViewShowsSubmittedForInstanceCreation(t *testing.T) {
	mgr := &spinnerMgr{
		entries: []*spinnerEntry{
			{message: "Step 1/2: Built service 'app' in environment Prod (PROD), Service ID: s-1", state: spinnerComplete},
			{message: "Step 2/2: Preparing instance deployment", state: spinnerComplete},
			{message: "Step 2/2: Instance creation submitted (ID: instance-1)", state: spinnerComplete},
		},
	}

	view, ok := spinnerModel{mgr: mgr, width: 100}.groupedDeploymentView(mgr.entries)

	require.True(t, ok)
	require.Contains(t, view, "Instance deployment")
	require.Contains(t, view, "submitted")
	require.Contains(t, view, "Deployment workflow continues below.")
	require.Contains(t, view, "Submit")
	require.NotContains(t, view, "Instance deployment  complete")
}

func TestFinalGroupedDeploymentViewRendersCompleteFrame(t *testing.T) {
	mgr := &spinnerMgr{
		width: 96,
		entries: []*spinnerEntry{
			{message: "Step 1/2: Service name resolved: app", state: spinnerComplete},
			{message: "Step 2/2: Instance creation submitted (ID: instance-1)", state: spinnerComplete},
		},
	}

	view := mgr.finalGroupedDeploymentView()
	lines := strings.Split(view, "\n")

	require.NotEmpty(t, view)
	require.True(t, strings.HasPrefix(lines[0], "╭"), lines[0])
	require.True(t, strings.HasPrefix(lines[len(lines)-1], "╰"), lines[len(lines)-1])
}
