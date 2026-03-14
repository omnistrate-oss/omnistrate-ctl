package instance

import (
	"context"
	"fmt"
	"math"
	"os"
	"strings"
	"time"

	"github.com/omnistrate-oss/omnistrate-ctl/internal/dataaccess"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/model"
	"golang.org/x/term"
)

// ANSI escape codes for in-place terminal updates (no TUI library needed)
const (
	ansiClearLine  = "\033[2K"
	ansiHideCursor = "\033[?25l"
	ansiShowCursor = "\033[?25h"
	ansiBold       = "\033[1m"
	ansiReset      = "\033[0m"
	ansiGreen      = "\033[32m"
	ansiRed        = "\033[31m"
	ansiYellow     = "\033[33m"
	ansiCyan       = "\033[36m"
	ansiDim        = "\033[2m"
)

// progressResource holds the display state for one resource during progress rendering
type progressResource struct {
	name      string
	key       string
	id        string
	percent   int
	status    string // "pending", "running", "completed", "failed"
	steps     []progressStep
	lastEvent string
}

// progressStep holds the display state for one workflow step within a resource
type progressStep struct {
	name      string
	status    string // "pending", "running", "completed", "failed"
	eventType string
	lastEvent string // most recent event message for this step
}

// DisplayWorkflowProgressTable shows a non-TUI, ANSI-based live progress table
// for all resources in a deployment workflow. It polls every few seconds and
// redraws the table in-place using cursor movement escape codes.
// Returns the same WorkflowMonitorResult as the spinner-based version.
func DisplayWorkflowProgressTable(ctx context.Context, token, instanceID, actionType string) (WorkflowMonitorResult, error) {
	result := WorkflowMonitorResult{
		InstanceID: instanceID,
		ActionType: actionType,
	}

	// Search for the instance to get service details
	searchRes, err := dataaccess.SearchInventory(ctx, token, fmt.Sprintf("resourceinstance:%s", instanceID))
	if err != nil {
		return result, err
	}
	if len(searchRes.ResourceInstanceResults) == 0 {
		return result, fmt.Errorf("instance not found")
	}

	inst := searchRes.ResourceInstanceResults[0]
	result.ServiceID = inst.ServiceId
	result.EnvironmentID = inst.ServiceEnvironmentId

	// Hide cursor during rendering, restore on exit
	fmt.Print(ansiHideCursor)
	defer fmt.Print(ansiShowCursor)

	prevLineCount := 0
	fetchInterval := 5 * time.Second
	renderInterval := 200 * time.Millisecond

	renderTicker := time.NewTicker(renderInterval)
	defer renderTicker.Stop()

	spinnerFrames := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
	spinnerIdx := 0
	elapsedStart := time.Now()
	lastFetch := time.Time{} // zero → triggers immediate first fetch

	// Cached data between fetches
	var resources []progressResource
	var workflowInfo *dataaccess.WorkflowInfo
	var isComplete bool

	for {
		// ── Fetch data when interval has elapsed ──
		if time.Since(lastFetch) >= fetchInterval {
			resourcesData, wfInfo, fetchErr := dataaccess.GetDebugEventsForAllResources(
				ctx, token,
				inst.ServiceId,
				inst.ServiceEnvironmentId,
				instanceID,
				true,
				actionType,
			)
			if fetchErr != nil {
				return result, fetchErr
			}
			lastFetch = time.Now()

			if wfInfo == nil {
				<-renderTicker.C
				continue
			}

			workflowInfo = wfInfo
			result.WorkflowID = workflowInfo.WorkflowID
			result.WorkflowStatus = workflowInfo.WorkflowStatus

			isComplete = strings.ToLower(workflowInfo.WorkflowStatus) == "success" ||
				strings.ToLower(workflowInfo.WorkflowStatus) == "failed" ||
				strings.ToLower(workflowInfo.WorkflowStatus) == "cancelled"

			resources = buildProgressResources(resourcesData, workflowInfo)

			// Track failures
			for _, res := range resources {
				if res.status == "failed" {
					for _, rd := range resourcesData {
						if rd.ResourceID == res.id || rd.ResourceKey == res.key {
							result.FailedResourceID = rd.ResourceID
							result.FailedResourceKey = rd.ResourceKey
							result.FailedResourceName = rd.ResourceName
							result.FailedStep, result.FailedReason = getFailedStepAndMessage(rd.EventsByWorkflowStep)
							break
						}
					}
				}
			}
		}

		// Skip rendering until first data is available
		if workflowInfo == nil {
			<-renderTicker.C
			continue
		}

		// ── Render frame ──
		elapsed := time.Since(elapsedStart).Truncate(time.Second)
		spinnerChar := spinnerFrames[spinnerIdx%len(spinnerFrames)]
		spinnerIdx++

		termWidth := getTermWidth()
		lines := renderProgressLines(resources, workflowInfo, elapsed, spinnerChar, isComplete, termWidth)

		// Move cursor up to overwrite previous output
		if prevLineCount > 0 {
			fmt.Printf("\033[%dA", prevLineCount)
		}
		for _, line := range lines {
			fmt.Printf("\r%s%s\n", ansiClearLine, line)
		}
		if len(lines) < prevLineCount {
			for i := len(lines); i < prevLineCount; i++ {
				fmt.Printf("\r%s\n", ansiClearLine)
			}
			fmt.Printf("\033[%dA", prevLineCount-len(lines))
		}
		prevLineCount = len(lines)

		// ── Exit on completion or failure ──
		if isComplete {
			if strings.ToLower(workflowInfo.WorkflowStatus) == "failed" || strings.ToLower(workflowInfo.WorkflowStatus) == "cancelled" {
				if result.FailedStep != "" && result.FailedReason != "" {
					return result, fmt.Errorf("resource %s failed at %s: %s", result.FailedResourceName, result.FailedStep, result.FailedReason)
				}
				return result, fmt.Errorf("with status: %s", workflowInfo.WorkflowStatus)
			}
			return result, nil
		}

		for _, res := range resources {
			if res.status == "failed" {
				if result.FailedStep != "" && result.FailedReason != "" {
					return result, fmt.Errorf("resource %s failed at %s: %s", result.FailedResourceName, result.FailedStep, result.FailedReason)
				}
				return result, fmt.Errorf("resource %s failed", res.name)
			}
		}

		<-renderTicker.C
	}
}

// buildProgressResources converts API data into display-ready progress resources
func buildProgressResources(resourcesData []dataaccess.ResourceWorkflowDebugEvents, workflowInfo *dataaccess.WorkflowInfo) []progressResource {
	var resources []progressResource

	for _, rd := range resourcesData {
		res := progressResource{
			name: rd.ResourceName,
			key:  rd.ResourceKey,
			id:   rd.ResourceID,
		}

		if rd.EventsByWorkflowStep != nil {
			orderedSteps := []struct {
				name   string
				events []dataaccess.DebugEvent
			}{
				{"BOOTSTRAP", rd.EventsByWorkflowStep.Bootstrap},
				{"STORAGE", rd.EventsByWorkflowStep.Storage},
				{"NETWORK", rd.EventsByWorkflowStep.Network},
				{"COMPUTE", rd.EventsByWorkflowStep.Compute},
				{"DEPLOYMENT", rd.EventsByWorkflowStep.Deployment},
				{"MONITORING", rd.EventsByWorkflowStep.Monitoring},
			}

			total := 0
			completed := 0
			hasFailed := false
			hasRunning := false

			for _, step := range orderedSteps {
				if len(step.events) == 0 {
					continue
				}
				total++
				eventType := getHighestPriorityEventType(step.events)
				stepStatus := "pending"
				switch model.WorkflowStepEventType(eventType) {
				case model.WorkflowStepCompleted:
					stepStatus = "completed"
					completed++
				case model.WorkflowStepFailed:
					stepStatus = "failed"
					hasFailed = true
				case model.WorkflowStepDebug, model.WorkflowStepStarted:
					stepStatus = "running"
					hasRunning = true
				}
				// Capture latest event message for this step
				var stepLastEvent string
				if len(step.events) > 0 {
					lastEvt := step.events[len(step.events)-1]
					if lastEvt.Message != "" {
						stepLastEvent = lastEvt.Message
						res.lastEvent = lastEvt.Message
					}
				}

				res.steps = append(res.steps, progressStep{
					name:      step.name,
					status:    stepStatus,
					eventType: eventType,
					lastEvent: stepLastEvent,
				})
			}

			if total > 0 {
				res.percent = int(math.Round((float64(completed) / float64(total)) * 100))
			}

			if hasFailed {
				res.status = "failed"
			} else if completed == total && total > 0 {
				res.status = "completed"
				res.percent = 100
			} else if hasRunning || completed > 0 {
				res.status = "running"
			} else {
				res.status = "pending"
				res.percent = 0
			}
		} else {
			res.status = "pending"
			res.percent = 0
		}

		// Override from workflow-level status if available
		if rd.WorkflowStatus != nil {
			ws := model.ParseWorkflowStatus(*rd.WorkflowStatus)
			if ws == model.WorkflowStatusCompleted && res.status != "failed" {
				res.status = "completed"
				res.percent = 100
			} else if ws == model.WorkflowStatusFailed {
				res.status = "failed"
			}
		}

		resources = append(resources, res)
	}

	return resources
}

// renderProgressLines builds the full display as a slice of plain-string lines.
// Each line is padded to exactly termWidth visible characters so that it occupies
// a single terminal row and never wraps — this is essential for the cursor-up
// redraw to work correctly.
func renderProgressLines(resources []progressResource, workflowInfo *dataaccess.WorkflowInfo, elapsed time.Duration, spinnerChar string, isComplete bool, termWidth int) []string {
	if termWidth <= 0 {
		termWidth = 100
	}
	maxW := min(termWidth, 140)

	var lines []string

	// ── Header ──────────────────────────────────────────────
	statusLabel := spinnerChar + " Running"
	if isComplete {
		if strings.ToLower(workflowInfo.WorkflowStatus) == "success" {
			statusLabel = "✅ Completed"
		} else {
			statusLabel = "❌ " + strings.ToUpper(workflowInfo.WorkflowStatus)
		}
	}
	wfIDDisplay := truncateStr(workflowInfo.WorkflowID, 45)
	header := fmt.Sprintf("Workflow: %s  Status: %s  Elapsed: %s", wfIDDisplay, statusLabel, elapsed.String())
	lines = append(lines, padToWidth(header, maxW))
	lines = append(lines, strings.Repeat("─", maxW))

	// ── Resource rows ───────────────────────────────────────
	for i, res := range resources {
		// Blank separator between resources
		if i > 0 {
			lines = append(lines, padToWidth("", maxW))
		}

		icon := statusIcon(res.status)
		nameLabel := res.name
		if nameLabel == "" {
			nameLabel = res.key
		}

		// Line 1: icon  Name          ████░░░░  60%   (key: xxx)
		bar := progressBar(res.percent, res.status, 20)
		pctStr := fmt.Sprintf("%3d%%", res.percent)

		resourceLine := fmt.Sprintf("  %s %s%-18s%s  %s %s", icon, ansiBold, truncateStr(nameLabel, 18), ansiReset, bar, pctStr)
		if res.key != "" && res.key != res.name {
			resourceLine += fmt.Sprintf("  %s(key: %s)%s", ansiDim, truncateStr(res.key, 20), ansiReset)
		}
		lines = append(lines, padToWidth(resourceLine, maxW))

		// Line 2: step icons on one row  BOOTSTRAP:✓ │ STORAGE:✓ │ NETWORK:⟳
		if len(res.steps) > 0 {
			var parts []string
			for _, step := range res.steps {
				parts = append(parts, fmt.Sprintf("%s:%s", step.name, stepIcon(step.status, spinnerChar)))
			}
			stepsLine := "     " + strings.Join(parts, " │ ")
			lines = append(lines, padToWidth(stepsLine, maxW))

			// Line 3: latest event from the most recent active/completed step
			if res.lastEvent != "" {
				maxEvtLen := maxW - 8
				if maxEvtLen < 20 {
					maxEvtLen = 20
				}
				evtLine := fmt.Sprintf("     %s↳ %s%s", ansiDim, truncateStr(cleanEventMessage(res.lastEvent), maxEvtLen), ansiReset)
				lines = append(lines, padToWidth(evtLine, maxW))
			}
		} else if res.status == "pending" {
			waitLine := fmt.Sprintf("     %s%s Waiting for dependencies…%s", ansiDim, spinnerChar, ansiReset)
			lines = append(lines, padToWidth(waitLine, maxW))
		}
	}

	lines = append(lines, strings.Repeat("─", maxW))
	return lines
}

// ── Rendering helpers ─────────────────────────────────────────────

// progressBar renders a plain-text colored bar.
func progressBar(percent int, status string, width int) string {
	if width < 4 {
		width = 10
	}
	filled := int(math.Round(float64(percent) / 100.0 * float64(width)))
	if filled > width {
		filled = width
	}
	empty := width - filled

	color := ansiDim
	switch status {
	case "completed":
		color = ansiGreen
	case "failed":
		color = ansiRed
	case "running":
		color = ansiYellow
	}

	return color + strings.Repeat("█", filled) + ansiReset +
		ansiDim + strings.Repeat("░", empty) + ansiReset
}

func statusIcon(status string) string {
	switch status {
	case "completed":
		return ansiGreen + "✓" + ansiReset
	case "failed":
		return ansiRed + "✗" + ansiReset
	case "running":
		return ansiYellow + "●" + ansiReset
	default:
		return ansiDim + "○" + ansiReset
	}
}

func stepIcon(status string, spinnerChar string) string {
	switch status {
	case "completed":
		return ansiGreen + "✓" + ansiReset
	case "failed":
		return ansiRed + "✗" + ansiReset
	case "running":
		return ansiYellow + spinnerChar + ansiReset
	default:
		return ansiDim + "○" + ansiReset
	}
}

// cleanEventMessage strips JSON wrapper if the message looks like
// {"message":"workflow step Monitoring completed."}
func cleanEventMessage(msg string) string {
	msg = strings.TrimSpace(msg)
	if strings.HasPrefix(msg, "{\"message\":\"") && strings.HasSuffix(msg, "\"}") {
		inner := msg[len("{\"message\":\"") : len(msg)-len("\"}")]
		if inner != "" {
			return inner
		}
	}
	if strings.HasPrefix(msg, "{") && strings.HasSuffix(msg, "}") {
		idx := strings.Index(msg, "\"message\"")
		if idx >= 0 {
			rest := msg[idx+len("\"message\""):]
			rest = strings.TrimLeft(rest, ": ")
			rest = strings.TrimPrefix(rest, "\"")
			endQuote := strings.Index(rest, "\"")
			if endQuote > 0 {
				return rest[:endQuote]
			}
		}
	}
	return msg
}

// truncateStr truncates s to maxLen runes, appending "…" if truncated.
func truncateStr(s string, maxLen int) string {
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	if maxLen <= 1 {
		return "…"
	}
	return string(runes[:maxLen-1]) + "…"
}

// padToWidth pads the string with spaces so its visible length (ignoring ANSI
// escape sequences) is exactly width. This ensures each line occupies one
// terminal row and never wraps, which is critical for cursor-up redraws.
func padToWidth(s string, width int) string {
	vLen := visibleLen(s)
	if vLen >= width {
		return s
	}
	return s + strings.Repeat(" ", width-vLen)
}

// visibleLen counts visible characters, skipping ANSI escape sequences.
func visibleLen(s string) int {
	n := 0
	inEsc := false
	for _, r := range s {
		if inEsc {
			if (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') || r == '~' {
				inEsc = false
			}
			continue
		}
		if r == '\033' {
			inEsc = true
			continue
		}
		n++
	}
	return n
}

// getTermWidth returns the terminal width, defaulting to 120 if unavailable.
func getTermWidth() int {
	w, _, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil || w <= 0 {
		return 120
	}
	return w
}
