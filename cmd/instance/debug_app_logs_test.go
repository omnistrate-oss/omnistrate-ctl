package instance

import (
	"errors"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/dataaccess"
	"github.com/stretchr/testify/require"
)

func TestAppLogStreamsForResourceUsesResourceKey(t *testing.T) {
	data := DebugData{
		AppLogStreams: map[string][]dataaccess.LogsStream{
			"api": {
				{PodName: "api-1", LogsURL: "wss://logs.example/api-1"},
				{PodName: "api-0", LogsURL: "wss://logs.example/api-0"},
			},
		},
	}

	streams := appLogStreamsForResource(data, PlanDAGNode{ID: "r-api", Key: "api", Name: "API"})

	require.Len(t, streams, 2)
	require.Equal(t, "api-0", streams[0].PodName)
	require.Equal(t, "api-1", streams[1].PodName)
}

func TestAppLogStreamsForResourceMatchesCaseInsensitiveAlias(t *testing.T) {
	data := DebugData{
		AppLogStreams: map[string][]dataaccess.LogsStream{
			"API": {
				{PodName: "api-0"},
			},
		},
	}

	streams := appLogStreamsForResource(data, PlanDAGNode{Key: "api"})

	require.Len(t, streams, 1)
	require.Equal(t, "api-0", streams[0].PodName)
}

func TestFormatAppLogPayloadPrefixesPodName(t *testing.T) {
	lines := formatAppLogPayload("api-0", "started\nready\n")

	require.Equal(t, []string{"[api-0] started", "[api-0] ready"}, lines)
}

func TestRenderAppLogsTabShowsFeatureErrorWhenNoStreams(t *testing.T) {
	rendered := renderAppLogsTab(nil, "logs are not enabled for this instance", "", 20, 80)

	require.Contains(t, rendered, "logs are not enabled for this instance")
}

func TestAppLogsUnavailableMessageShowsSelectedAndDiscoveredKeys(t *testing.T) {
	data := DebugData{
		AppLogStreams: map[string][]dataaccess.LogsStream{
			"api": {
				{PodName: "api-0"},
			},
			"worker": {
				{PodName: "worker-0"},
			},
		},
	}

	msg := appLogsUnavailableMessage(data, PlanDAGNode{Key: "mysql-proxy", ID: "r-mysql", Name: "MySQL Proxy"})

	require.Contains(t, msg, "Selected resource: mysql-proxy, r-mysql, MySQL Proxy")
	require.Contains(t, msg, "Logs are available for other resource keys: api, worker")
	require.Contains(t, msg, "Select one of those resources")
}

func TestAppLogsUnavailableMessagePrefersDiscoveryError(t *testing.T) {
	msg := appLogsUnavailableMessage(DebugData{AppLogsError: "failed to list pods: Unauthorized"}, PlanDAGNode{Key: "api"})

	require.Equal(t, "failed to list pods: Unauthorized", msg)
}

func TestRenderAppLogsTabShowsLinesAndStatus(t *testing.T) {
	state := newAppLogsState([]dataaccess.LogsStream{{PodName: "api-0", LogsURL: "wss://logs.example/api-0"}})
	state.streaming = true
	state.lines = []string{"[api-0] started", "[api-0] ready"}

	rendered := renderAppLogsTab(state, "", "", 20, 80)

	require.Contains(t, rendered, "App Logs: api-0 (2 lines)")
	require.Contains(t, rendered, "[api-0] started")
	require.Contains(t, rendered, "live")
}

func TestAppLogPodStreamsForStreamsGroupsContainersByPod(t *testing.T) {
	pods := appLogPodStreamsForStreams([]dataaccess.LogsStream{
		{PodName: "api-1", ContainerName: "sidecar"},
		{PodName: "api-0", ContainerName: "worker"},
		{PodName: "api-0", ContainerName: "app"},
	})

	require.Len(t, pods, 2)
	require.Equal(t, "api-0", pods[0].name)
	require.Len(t, pods[0].streams, 2)
	require.Equal(t, "app", pods[0].streams[0].ContainerName)
	require.Equal(t, "worker", pods[0].streams[1].ContainerName)
	require.Equal(t, "api-1", pods[1].name)
}

func TestRenderAppLogsTabShowsPodSelector(t *testing.T) {
	state := newAppLogsState([]dataaccess.LogsStream{
		{PodName: "api-0"},
		{PodName: "api-1"},
	})
	state.streaming = true
	state.lines = []string{"[api-0] ready"}

	rendered := renderAppLogsTab(state, "", "", 20, 100)

	require.Contains(t, rendered, "Pods: use")
	require.Contains(t, rendered, "api-0")
	require.Contains(t, rendered, "api-1")
	require.Contains(t, rendered, "App Logs: api-0")
}

func TestAppLogsStateSwitchesSelectedPodAndResetsLines(t *testing.T) {
	state := newAppLogsState([]dataaccess.LogsStream{
		{PodName: "api-0"},
		{PodName: "api-1"},
	})
	state.lines = []string{"[api-0] ready"}
	state.streaming = true
	state.started = true

	cmd := state.selectNextPod()

	require.NotNil(t, cmd)
	require.Equal(t, 1, state.selectedPod)
	require.Equal(t, "api-1", state.pods[state.selectedPod].name)
	require.Empty(t, state.lines)
	require.True(t, state.follow)
}

func TestAppLogsStateIgnoresStaleStreamMessages(t *testing.T) {
	state := newAppLogsState([]dataaccess.LogsStream{{PodName: "api-0"}})
	state.generation = 2

	cmd := state.handleLine(appLogLineMsg{lines: []string{"stale"}, generation: 1}, 20, 80)
	state.handleDone(appLogStreamDoneMsg{generation: 1, err: errors.New("stale")})

	require.Nil(t, cmd)
	require.Empty(t, state.lines)
	require.False(t, state.done)
	require.NoError(t, state.err)
}

func TestComposeAppLogsTabReceivesLinesAndCopies(t *testing.T) {
	model := composeDetailModel{
		activeTab: composeTabAppLogs,
		width:     80,
		height:    20,
		appLogs: newAppLogsState([]dataaccess.LogsStream{
			{PodName: "api-0", LogsURL: "wss://logs.example/api-0"},
		}),
		wfErrors: &workflowErrorsState{},
	}

	updatedAny, cmd := model.Update(appLogLineMsg{lines: []string{"[api-0] ready"}})
	updated := updatedAny.(composeDetailModel)

	require.NotNil(t, cmd)
	require.Equal(t, "[api-0] ready", strings.TrimSpace(updated.composeCopyableContent()))
}

func TestComposeAppLogsFollowToggle(t *testing.T) {
	model := composeDetailModel{
		activeTab: composeTabAppLogs,
		width:     80,
		height:    20,
		appLogs:   newAppLogsState(nil),
		wfErrors:  &workflowErrorsState{},
	}

	updatedAny, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("f")})
	updated := updatedAny.(composeDetailModel)

	require.False(t, updated.appLogs.follow)
}
