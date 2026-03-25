package instance

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/dataaccess"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/model"
)

// workflowErrorsState holds the scroll and selection state for the workflow events tab.
type workflowErrorsState struct {
	scroll      int
	cursor      int
	modalText   string // non-empty means event detail modal is open
	modalTitle  string
	modalScroll int
	refreshing  bool      // true while fetching fresh workflow events
	lastRefresh time.Time // when the last successful refresh completed
}

// wfEventsRefreshMsg carries refreshed workflow steps for a resource.
type wfEventsRefreshMsg struct {
	steps *ResourceWorkflowSteps
	err   error
}

// wfCountdownTickMsg fires every second to update the countdown display.
type wfCountdownTickMsg struct{}

const wfEventsRefreshInterval = 5 * time.Second

// isWorkflowInProgress returns true if any step is still in-progress or pending.
func isWorkflowInProgress(steps *ResourceWorkflowSteps) bool {
	if steps == nil || len(steps.Steps) == 0 {
		return false
	}
	for _, s := range steps.Steps {
		if s.Status == "in-progress" || s.Status == "pending" {
			return true
		}
	}
	return false
}

// renderLiveIndicator renders a "● LIVE  Refreshing..." or "● LIVE  Next refresh in Xs" line.
func renderLiveIndicator(spinnerView string, refreshing bool, lastRefresh time.Time, interval ...time.Duration) string {
	liveStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("82"))
	dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("245"))

	refreshInterval := wfEventsRefreshInterval
	if len(interval) > 0 {
		refreshInterval = interval[0]
	}

	live := liveStyle.Render("● LIVE")
	if refreshing {
		return fmt.Sprintf("  %s  %s %s", live, spinnerView, dimStyle.Render("Refreshing…"))
	}
	if lastRefresh.IsZero() {
		return fmt.Sprintf("  %s", live)
	}
	elapsed := time.Since(lastRefresh)
	remaining := refreshInterval - elapsed
	if remaining < 0 {
		remaining = 0
	}
	secs := int(remaining.Seconds())
	if secs <= 0 {
		return fmt.Sprintf("  %s  %s %s", live, spinnerView, dimStyle.Render("Refreshing…"))
	}
	return fmt.Sprintf("  %s  %s", live, dimStyle.Render(fmt.Sprintf("Next refresh in %ds", secs)))
}

func scheduleWfEventsRefresh() tea.Cmd {
	return tea.Tick(wfEventsRefreshInterval, func(time.Time) tea.Msg {
		return wfEventsRefreshTickMsg{}
	})
}

func scheduleWfCountdownTick() tea.Cmd {
	return tea.Tick(1*time.Second, func(time.Time) tea.Msg {
		return wfCountdownTickMsg{}
	})
}

// wfEventsRefreshTickMsg triggers a workflow events data refresh.
type wfEventsRefreshTickMsg struct{}

// fetchWfEventsForResource fetches fresh workflow events for a specific resource key.
func fetchWfEventsForResource(data DebugData, resourceKey string) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		resourcesData, _, err := dataaccess.GetDebugEventsForAllResources(
			ctx, data.Token, data.ServiceID, data.EnvironmentID, data.InstanceID, true,
		)
		if err != nil {
			return wfEventsRefreshMsg{err: err}
		}
		for _, resource := range resourcesData {
			if resource.ResourceKey == resourceKey {
				steps := buildStepsFromRawSteps(resource.RawSteps)
				return wfEventsRefreshMsg{steps: steps}
			}
		}
		return wfEventsRefreshMsg{}
	}
}

// WorkflowStepInfo holds step-level summary with timing and events.
type WorkflowStepInfo struct {
	Name         string                  `json:"name"`
	DisplayName  string                  `json:"displayName,omitempty"` // overridden display name (e.g. "Waiting for dependencies" for Bootstrap)
	Status       string                  `json:"status"`                // "success", "in-progress", "failed", "pending"
	StartTime    string                  `json:"startTime,omitempty"`
	EndTime      string                  `json:"endTime,omitempty"`
	Events       []dataaccess.DebugEvent `json:"events,omitempty"`
	DepTimelines []depTimeline           `json:"depTimelines,omitempty"` // populated for bootstrap steps only
}

// depTimeline holds the completion status of a dependency resource for Bootstrap step rendering.
type depTimeline struct {
	Name       string `json:"name"`                 // dependency resource name or key
	Status     string `json:"status"`               // overall status: "completed", "running", "pending", "failed"
	FinishedAt string `json:"finishedAt,omitempty"` // RFC3339 time when the dependency finished (empty if not done)
}

// stepDisplayName returns the display name for the step (using override if set).
func (s WorkflowStepInfo) stepDisplayName() string {
	if s.DisplayName != "" {
		return s.DisplayName
	}
	return s.Name
}

// ResourceWorkflowSteps holds the ordered list of workflow steps for a resource.
type ResourceWorkflowSteps struct {
	Steps []WorkflowStepInfo `json:"steps"`
}

// wfEventItem represents a selectable row in the workflow events tab.
type wfEventItem struct {
	isStepHeader bool                   // true if this is a step header row
	stepIdx      int                    // index into steps
	event        *dataaccess.DebugEvent // non-nil for event rows
}

// flattenWfEventItems builds a flat list of selectable items from steps.
// It mirrors the rendering order in renderTimelineView.
func flattenWfEventItems(steps *ResourceWorkflowSteps) []wfEventItem {
	if steps == nil || len(steps.Steps) == 0 {
		return nil
	}
	var items []wfEventItem
	for i, step := range steps.Steps {
		// Filter same as renderTimelineView
		var actionEvents []dataaccess.DebugEvent
		for _, evt := range step.Events {
			et := model.WorkflowStepEventType(evt.EventType)
			if et == model.WorkflowStepStarted || et == model.WorkflowStepCompleted {
				if !eventHasAction(evt.Message) {
					continue
				}
			}
			actionEvents = append(actionEvents, evt)
		}
		if len(actionEvents) == 0 && (!isBootstrapStep(step.Name) || len(step.DepTimelines) == 0) {
			continue
		}
		items = append(items, wfEventItem{isStepHeader: true, stepIdx: i})
		for j := range actionEvents {
			items = append(items, wfEventItem{stepIdx: i, event: &actionEvents[j]})
		}
	}
	return items
}

// buildStepsFromRawSteps builds step summaries from the raw API step data.
func buildStepsFromRawSteps(rawSteps []dataaccess.RawWorkflowStep) *ResourceWorkflowSteps {
	if len(rawSteps) == 0 {
		return nil
	}

	var result []WorkflowStepInfo
	for _, s := range rawSteps {
		info := WorkflowStepInfo{
			Name:   s.StepName,
			Events: s.Events,
		}

		if len(s.Events) > 0 {
			info.StartTime = s.Events[0].EventTime
			info.EndTime = s.Events[len(s.Events)-1].EventTime
			info.Status = determineStepStatusFromEvents(s.Events)
		} else {
			info.Status = "pending"
		}

		result = append(result, info)
	}

	return &ResourceWorkflowSteps{Steps: result}
}

// isBootstrapStep returns true if the step name corresponds to a bootstrap/dependency-wait step.
func isBootstrapStep(name string) bool {
	return strings.Contains(strings.ToLower(name), "bootstrap")
}

// enrichBootstrapSteps renames Bootstrap steps to "Waiting for dependencies" and populates
// their DepTimelines from the PlanDAG dependency graph and workflow events of peer resources.
func enrichBootstrapSteps(steps *ResourceWorkflowSteps, resourceKey string, dag *PlanDAG) {
	if steps == nil || dag == nil {
		return
	}

	// Find the resource ID for this key
	var resourceID string
	for id, node := range dag.Nodes {
		if node.Key == resourceKey {
			resourceID = id
			break
		}
	}
	if resourceID == "" {
		// Still rename even if we can't find deps
		for i := range steps.Steps {
			if isBootstrapStep(steps.Steps[i].Name) {
				steps.Steps[i].DisplayName = "Waiting for dependencies"
			}
		}
		return
	}

	// Find parent dependencies (edges where To == resourceID)
	var depIDs []string
	for _, edge := range dag.Edges {
		if edge.To == resourceID {
			depIDs = append(depIDs, edge.From)
		}
	}

	for i := range steps.Steps {
		if !isBootstrapStep(steps.Steps[i].Name) {
			continue
		}
		steps.Steps[i].DisplayName = "Waiting for dependencies"

		if len(depIDs) == 0 {
			continue
		}

		var timelines []depTimeline
		for _, depID := range depIDs {
			depNode, ok := dag.Nodes[depID]
			if !ok {
				continue
			}
			name := depNode.Name
			if name == "" {
				name = depNode.Key
			}
			if name == "" {
				name = depID
			}

			dt := depTimeline{Name: name, Status: "pending"}

			// Look up the dependency's workflow events to determine status and finish time
			if dag.WorkflowStepsByKey != nil {
				depSteps := dag.WorkflowStepsByKey[depNode.Key]
				if depSteps != nil && len(depSteps.Steps) > 0 {
					allDone := true
					hasFailed := false
					hasRunning := false
					latestEnd := ""
					for _, ds := range depSteps.Steps {
						switch ds.Status {
						case "failed":
							hasFailed = true
						case "in-progress", "pending":
							allDone = false
							if ds.Status == "in-progress" {
								hasRunning = true
							}
						}
						if ds.EndTime != "" && ds.EndTime > latestEnd {
							latestEnd = ds.EndTime
						}
					}
					if hasFailed {
						dt.Status = "failed"
						dt.FinishedAt = latestEnd
					} else if allDone {
						dt.Status = "completed"
						dt.FinishedAt = latestEnd
					} else if hasRunning {
						dt.Status = "running"
					}
				}
			}

			// Also check ProgressByKey for status
			if dag.ProgressByKey != nil {
				if prog, ok := dag.ProgressByKey[depNode.Key]; ok {
					if dt.Status == "pending" {
						dt.Status = prog.Status
					}
				}
			}

			timelines = append(timelines, dt)
		}
		steps.Steps[i].DepTimelines = timelines
	}
}

func determineStepStatusFromEvents(events []dataaccess.DebugEvent) string {
	hasFailed := false
	hasCompleted := false
	hasStarted := false

	for _, e := range events {
		switch model.WorkflowStepEventType(e.EventType) {
		case model.WorkflowStepFailed:
			hasFailed = true
		case model.WorkflowStepCompleted:
			hasCompleted = true
		case model.WorkflowStepStarted, model.WorkflowStepDebug:
			hasStarted = true
		}
	}

	if hasFailed {
		return "failed"
	}
	if hasCompleted {
		return "success"
	}
	if hasStarted {
		return "in-progress"
	}
	return "pending"
}

// renderWorkflowEventsTab renders the workflow events tab content.
func renderWorkflowEventsTab(steps *ResourceWorkflowSteps, state *workflowErrorsState, bodyHeight, contentWidth int, loading bool, spinnerView string, isLive bool) string {
	if loading && (steps == nil || len(steps.Steps) == 0) {
		return fmt.Sprintf("\n  %s Fetching workflow events...", spinnerView)
	}
	if steps == nil || len(steps.Steps) == 0 {
		subtleStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
		return fmt.Sprintf("\n  %s\n", subtleStyle.Render("No workflow events available for this resource."))
	}

	items := flattenWfEventItems(steps)
	rendered, cursorLine := renderTimelineView(steps, contentWidth, items, state.cursor)

	// Prepend live indicator when workflow is in progress
	liveOffset := 0
	if isLive {
		indicator := renderLiveIndicator(spinnerView, state.refreshing, state.lastRefresh)
		rendered = append([]string{indicator, ""}, rendered...)
		liveOffset = 2
		if cursorLine >= 0 {
			cursorLine += liveOffset
		}
	}

	totalLines := len(rendered)

	var b strings.Builder

	viewH := bodyHeight - 2
	if viewH < 1 {
		viewH = 1
	}

	maxScroll := totalLines - viewH
	if maxScroll < 0 {
		maxScroll = 0
	}

	// Auto-scroll to keep cursor visible
	if cursorLine >= 0 {
		if cursorLine < state.scroll {
			state.scroll = cursorLine
		} else if cursorLine >= state.scroll+viewH {
			state.scroll = cursorLine - viewH + 1
		}
	}
	if state.scroll > maxScroll {
		state.scroll = maxScroll
	}
	if state.scroll < 0 {
		state.scroll = 0
	}

	scroll := state.scroll

	end := scroll + viewH
	if end > totalLines {
		end = totalLines
	}

	for i := scroll; i < end; i++ {
		b.WriteString(rendered[i])
		b.WriteString("\n")
	}

	for i := end - scroll; i < viewH; i++ {
		b.WriteString("\n")
	}

	if totalLines > viewH {
		dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
		pos := ""
		if scroll == 0 {
			pos = "top"
		} else if end >= totalLines {
			pos = "end"
		} else {
			pct := (scroll * 100) / maxScroll
			pos = fmt.Sprintf("%d%%", pct)
		}
		b.WriteString(fmt.Sprintf("  %s\n", dimStyle.Render(
			fmt.Sprintf("[%d/%d %s]", scroll+viewH, totalLines, pos))))
	}

	return b.String()
}

// parsedStep holds a step with parsed timestamps for timeline rendering.
type parsedStep struct {
	idx       int
	startTime time.Time
	endTime   time.Time
	duration  time.Duration
}

// renderTimelineView renders a Gantt-chart timeline followed by event details.
func renderTimelineView(steps *ResourceWorkflowSteps, contentWidth int, items []wfEventItem, cursor int) ([]string, int) {
	// Try to parse timestamps for gantt chart
	var ps []parsedStep
	var globalStart, globalEnd time.Time
	allParsed := true

	for i, step := range steps.Steps {
		if step.StartTime == "" || step.EndTime == "" {
			allParsed = false
			break
		}
		st, err := time.Parse(time.RFC3339, step.StartTime)
		if err != nil {
			allParsed = false
			break
		}
		et, err := time.Parse(time.RFC3339, step.EndTime)
		if err != nil {
			allParsed = false
			break
		}
		if i == 0 || st.Before(globalStart) {
			globalStart = st
		}
		if i == 0 || et.After(globalEnd) {
			globalEnd = et
		}
		ps = append(ps, parsedStep{idx: i, startTime: st, endTime: et, duration: et.Sub(st)})
	}

	totalDuration := globalEnd.Sub(globalStart)
	if !allParsed || totalDuration <= 0 {
		return renderFlatStepRows(steps, contentWidth, items, cursor)
	}

	// Styles
	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("255"))
	dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	nameStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("117"))
	completedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("82"))
	runningStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("220"))
	failedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("203"))
	pendingStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	durStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
	timeStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	msgStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
	sepStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("238"))

	// Compute layout widths
	maxNameLen := 0
	for _, step := range steps.Steps {
		if len(step.stepDisplayName()) > maxNameLen {
			maxNameLen = len(step.stepDisplayName())
		}
	}
	if maxNameLen < 10 {
		maxNameLen = 10
	}

	// Layout: "  icon name  bar  duration"
	// 2 indent + 2 icon+space + maxNameLen name + 2 gap + barW + 2 gap + 7 duration
	barWidth := contentWidth - 2 - 2 - maxNameLen - 2 - 2 - 7
	if barWidth < 20 {
		barWidth = 20
	}

	var lines []string

	// Header
	failedCount := 0
	for _, s := range steps.Steps {
		if s.Status == "failed" {
			failedCount++
		}
	}
	headerText := fmt.Sprintf("Workflow Timeline · %d steps", len(steps.Steps))
	if failedCount > 0 {
		headerText += fmt.Sprintf(" · %d failed", failedCount)
	}
	headerText += fmt.Sprintf(" · %s", formatStepDuration(totalDuration))
	lines = append(lines, fmt.Sprintf("  %s", headerStyle.Render(headerText)))
	lines = append(lines, "")

	// Time axis
	leftPad := 2 + 2 + maxNameLen + 2
	startLabel := globalStart.Format("15:04:05")
	endLabel := globalEnd.Format("15:04:05")
	fillWidth := barWidth - len(startLabel) - len(endLabel)
	if fillWidth < 1 {
		fillWidth = 1
	}
	axisTimeLine := strings.Repeat(" ", leftPad) +
		timeStyle.Render(startLabel) +
		timeStyle.Render(strings.Repeat(" ", fillWidth)) +
		timeStyle.Render(endLabel)
	lines = append(lines, axisTimeLine)

	axisBarLine := strings.Repeat(" ", leftPad) +
		dimStyle.Render("┼"+strings.Repeat("─", barWidth-2)+"┼")
	lines = append(lines, axisBarLine)

	// Gantt bars
	for i, step := range steps.Steps {
		icon, sStyle := stepStatusIconAndStyle(step.Status, completedStyle, runningStyle, failedStyle, pendingStyle)
		bar := renderGanttBar(ps[i].startTime, ps[i].endTime, globalStart, totalDuration, barWidth, sStyle, dimStyle)
		durStr := formatStepDuration(ps[i].duration)
		namePadded := fmt.Sprintf("%-*s", maxNameLen, step.stepDisplayName())

		lines = append(lines, fmt.Sprintf("  %s %s  %s  %s",
			sStyle.Render(icon),
			nameStyle.Render(namePadded),
			bar,
			durStyle.Render(fmt.Sprintf("%7s", durStr)),
		))
	}

	// Separator
	lines = append(lines, "")
	sepWidth := contentWidth - 4
	if sepWidth < 20 {
		sepWidth = 20
	}
	lines = append(lines, fmt.Sprintf("  %s", sepStyle.Render("── Events "+strings.Repeat("─", sepWidth-10))))
	lines = append(lines, "")

	// Event details per step
	maxMsgWidth := contentWidth - 28
	if maxMsgWidth < 20 {
		maxMsgWidth = 20
	}

	// Build line-to-item mapping for cursor highlighting
	selectedLine := -1
	cursorItem := cursor
	itemIdx := 0

	hasAnyEvents := false
	for i, step := range steps.Steps {
		// Filter out generic step Started/Completed events (e.g. "workflow step Bootstrap started.")
		// Keep action-level events that happen to use Started/Completed types
		var actionEvents []dataaccess.DebugEvent
		for _, evt := range step.Events {
			et := model.WorkflowStepEventType(evt.EventType)
			if et == model.WorkflowStepStarted || et == model.WorkflowStepCompleted {
				// Keep if it has an action field (e.g. CreateLaunchTemplate)
				if !eventHasAction(evt.Message) {
					continue
				}
			}
			actionEvents = append(actionEvents, evt)
		}
		if len(actionEvents) == 0 && (!isBootstrapStep(step.Name) || len(step.DepTimelines) == 0) {
			continue
		}
		hasAnyEvents = true

		icon, sStyle := stepStatusIconAndStyle(step.Status, completedStyle, runningStyle, failedStyle, pendingStyle)
		timing := ""
		if step.StartTime != "" {
			timing = fmt.Sprintf("  %s → %s  (%s)",
				formatShortTime(step.StartTime),
				formatShortTime(step.EndTime),
				formatStepDuration(ps[i].duration))
		}

		lineIdx := len(lines)
		if itemIdx == cursorItem {
			selectedLine = lineIdx
		}
		itemIdx++

		lines = append(lines, fmt.Sprintf("  %s %s%s",
			sStyle.Render(icon),
			nameStyle.Render(step.stepDisplayName()),
			dimStyle.Render(timing),
		))

		// For bootstrap steps, show dependency timelines
		if isBootstrapStep(step.Name) && len(step.DepTimelines) > 0 {
			for _, dep := range step.DepTimelines {
				depIcon, depStyle := depStatusIconAndStyle(dep.Status, completedStyle, runningStyle, failedStyle, pendingStyle)
				depLine := fmt.Sprintf("    %s %s", depStyle.Render(depIcon), msgStyle.Render(dep.Name))
				if dep.FinishedAt != "" {
					depLine += fmt.Sprintf("  %s", dimStyle.Render("finished "+formatShortTime(dep.FinishedAt)))
				} else if dep.Status == "running" {
					depLine += fmt.Sprintf("  %s", runningStyle.Render("in progress…"))
				} else if dep.Status == "pending" {
					depLine += fmt.Sprintf("  %s", pendingStyle.Render("waiting"))
				}
				lines = append(lines, depLine)
			}
			lines = append(lines, "")
		}

		for _, evt := range actionEvents {
			evtIcon, evtStyle := actionIconAndStyle(evt.Message, completedStyle, runningStyle, failedStyle, dimStyle)
			ts := formatShortTime(evt.EventTime)
			msg := extractEventMessage(evt.Message)

			firstLine := strings.Split(msg, "\n")[0]
			wrappedMsg := softWrapLine(firstLine, maxMsgWidth)

			evtLineIdx := len(lines)
			if itemIdx == cursorItem {
				selectedLine = evtLineIdx
			}
			itemIdx++

			lines = append(lines, fmt.Sprintf("    %s %s %s",
				evtStyle.Render(evtIcon),
				dimStyle.Render(ts),
				msgStyle.Render(wrappedMsg[0]),
			))
			// Continuation lines aligned under message text
			if len(wrappedMsg) > 1 {
				contPad := "      " + strings.Repeat(" ", len(ts)) + " "
				for _, wl := range wrappedMsg[1:] {
					lines = append(lines, fmt.Sprintf("%s%s", contPad, msgStyle.Render(wl)))
				}
			}
		}

		if i < len(steps.Steps)-1 {
			lines = append(lines, "")
		}
	}

	if !hasAnyEvents {
		lines = append(lines, fmt.Sprintf("  %s", dimStyle.Render("No detailed events recorded.")))
	}

	// Apply cursor highlight to the selected line
	if selectedLine >= 0 && selectedLine < len(lines) {
		selectStyle := lipgloss.NewStyle().Background(lipgloss.Color("237"))
		lines[selectedLine] = selectStyle.Render(lines[selectedLine])
	}

	return lines, selectedLine
}

// renderGanttBar renders a single horizontal bar within the timeline.
func renderGanttBar(start, end time.Time, globalStart time.Time, totalDuration time.Duration, barWidth int, activeStyle, dimStyle lipgloss.Style) string {
	startFrac := float64(start.Sub(globalStart)) / float64(totalDuration)
	endFrac := float64(end.Sub(globalStart)) / float64(totalDuration)

	startCol := int(startFrac * float64(barWidth))
	endCol := int(endFrac * float64(barWidth))

	// Ensure at least 1 char wide
	if endCol <= startCol {
		endCol = startCol + 1
	}
	if startCol >= barWidth {
		startCol = barWidth - 1
	}
	if endCol > barWidth {
		endCol = barWidth
	}

	beforeLen := startCol
	activeLen := endCol - startCol
	afterLen := barWidth - endCol

	var bar strings.Builder
	if beforeLen > 0 {
		bar.WriteString(dimStyle.Render(strings.Repeat("─", beforeLen)))
	}
	if activeLen > 0 {
		bar.WriteString(activeStyle.Render(strings.Repeat("━", activeLen)))
	}
	if afterLen > 0 {
		bar.WriteString(dimStyle.Render(strings.Repeat("─", afterLen)))
	}
	return bar.String()
}

// formatStepDuration formats a duration for display.
func formatStepDuration(d time.Duration) string {
	if d < time.Second {
		return "<1s"
	}
	totalSec := int(d.Seconds())
	if totalSec < 60 {
		return fmt.Sprintf("%ds", totalSec)
	}
	m := totalSec / 60
	s := totalSec % 60
	if s == 0 {
		return fmt.Sprintf("%dm", m)
	}
	return fmt.Sprintf("%dm %ds", m, s)
}

// renderFlatStepRows is a fallback when timestamps can't be parsed for the gantt chart.
func renderFlatStepRows(steps *ResourceWorkflowSteps, contentWidth int, _ []wfEventItem, cursor int) ([]string, int) {
	nameStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("117"))
	dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	failedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("203"))
	completedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("82"))
	runningStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("220"))
	msgStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("252"))

	maxMsgWidth := contentWidth - 32
	if maxMsgWidth < 20 {
		maxMsgWidth = 20
	}

	var lines []string
	selectedLine := -1
	itemIdx := 0

	for i, step := range steps.Steps {
		icon, sStyle := stepStatusIconAndStyle(step.Status, completedStyle, runningStyle, failedStyle, dimStyle)

		timing := ""
		if step.StartTime != "" {
			start := formatShortTime(step.StartTime)
			end := formatShortTime(step.EndTime)
			if start == end {
				timing = start
			} else {
				timing = fmt.Sprintf("%s → %s", start, end)
			}
		}

		lineIdx := len(lines)
		if itemIdx == cursor {
			selectedLine = lineIdx
		}
		itemIdx++

		lines = append(lines, fmt.Sprintf("  %s %s  %s",
			sStyle.Render(icon),
			nameStyle.Render(step.stepDisplayName()),
			dimStyle.Render(timing),
		))

		// For bootstrap steps, show dependency timelines
		if isBootstrapStep(step.Name) && len(step.DepTimelines) > 0 {
			pendingStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
			for _, dep := range step.DepTimelines {
				depIcon, depStyle := depStatusIconAndStyle(dep.Status, completedStyle, runningStyle, failedStyle, pendingStyle)
				depLine := fmt.Sprintf("    %s %s", depStyle.Render(depIcon), msgStyle.Render(dep.Name))
				if dep.FinishedAt != "" {
					depLine += fmt.Sprintf("  %s", dimStyle.Render("finished "+formatShortTime(dep.FinishedAt)))
				} else if dep.Status == "running" {
					depLine += fmt.Sprintf("  %s", runningStyle.Render("in progress…"))
				} else if dep.Status == "pending" {
					depLine += fmt.Sprintf("  %s", pendingStyle.Render("waiting"))
				}
				lines = append(lines, depLine)
			}
			lines = append(lines, "")
		}

		for _, evt := range step.Events {
			evtIcon, evtStyle := eventIconAndStyle(evt.EventType, completedStyle, runningStyle, failedStyle, dimStyle)
			ts := formatShortTime(evt.EventTime)
			msg := extractEventMessage(evt.Message)

			firstLine := strings.Split(msg, "\n")[0]
			wrappedMsg := softWrapLine(firstLine, maxMsgWidth)

			evtLineIdx := len(lines)
			if itemIdx == cursor {
				selectedLine = evtLineIdx
			}
			itemIdx++

			lines = append(lines, fmt.Sprintf("    %s %s %s",
				evtStyle.Render(evtIcon),
				dimStyle.Render(ts),
				msgStyle.Render(wrappedMsg[0]),
			))
			// Continuation lines aligned under message text
			if len(wrappedMsg) > 1 {
				contPad := "      " + strings.Repeat(" ", len(ts)) + " "
				for _, wl := range wrappedMsg[1:] {
					lines = append(lines, fmt.Sprintf("%s%s", contPad, msgStyle.Render(wl)))
				}
			}
		}

		if i < len(steps.Steps)-1 {
			lines = append(lines, "")
		}
	}

	// Apply cursor highlight
	if selectedLine >= 0 && selectedLine < len(lines) {
		selectStyle := lipgloss.NewStyle().Background(lipgloss.Color("237"))
		lines[selectedLine] = selectStyle.Render(lines[selectedLine])
	}

	return lines, selectedLine
}

func stepStatusIconAndStyle(status string, completed, running, failed, dim lipgloss.Style) (string, lipgloss.Style) {
	switch status {
	case "success":
		return "✓", completed
	case "in-progress":
		return "●", running
	case "failed":
		return "✗", failed
	default:
		return "○", dim
	}
}

func depStatusIconAndStyle(status string, completed, running, failed, dim lipgloss.Style) (string, lipgloss.Style) {
	switch status {
	case "completed":
		return "✓", completed
	case "running":
		return "●", running
	case "failed":
		return "✗", failed
	default:
		return "○", dim
	}
}

// actionIconAndStyle picks icon/style based on the actionStatus field in the event message JSON.
func actionIconAndStyle(rawMessage string, completed, running, failed, dim lipgloss.Style) (string, lipgloss.Style) {
	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(rawMessage), &parsed); err != nil {
		return "·", dim
	}
	status, _ := parsed["actionStatus"].(string)
	switch strings.ToLower(status) {
	case "completed":
		return "✓", completed
	case "running", "started":
		return "●", running
	case "failed":
		return "✗", failed
	default:
		return "·", dim
	}
}

func eventIconAndStyle(eventType string, completed, running, failed, dim lipgloss.Style) (string, lipgloss.Style) {
	switch model.WorkflowStepEventType(eventType) {
	case model.WorkflowStepFailed:
		return "✗", failed
	case model.WorkflowStepCompleted:
		return "✓", completed
	case model.WorkflowStepStarted, model.WorkflowStepDebug:
		return "·", running
	case model.WorkflowStepPending:
		return "○", dim
	default:
		return "·", dim
	}
}

// extractEventMessage parses the JSON message field and returns a human-readable string.
func extractEventMessage(raw string) string {
	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(raw), &parsed); err != nil {
		return raw
	}
	if msg, ok := parsed["message"].(string); ok {
		if action, ok := parsed["action"].(string); ok {
			status := ""
			if s, ok := parsed["actionStatus"].(string); ok {
				status = " (" + s + ")"
			}
			return action + status + ": " + msg
		}
		return msg
	}
	return raw
}

// eventHasAction checks if a JSON event message contains an "action" field.
func eventHasAction(raw string) bool {
	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(raw), &parsed); err != nil {
		return false
	}
	_, ok := parsed["action"]
	return ok
}

func formatShortTime(ts string) string {
	if len(ts) >= 19 {
		return ts[11:19]
	}
	if len(ts) > 10 {
		return ts[11:]
	}
	return ts
}

// workflowEventsMaxScroll returns the max scroll position for workflow events.
// renderWfEventModal renders a full-screen modal showing event detail.
func renderWfEventModal(state *workflowErrorsState, width, height int) string {
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("230")).Background(lipgloss.Color("63")).Padding(0, 1)
	header := lipgloss.Place(width, 1, lipgloss.Left, lipgloss.Top, titleStyle.Render(fmt.Sprintf("Event Detail · %s", state.modalTitle)))

	bodyH := height - 4
	if bodyH < 1 {
		bodyH = 1
	}
	maxCodeWidth := width - 10
	if maxCodeWidth < 20 {
		maxCodeWidth = 20
	}

	lines := strings.Split(state.modalText, "\n")

	// Expand source lines into visual lines with wrapping
	vlines := expandLinesToVisual(lines, maxCodeWidth)
	totalLines := len(vlines)

	scroll := state.modalScroll
	maxScroll := totalLines - bodyH
	if maxScroll < 0 {
		maxScroll = 0
	}
	if scroll > maxScroll {
		scroll = maxScroll
	}

	end := scroll + bodyH
	if end > totalLines {
		end = totalLines
	}

	lineNumStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	textStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
	var b strings.Builder
	for i := scroll; i < end; i++ {
		vl := vlines[i]
		if vl.sourceNum > 0 {
			lineNum := lineNumStyle.Render(fmt.Sprintf("%4d", vl.sourceNum))
			b.WriteString(fmt.Sprintf("  %s │ %s\n", lineNum, textStyle.Render(vl.text)))
		} else {
			b.WriteString(fmt.Sprintf("  %s   %s\n", "    ", textStyle.Render(vl.text)))
		}
	}
	for i := end - scroll; i < bodyH; i++ {
		b.WriteString("\n")
	}

	footerStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Padding(0, 1)
	pos := ""
	if totalLines <= bodyH {
		pos = "all"
	} else if scroll == 0 {
		pos = "top"
	} else if end >= totalLines {
		pos = "end"
	} else if maxScroll > 0 {
		pos = fmt.Sprintf("%d%%", (scroll*100)/maxScroll)
	}
	footerText := fmt.Sprintf("↑↓/pgup/pgdn: scroll  esc: close  [%d/%d %s]", scroll+bodyH, totalLines, pos)
	footer := lipgloss.Place(width, 1, lipgloss.Left, lipgloss.Top, footerStyle.Render(footerText))

	return lipgloss.JoinVertical(lipgloss.Left, header, b.String(), footer)
}

// wfEventModalMaxScroll returns the max scroll for the event detail modal.
func wfEventModalMaxScroll(state *workflowErrorsState, width, height int) int {
	bodyH := height - 4
	if bodyH < 1 {
		bodyH = 1
	}
	maxCodeWidth := width - 10
	if maxCodeWidth < 20 {
		maxCodeWidth = 20
	}
	lines := strings.Split(state.modalText, "\n")
	vlines := expandLinesToVisual(lines, maxCodeWidth)
	maxScroll := len(vlines) - bodyH
	if maxScroll < 0 {
		maxScroll = 0
	}
	return maxScroll
}

// workflowEventsCopyText returns the plain text for clipboard copying.
func workflowEventsCopyText(steps *ResourceWorkflowSteps) string {
	if steps == nil || len(steps.Steps) == 0 {
		return ""
	}
	var b strings.Builder
	for _, step := range steps.Steps {
		b.WriteString(fmt.Sprintf("\n=== %s [%s] %s → %s ===\n", step.stepDisplayName(), step.Status, step.StartTime, step.EndTime))
		for _, evt := range step.Events {
			b.WriteString(fmt.Sprintf("[%s] %s: %s\n", evt.EventTime, evt.EventType, evt.Message))
		}
	}
	return b.String()
}

// formatEventDetail formats a DebugEvent's full message for display in the modal.
func formatEventDetail(evt *dataaccess.DebugEvent) string {
	if evt == nil {
		return ""
	}
	// Try to pretty-print as JSON
	var parsed interface{}
	if err := json.Unmarshal([]byte(evt.Message), &parsed); err == nil {
		pretty, err := json.MarshalIndent(parsed, "", "  ")
		if err == nil {
			return fmt.Sprintf("Time:  %s\nType:  %s\n\n%s", evt.EventTime, evt.EventType, string(pretty))
		}
	}
	return fmt.Sprintf("Time:  %s\nType:  %s\n\n%s", evt.EventTime, evt.EventType, evt.Message)
}

// extractEventAction returns the action name from a JSON event message, or a fallback.
func extractEventAction(raw string) string {
	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(raw), &parsed); err == nil {
		if action, ok := parsed["action"].(string); ok {
			return action
		}
	}
	return "Event"
}
