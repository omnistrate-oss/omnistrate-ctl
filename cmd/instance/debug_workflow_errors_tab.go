package instance

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/dataaccess"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/model"
)

// workflowErrorsState holds the scroll state for the workflow events tab.
type workflowErrorsState struct {
	scroll int
}

// WorkflowStepInfo holds step-level summary with timing and events.
type WorkflowStepInfo struct {
	Name      string
	Status    string // "success", "in-progress", "failed", "pending"
	StartTime string
	EndTime   string
	Events    []dataaccess.DebugEvent
}

// ResourceWorkflowSteps holds the ordered list of workflow steps for a resource.
type ResourceWorkflowSteps struct {
	Steps []WorkflowStepInfo
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
func renderWorkflowEventsTab(steps *ResourceWorkflowSteps, scroll, bodyHeight, contentWidth int, loading bool, spinnerView string) string {
	if loading && (steps == nil || len(steps.Steps) == 0) {
		return fmt.Sprintf("\n  %s Fetching workflow events...", spinnerView)
	}
	if steps == nil || len(steps.Steps) == 0 {
		subtleStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
		return fmt.Sprintf("\n  %s\n", subtleStyle.Render("No workflow events available for this resource."))
	}

	rendered := renderTimelineView(steps, contentWidth)
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
	if scroll > maxScroll {
		scroll = maxScroll
	}
	if scroll < 0 {
		scroll = 0
	}

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
func renderTimelineView(steps *ResourceWorkflowSteps, contentWidth int) []string {
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
		return renderFlatStepRows(steps, contentWidth)
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
		if len(step.Name) > maxNameLen {
			maxNameLen = len(step.Name)
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
		namePadded := fmt.Sprintf("%-*s", maxNameLen, step.Name)

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
		if len(actionEvents) == 0 {
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

		lines = append(lines, fmt.Sprintf("  %s %s%s",
			sStyle.Render(icon),
			nameStyle.Render(step.Name),
			dimStyle.Render(timing),
		))

		for _, evt := range actionEvents {
			evtIcon, evtStyle := actionIconAndStyle(evt.Message, completedStyle, runningStyle, failedStyle, dimStyle)
			ts := formatShortTime(evt.EventTime)
			msg := extractEventMessage(evt.Message)

			firstLine := strings.Split(msg, "\n")[0]
			runes := []rune(firstLine)
			if len(runes) > maxMsgWidth {
				firstLine = string(runes[:maxMsgWidth-1]) + "…"
			}

			lines = append(lines, fmt.Sprintf("    %s %s %s",
				evtStyle.Render(evtIcon),
				dimStyle.Render(ts),
				msgStyle.Render(firstLine),
			))
		}

		if i < len(steps.Steps)-1 {
			lines = append(lines, "")
		}
	}

	if !hasAnyEvents {
		lines = append(lines, fmt.Sprintf("  %s", dimStyle.Render("No detailed events recorded.")))
	}

	return lines
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
func renderFlatStepRows(steps *ResourceWorkflowSteps, contentWidth int) []string {
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

		lines = append(lines, fmt.Sprintf("  %s %s  %s",
			sStyle.Render(icon),
			nameStyle.Render(step.Name),
			dimStyle.Render(timing),
		))

		for _, evt := range step.Events {
			evtIcon, evtStyle := eventIconAndStyle(evt.EventType, completedStyle, runningStyle, failedStyle, dimStyle)
			ts := formatShortTime(evt.EventTime)
			msg := extractEventMessage(evt.Message)

			firstLine := strings.Split(msg, "\n")[0]
			runes := []rune(firstLine)
			if len(runes) > maxMsgWidth {
				firstLine = string(runes[:maxMsgWidth-1]) + "…"
			}

			lines = append(lines, fmt.Sprintf("    %s %s %s",
				evtStyle.Render(evtIcon),
				dimStyle.Render(ts),
				msgStyle.Render(firstLine),
			))
		}

		if i < len(steps.Steps)-1 {
			lines = append(lines, "")
		}
	}
	return lines
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
func workflowEventsMaxScroll(steps *ResourceWorkflowSteps, contentWidth, bodyHeight int) int {
	if steps == nil {
		return 0
	}
	rendered := renderTimelineView(steps, contentWidth)
	viewH := bodyHeight - 2
	if viewH < 1 {
		viewH = 1
	}
	maxScroll := len(rendered) - viewH
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
		b.WriteString(fmt.Sprintf("\n=== %s [%s] %s → %s ===\n", step.Name, step.Status, step.StartTime, step.EndTime))
		for _, evt := range step.Events {
			b.WriteString(fmt.Sprintf("[%s] %s: %s\n", evt.EventTime, evt.EventType, evt.Message))
		}
	}
	return b.String()
}
