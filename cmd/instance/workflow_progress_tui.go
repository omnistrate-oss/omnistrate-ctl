package instance

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/dataaccess"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/model"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/utils"
)

const (
	workflowProgressPollInterval = 10 * time.Second
	workflowProgressFinalPause   = 750 * time.Millisecond
	workflowProgressMaxEvents    = 5
)

var (
	workflowProgressFrameStyle = lipgloss.NewStyle().
					Border(lipgloss.RoundedBorder()).
					BorderForeground(lipgloss.Color("#334155")).
					Foreground(lipgloss.Color("#D1D5DB")).
					Padding(1, 2)
	workflowProgressTitleStyle = lipgloss.NewStyle().
					Bold(true).
					Foreground(lipgloss.Color("#F8FAFC"))
	workflowProgressCommandStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#9CA3AF"))
	workflowProgressMutedStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("#94A3B8"))
	workflowProgressSuccessStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#50C878")).Bold(true)
	workflowProgressRunningStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#7DD3FC")).Bold(true)
	workflowProgressPendingStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#64748B"))
	workflowProgressErrorStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("#F87171")).Bold(true)
	workflowProgressDotStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
)

type workflowProgressModel struct {
	instanceID     string
	actionType     string
	targetProvider string
	targetRegion   string

	spinner          spinner.Model
	overallProgress  progress.Model
	resourceProgress progress.Model

	width       int
	loading     bool
	pulseOn     bool
	pulseTicks  int
	pollCount   int
	snapshot    workflowProgressSnapshot
	events      map[string][]workflowProgressEvent
	err         error
	interrupted bool
}

type workflowProgressSnapshotMsg struct {
	snapshot workflowProgressSnapshot
}

type workflowProgressErrMsg struct {
	err error
}

type workflowProgressEventMsg struct {
	event workflowProgressEvent
}

type workflowProgressFinalPauseMsg struct{}

type workflowProgressSnapshot struct {
	InstanceID         string
	ActionType         string
	WorkflowID         string
	WorkflowStatus     string
	Resources          []workflowProgressResource
	OverallPercent     int
	CompletedResources int
	FailedResources    int
	TotalResources     int
	Done               bool
	Failed             bool
	FailureMessage     string
	FetchedAt          time.Time
	Events             []workflowProgressEvent
}

type workflowProgressResource struct {
	ID             string
	Key            string
	Name           string
	Status         string
	Percent        int
	CompletedSteps int
	TotalSteps     int
	Sections       []workflowProgressSection
	Events         []workflowProgressEvent
}

type workflowProgressSection struct {
	Name    string
	Status  string
	Message string
	Events  int
}

type workflowProgressEvent struct {
	Key          string
	ResourceID   string
	ResourceKey  string
	ResourceName string
	Section      string
	Action       string
	Message      string
	Status       string
	Time         string
}

func (event workflowProgressEvent) String() string {
	action := strings.TrimSpace(event.Action)
	section := strings.TrimSpace(event.Section)
	if action == "" {
		return workflowProgressDotStyle.Render(strings.Repeat(".", 30))
	}
	if section != "" {
		return fmt.Sprintf("%s: %s", section, action)
	}
	return action
}

func displayWorkflowResourceDataWithProgress(ctx context.Context, token, instanceID, actionType string, targetRegion ...string) error {
	searchRes, err := dataaccess.SearchInventory(ctx, token, fmt.Sprintf("resourceinstance:%s", instanceID))
	if err != nil {
		return err
	}
	if len(searchRes.ResourceInstanceResults) == 0 {
		return fmt.Errorf("instance not found")
	}

	instance := searchRes.ResourceInstanceResults[0]
	streamCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	program := tea.NewProgram(newWorkflowProgressModel(instanceID, actionType, targetRegion...))
	go streamWorkflowProgress(
		streamCtx,
		program,
		token,
		instance.ServiceId,
		instance.ServiceEnvironmentId,
		instanceID,
		actionType,
	)

	finalModel, err := program.Run()
	if err != nil {
		return err
	}

	model, ok := finalModel.(workflowProgressModel)
	if !ok {
		return nil
	}
	if model.err != nil {
		return model.err
	}
	if model.interrupted {
		return fmt.Errorf("workflow progress display interrupted")
	}
	if model.snapshot.Failed {
		if model.snapshot.FailureMessage != "" {
			return fmt.Errorf("%s", model.snapshot.FailureMessage)
		}
		return fmt.Errorf("with status: %s", model.snapshot.WorkflowStatus)
	}
	return nil
}

func newWorkflowProgressModel(instanceID, actionType string, targetRegion ...string) workflowProgressModel {
	spin := spinner.New()
	spin.Spinner = spinner.Dot
	spin.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("#50C878"))

	return workflowProgressModel{
		instanceID:       instanceID,
		actionType:       actionType,
		targetProvider:   firstWorkflowProgressProvider(targetRegion),
		targetRegion:     firstWorkflowProgressRegion(targetRegion),
		spinner:          spin,
		overallProgress:  progress.New(progress.WithSolidFill("#50C878"), progress.WithWidth(60)),
		resourceProgress: progress.New(progress.WithSolidFill("#50C878"), progress.WithWidth(36), progress.WithoutPercentage()),
		width:            96,
		loading:          true,
		pulseOn:          true,
		events:           make(map[string][]workflowProgressEvent),
	}
}

func (m workflowProgressModel) Init() tea.Cmd {
	return m.spinner.Tick
}

func (m workflowProgressModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc", "q":
			m.interrupted = true
			return m, tea.Quit
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		width := workflowProgressBarWidth(msg.Width)
		m.overallProgress.Width = width
		m.resourceProgress.Width = clampInt(width/2, 24, 44)
		return m, nil

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		m.pulseTicks++
		if m.pulseTicks >= utils.RegionGlobePulseFrames {
			m.pulseTicks = 0
			m.pulseOn = !m.pulseOn
		}
		if m.snapshot.Done || m.err != nil || m.interrupted {
			return m, nil
		}
		return m, cmd

	case progress.FrameMsg:
		updated, cmd := m.overallProgress.Update(msg)
		if progressModel, ok := updated.(progress.Model); ok {
			m.overallProgress = progressModel
		}
		return m, cmd

	case workflowProgressSnapshotMsg:
		m.loading = false
		m.pollCount++
		m.snapshot = msg.snapshot
		cmd := m.overallProgress.SetPercent(percentToFloat(msg.snapshot.OverallPercent))
		if msg.snapshot.Done {
			return m, tea.Sequence(cmd, workflowProgressFinalPauseCmd(), tea.Quit)
		}
		return m, cmd

	case workflowProgressErrMsg:
		m.err = msg.err
		return m, tea.Quit

	case workflowProgressEventMsg:
		resourceKey := workflowProgressEventResourceKey(msg.event)
		events := m.events[resourceKey]
		if len(events) == 0 {
			events = make([]workflowProgressEvent, workflowProgressMaxEvents)
		}
		m.events[resourceKey] = append(events[1:], msg.event)
		return m, nil
	}

	return m, nil
}

func (m workflowProgressModel) View() string {
	width := workflowProgressContentWidth(m.width)
	if m.err != nil {
		return workflowProgressFrameStyle.Width(width).Render(
			workflowProgressErrorStyle.Render("Deployment workflow failed")+"\n\n"+
				workflowProgressMutedStyle.Render(m.err.Error()),
		) + "\n"
	}

	var body strings.Builder
	body.WriteString(workflowProgressTitleStyle.Render("omnistrate-ctl deploy"))
	body.WriteString("\n\n")
	body.WriteString(workflowProgressCommandStyle.Render(fmt.Sprintf("$ tracking %s workflow for %s", workflowProgressActionLabel(m.actionType), m.instanceID)))
	body.WriteString("\n")
	if m.targetRegion != "" {
		body.WriteString("\n")
		body.WriteString(utils.RenderRegionGlobeWithProvider(m.targetProvider, m.targetRegion, clampInt(width-8, 42, 64), m.pulseOn || m.snapshot.Done))
		body.WriteString("\n")
	}

	if m.loading && m.pollCount == 0 {
		fmt.Fprintf(&body, "\n%s Resolving deployment workflow...\n", m.spinner.View())
		body.WriteString(workflowProgressMutedStyle.Render("Waiting for workflow events from Omnistrate."))
		return workflowProgressFrameStyle.Width(width).Render(body.String()) + "\n"
	}

	body.WriteString(m.renderOverview(width))
	body.WriteString("\n")
	body.WriteString(m.renderResources(width))

	return workflowProgressFrameStyle.Width(width).Render(body.String()) + "\n"
}

func firstWorkflowProgressRegion(regions []string) string {
	if len(regions) == 0 {
		return ""
	}
	if len(regions) >= 2 {
		return strings.TrimSpace(regions[1])
	}
	return strings.TrimSpace(regions[0])
}

func firstWorkflowProgressProvider(target []string) string {
	if len(target) < 2 {
		return ""
	}
	return strings.TrimSpace(target[0])
}

func (m workflowProgressModel) renderOverview(width int) string {
	bar := m.overallProgress
	bar.Width = workflowProgressBarWidth(width)

	status := workflowProgressNormalizeStatus(m.snapshot.WorkflowStatus)
	statusLine := workflowProgressStatusBadge(status)
	if m.snapshot.WorkflowID != "" {
		statusLine = fmt.Sprintf("%s  %s", statusLine, workflowProgressMutedStyle.Render(m.snapshot.WorkflowID))
	}

	summary := "Waiting for resource events"
	if m.snapshot.TotalResources > 0 {
		summary = fmt.Sprintf(
			"%d/%d resources complete",
			m.snapshot.CompletedResources,
			m.snapshot.TotalResources,
		)
		if m.snapshot.FailedResources > 0 {
			summary = fmt.Sprintf("%s, %d failed", summary, m.snapshot.FailedResources)
		}
	}

	return strings.Join([]string{
		"",
		fmt.Sprintf("Workflow   %s", statusLine),
		fmt.Sprintf("Progress   %s", bar.ViewAs(percentToFloat(m.snapshot.OverallPercent))),
		fmt.Sprintf("Summary    %s", workflowProgressMutedStyle.Render(summary)),
	}, "\n")
}

func (m workflowProgressModel) renderResources(width int) string {
	if len(m.snapshot.Resources) == 0 {
		return fmt.Sprintf("%s Waiting for workflow resource sections...", m.spinner.View())
	}

	resourceBar := m.resourceProgress
	resourceBar.Width = clampInt(workflowProgressBarWidth(width)/2, 24, 44)
	sectionWidth := clampInt(width-8, 40, 108)

	lines := []string{workflowProgressMutedStyle.Render("Resources")}
	for _, resource := range m.snapshot.Resources {
		name := workflowProgressResourceName(resource)
		header := fmt.Sprintf(
			"%s %s  %s  %s",
			workflowProgressIcon(resource.Status, m.spinner.View()),
			name,
			workflowProgressStatusBadge(resource.Status),
			workflowProgressMutedStyle.Render(fmt.Sprintf("%d/%d sections", resource.CompletedSteps, resource.TotalSteps)),
		)
		lines = append(lines, header)
		lines = append(lines, fmt.Sprintf("  %s", resourceBar.ViewAs(percentToFloat(resource.Percent))))
		lines = append(lines, "  "+workflowProgressWrapTokens(workflowProgressSectionTokens(resource.Sections, m.spinner.View()), sectionWidth))
		lines = append(lines, m.renderResourceEvents(resource, width)...)
		lines = append(lines, "")
	}

	return strings.TrimRight(strings.Join(lines, "\n"), "\n")
}

func (m workflowProgressModel) renderResourceEvents(resource workflowProgressResource, width int) []string {
	events := m.events[workflowProgressResourceEventKey(resource)]
	if len(events) == 0 {
		events = workflowProgressLastEvents(resource.Events, workflowProgressMaxEvents)
	}
	if len(events) == 0 && workflowProgressNormalizeStatus(resource.Status) != "running" {
		return nil
	}

	events = workflowProgressEventWindow(events, workflowProgressMaxEvents)
	eventWidth := clampInt(width-14, 40, 102)
	lines := []string{"    " + workflowProgressMutedStyle.Render("Events")}
	for _, event := range events {
		lines = append(lines, fmt.Sprintf("      %s", workflowProgressTruncate(event.String(), eventWidth)))
	}
	return lines
}

func streamWorkflowProgress(ctx context.Context, program *tea.Program, token, serviceID, environmentID, instanceID, actionType string) {
	seenEvents := make(map[string]bool)
	delay := time.Duration(0)

	for {
		if delay > 0 {
			timer := time.NewTimer(delay)
			select {
			case <-ctx.Done():
				timer.Stop()
				return
			case <-timer.C:
			}
		}

		snapshot, err := buildWorkflowProgressSnapshot(ctx, token, serviceID, environmentID, instanceID, actionType)
		if err != nil {
			program.Send(workflowProgressErrMsg{err: err})
			return
		}

		for _, event := range snapshot.Events {
			if seenEvents[event.Key] {
				continue
			}
			seenEvents[event.Key] = true
			program.Send(workflowProgressEventMsg{event: event})
		}
		program.Send(workflowProgressSnapshotMsg{snapshot: snapshot})

		if snapshot.Done {
			return
		}
		delay = workflowProgressPollInterval
	}
}

func workflowProgressFinalPauseCmd() tea.Cmd {
	return tea.Tick(workflowProgressFinalPause, func(time.Time) tea.Msg {
		return workflowProgressFinalPauseMsg{}
	})
}

func buildWorkflowProgressSnapshot(ctx context.Context, token, serviceID, environmentID, instanceID, actionType string) (workflowProgressSnapshot, error) {
	resourcesData, workflowInfo, err := dataaccess.GetDebugEventsForAllResources(
		ctx,
		token,
		serviceID,
		environmentID,
		instanceID,
		true,
		actionType,
	)
	if err != nil {
		return workflowProgressSnapshot{}, err
	}

	snapshot := workflowProgressSnapshot{
		InstanceID: instanceID,
		ActionType: actionType,
		FetchedAt:  time.Now(),
	}
	if workflowInfo != nil {
		snapshot.WorkflowID = workflowInfo.WorkflowID
		snapshot.WorkflowStatus = workflowInfo.WorkflowStatus
	}

	for _, resourceData := range resourcesData {
		resource := buildWorkflowProgressResource(resourceData, workflowInfo)
		snapshot.Resources = append(snapshot.Resources, resource)
		snapshot.Events = append(snapshot.Events, resource.Events...)
		snapshot.TotalResources++
		switch resource.Status {
		case "completed":
			snapshot.CompletedResources++
		case "failed":
			snapshot.FailedResources++
		}
	}
	sort.Slice(snapshot.Events, func(i, j int) bool {
		return snapshot.Events[i].Key < snapshot.Events[j].Key
	})

	sort.Slice(snapshot.Resources, func(i, j int) bool {
		left := workflowProgressResourceName(snapshot.Resources[i])
		right := workflowProgressResourceName(snapshot.Resources[j])
		if left == right {
			return snapshot.Resources[i].ID < snapshot.Resources[j].ID
		}
		return left < right
	})

	if workflowProgressNormalizeStatus(snapshot.WorkflowStatus) == "pending" && len(snapshot.Resources) > 0 {
		snapshot.WorkflowStatus = workflowProgressStatusFromResources(snapshot.Resources)
	}
	snapshot.OverallPercent = workflowProgressOverallPercent(snapshot.Resources, snapshot.WorkflowStatus)
	snapshot.Failed = workflowProgressIsFailed(snapshot.WorkflowStatus) || snapshot.FailedResources > 0
	snapshot.Done = workflowProgressIsTerminal(snapshot.WorkflowStatus) || snapshot.Failed
	if !snapshot.Done && snapshot.TotalResources > 0 && snapshot.CompletedResources == snapshot.TotalResources {
		snapshot.Done = true
	}
	if snapshot.Done && !snapshot.Failed && snapshot.OverallPercent < 100 {
		snapshot.OverallPercent = 100
	}
	if snapshot.Failed {
		snapshot.FailureMessage = workflowProgressFailureMessage(snapshot)
	}

	return snapshot, nil
}

func buildWorkflowProgressResource(resourceData dataaccess.ResourceWorkflowDebugEvents, workflowInfo *dataaccess.WorkflowInfo) workflowProgressResource {
	statusSource := safeStatus(resourceData.WorkflowStatus, workflowInfo)
	workflowStatus := workflowProgressNormalizeStatus(statusSource)
	sections := buildWorkflowProgressSections(resourceData.EventsByWorkflowStep, workflowStatus)
	completed, total, running, failed := workflowProgressSectionCounts(sections)

	status := workflowProgressStatusFromSectionCounts(completed, total, running, failed)
	if status == "pending" && workflowStatus != "pending" {
		status = workflowStatus
	}

	percent := workflowProgressResourcePercent(status, completed, total, running)
	events := workflowProgressEventsForResource(resourceData)
	return workflowProgressResource{
		ID:             resourceData.ResourceID,
		Key:            resourceData.ResourceKey,
		Name:           resourceData.ResourceName,
		Status:         status,
		Percent:        percent,
		CompletedSteps: completed,
		TotalSteps:     total,
		Sections:       sections,
		Events:         events,
	}
}

func buildWorkflowProgressSections(events *dataaccess.DebugEventsByWorkflowSteps, workflowStatus string) []workflowProgressSection {
	sections := []workflowProgressSection{
		{Name: "Bootstrap", Status: "pending"},
		{Name: "Storage", Status: "pending"},
		{Name: "Network", Status: "pending"},
		{Name: "Compute", Status: "pending"},
		{Name: "Deployment", Status: "pending"},
		{Name: "Monitoring", Status: "pending"},
	}

	if events != nil {
		sectionEvents := [][]dataaccess.DebugEvent{
			events.Bootstrap,
			events.Storage,
			events.Network,
			events.Compute,
			events.Deployment,
			events.Monitoring,
		}
		for i := range sectionEvents {
			sections[i].Status = workflowProgressStatusFromEvents(sectionEvents[i])
			sections[i].Events = len(sectionEvents[i])
			sections[i].Message = workflowProgressLatestEventMessage(sectionEvents[i])
		}
		if len(events.Unknown) > 0 {
			sections = append(sections, workflowProgressSection{
				Name:    "Other",
				Status:  workflowProgressStatusFromEvents(events.Unknown),
				Events:  len(events.Unknown),
				Message: workflowProgressLatestEventMessage(events.Unknown),
			})
		}
	}

	switch workflowProgressNormalizeStatus(workflowStatus) {
	case "completed":
		for i := range sections {
			if sections[i].Status != "failed" {
				sections[i].Status = "completed"
			}
		}
	case "failed":
		if !workflowProgressHasSectionStatus(sections, "failed") {
			idx := workflowProgressLastActiveSectionIndex(sections)
			if idx >= 0 {
				sections[idx].Status = "failed"
			}
		}
	}

	return sections
}

func workflowProgressStatusFromEvents(events []dataaccess.DebugEvent) string {
	if len(events) == 0 {
		return "pending"
	}
	switch model.WorkflowStepEventType(getHighestPriorityEventType(events)) {
	case model.WorkflowStepFailed:
		return "failed"
	case model.WorkflowStepCompleted:
		return "completed"
	case model.WorkflowStepDebug, model.WorkflowStepStarted:
		return "running"
	default:
		return "pending"
	}
}

func workflowProgressSectionCounts(sections []workflowProgressSection) (completed, total, running, failed int) {
	total = len(sections)
	for _, section := range sections {
		switch section.Status {
		case "completed":
			completed++
		case "running":
			running++
		case "failed":
			failed++
		}
	}
	return completed, total, running, failed
}

func workflowProgressStatusFromSectionCounts(completed, total, running, failed int) string {
	switch {
	case failed > 0:
		return "failed"
	case total > 0 && completed == total:
		return "completed"
	case running > 0 || completed > 0:
		return "running"
	default:
		return "pending"
	}
}

func workflowProgressResourcePercent(status string, completed, total, running int) int {
	if total == 0 {
		switch status {
		case "completed", "failed":
			return 100
		case "running":
			return 5
		default:
			return 0
		}
	}
	if status == "completed" {
		return 100
	}

	weighted := float64(completed)
	if running > 0 {
		weighted += 0.5
	}
	percent := int((weighted / float64(total) * 100) + 0.5)
	if status == "running" && percent == 0 {
		percent = 5
	}
	return clampInt(percent, 0, 100)
}

func workflowProgressOverallPercent(resources []workflowProgressResource, workflowStatus string) int {
	status := workflowProgressNormalizeStatus(workflowStatus)
	if len(resources) == 0 {
		switch status {
		case "completed", "failed":
			return 100
		case "running":
			return 10
		default:
			return 0
		}
	}

	total := 0
	for _, resource := range resources {
		total += resource.Percent
	}
	percent := total / len(resources)
	if status == "completed" {
		return 100
	}
	if status == "running" && percent == 0 {
		return 5
	}
	return clampInt(percent, 0, 100)
}

func workflowProgressStatusFromResources(resources []workflowProgressResource) string {
	if len(resources) == 0 {
		return "pending"
	}

	completed := 0
	running := false
	for _, resource := range resources {
		switch resource.Status {
		case "failed":
			return "failed"
		case "completed":
			completed++
		case "running":
			running = true
		}
	}
	if completed == len(resources) {
		return "completed"
	}
	if running || completed > 0 {
		return "running"
	}
	return "pending"
}

func workflowProgressNormalizeStatus(status string) string {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "success", "succeeded", "complete", "completed":
		return "completed"
	case "failed", "failure", "cancelled", "canceled", "error":
		return "failed"
	case "running", "in_progress", "in-progress", "started", "debug":
		return "running"
	default:
		return "pending"
	}
}

func workflowProgressIsTerminal(status string) bool {
	normalized := workflowProgressNormalizeStatus(status)
	return normalized == "completed" || normalized == "failed"
}

func workflowProgressIsFailed(status string) bool {
	return workflowProgressNormalizeStatus(status) == "failed"
}

func workflowProgressFailureMessage(snapshot workflowProgressSnapshot) string {
	for _, resource := range snapshot.Resources {
		if resource.Status == "failed" {
			return fmt.Sprintf("for resource %s", workflowProgressResourceName(resource))
		}
	}
	if snapshot.WorkflowStatus != "" {
		return fmt.Sprintf("with status: %s", snapshot.WorkflowStatus)
	}
	return "workflow failed"
}

func workflowProgressLatestEventMessage(events []dataaccess.DebugEvent) string {
	if len(events) == 0 {
		return ""
	}

	var latest dataaccess.DebugEvent
	for _, event := range events {
		if event.Message == "" {
			continue
		}
		if latest.Message == "" || event.EventTime >= latest.EventTime {
			latest = event
		}
	}
	return latest.Message
}

func workflowProgressEventsForResource(resource dataaccess.ResourceWorkflowDebugEvents) []workflowProgressEvent {
	if resource.EventsByWorkflowStep == nil {
		return nil
	}

	sections := []struct {
		name   string
		events []dataaccess.DebugEvent
	}{
		{name: "Bootstrap", events: resource.EventsByWorkflowStep.Bootstrap},
		{name: "Storage", events: resource.EventsByWorkflowStep.Storage},
		{name: "Network", events: resource.EventsByWorkflowStep.Network},
		{name: "Compute", events: resource.EventsByWorkflowStep.Compute},
		{name: "Deployment", events: resource.EventsByWorkflowStep.Deployment},
		{name: "Monitoring", events: resource.EventsByWorkflowStep.Monitoring},
		{name: "Other", events: resource.EventsByWorkflowStep.Unknown},
	}

	var events []workflowProgressEvent
	for _, section := range sections {
		for _, event := range section.events {
			events = append(events, workflowProgressEventFromDebugEvent(resource, section.name, event))
		}
	}
	sort.Slice(events, func(i, j int) bool {
		return events[i].Key < events[j].Key
	})
	return events
}

func workflowProgressEventFromDebugEvent(resource dataaccess.ResourceWorkflowDebugEvents, sectionName string, event dataaccess.DebugEvent) workflowProgressEvent {
	action, status, message := workflowProgressActionMessage(event)
	key := strings.Join([]string{
		event.EventTime,
		resource.ResourceID,
		resource.ResourceKey,
		resource.ResourceName,
		sectionName,
		event.EventType,
		event.Message,
	}, "|")
	return workflowProgressEvent{
		Key:          key,
		ResourceID:   resource.ResourceID,
		ResourceKey:  resource.ResourceKey,
		ResourceName: resource.ResourceName,
		Section:      sectionName,
		Action:       action,
		Message:      message,
		Status:       status,
		Time:         event.EventTime,
	}
}

func workflowProgressActionMessage(event dataaccess.DebugEvent) (string, string, string) {
	type actionPayload struct {
		Action       string `json:"action"`
		ActionStatus string `json:"actionStatus"`
		Message      string `json:"message"`
	}

	action := strings.TrimSpace(event.EventType)
	status := ""
	message := strings.TrimSpace(event.Message)

	var payload actionPayload
	if err := json.Unmarshal([]byte(message), &payload); err == nil {
		if strings.TrimSpace(payload.Action) != "" {
			action = strings.TrimSpace(payload.Action)
		}
		status = strings.TrimSpace(payload.ActionStatus)
		if strings.TrimSpace(payload.Message) != "" {
			message = strings.TrimSpace(payload.Message)
		}
	}

	if action == "" {
		action = "WorkflowEvent"
	}
	message = workflowProgressCleanEventMessage(message)
	return action, status, message
}

func workflowProgressCleanEventMessage(message string) string {
	message = strings.TrimSpace(message)
	if message == "" {
		return "event received"
	}

	detailMarkers := []string{". Details:", " Details:", " details:", ". details:"}
	for _, marker := range detailMarkers {
		if idx := strings.Index(message, marker); idx >= 0 {
			message = strings.TrimSpace(message[:idx])
			break
		}
	}

	return strings.TrimSpace(strings.TrimSuffix(message, "."))
}

func workflowProgressHasSectionStatus(sections []workflowProgressSection, status string) bool {
	for _, section := range sections {
		if section.Status == status {
			return true
		}
	}
	return false
}

func workflowProgressLastActiveSectionIndex(sections []workflowProgressSection) int {
	for i := len(sections) - 1; i >= 0; i-- {
		if sections[i].Status != "pending" {
			return i
		}
	}
	if len(sections) == 0 {
		return -1
	}
	return 0
}

func workflowProgressSectionTokens(sections []workflowProgressSection, spinnerView string) []string {
	tokens := make([]string, 0, len(sections))
	for _, section := range sections {
		tokens = append(tokens, fmt.Sprintf("%s %s", workflowProgressIcon(section.Status, spinnerView), section.Name))
	}
	return tokens
}

func workflowProgressWrapTokens(tokens []string, maxWidth int) string {
	if len(tokens) == 0 {
		return ""
	}

	var lines []string
	current := ""
	for _, token := range tokens {
		if current == "" {
			current = token
			continue
		}
		next := current + "  " + token
		if lipgloss.Width(next) > maxWidth {
			lines = append(lines, current)
			current = token
			continue
		}
		current = next
	}
	if current != "" {
		lines = append(lines, current)
	}
	return strings.Join(lines, "\n  ")
}

func workflowProgressIcon(status, spinnerView string) string {
	switch workflowProgressNormalizeStatus(status) {
	case "completed":
		return workflowProgressSuccessStyle.Render("✓")
	case "failed":
		return workflowProgressErrorStyle.Render("✗")
	case "running":
		if spinnerView != "" {
			return workflowProgressRunningStyle.Render(spinnerView)
		}
		return workflowProgressRunningStyle.Render("●")
	default:
		return workflowProgressPendingStyle.Render("·")
	}
}

func workflowProgressStatusBadge(status string) string {
	switch workflowProgressNormalizeStatus(status) {
	case "completed":
		return workflowProgressSuccessStyle.Render("completed")
	case "failed":
		return workflowProgressErrorStyle.Render("failed")
	case "running":
		return workflowProgressRunningStyle.Render("running")
	default:
		return workflowProgressPendingStyle.Render("pending")
	}
}

func workflowProgressActionLabel(actionType string) string {
	if actionType == "" {
		return "deployment"
	}
	return strings.ToLower(actionType)
}

func workflowProgressResourceName(resource workflowProgressResource) string {
	switch {
	case resource.Name != "":
		return resource.Name
	case resource.Key != "":
		return resource.Key
	case resource.ID != "":
		return resource.ID
	default:
		return "resource"
	}
}

func workflowProgressResourceEventKey(resource workflowProgressResource) string {
	switch {
	case resource.ID != "":
		return resource.ID
	case resource.Key != "":
		return resource.Key
	case resource.Name != "":
		return resource.Name
	default:
		return "resource"
	}
}

func workflowProgressEventResourceKey(event workflowProgressEvent) string {
	switch {
	case event.ResourceID != "":
		return event.ResourceID
	case event.ResourceKey != "":
		return event.ResourceKey
	case event.ResourceName != "":
		return event.ResourceName
	default:
		return "resource"
	}
}

func workflowProgressLastEvents(events []workflowProgressEvent, maxEvents int) []workflowProgressEvent {
	if maxEvents <= 0 || len(events) <= maxEvents {
		return events
	}
	return events[len(events)-maxEvents:]
}

func workflowProgressEventWindow(events []workflowProgressEvent, maxEvents int) []workflowProgressEvent {
	if maxEvents <= 0 {
		return nil
	}
	window := make([]workflowProgressEvent, maxEvents)
	if len(events) >= maxEvents {
		copy(window, events[len(events)-maxEvents:])
		return window
	}
	copy(window[maxEvents-len(events):], events)
	return window
}

func workflowProgressContentWidth(width int) int {
	if width <= 0 {
		width = 96
	}
	return clampInt(width-6, 64, 120)
}

func workflowProgressBarWidth(width int) int {
	if width <= 0 {
		width = 96
	}
	return clampInt(width-30, 28, 72)
}

func workflowProgressTruncate(value string, maxWidth int) string {
	if lipgloss.Width(value) <= maxWidth {
		return value
	}
	if maxWidth <= 1 {
		return ""
	}
	runes := []rune(value)
	for len(runes) > 0 && lipgloss.Width(string(runes))+1 > maxWidth {
		runes = runes[:len(runes)-1]
	}
	return string(runes) + "…"
}

func percentToFloat(percent int) float64 {
	return float64(clampInt(percent, 0, 100)) / 100
}
