package instance

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// logLineMsg delivers a batch of new log lines
type logLineMsg struct {
	lines   []string
	label   string
	replace bool // if true, replace all existing lines instead of appending
}

// logStreamDoneMsg signals the log polling has ended
type logStreamDoneMsg struct {
	err error
}

const logPollInterval = 3 * time.Second

// findLatestOperationID finds the latest operation ID that has an apply or destroy log.
func findLatestApplyDestroyOperationID(cmData map[string]string, history []TerraformHistoryEntry) string {
	if len(cmData) == 0 || len(history) == 0 {
		return ""
	}
	for i := len(history) - 1; i >= 0; i-- {
		entry := history[i]
		op := strings.ToLower(entry.Operation)
		if op != "apply" && op != "destroy" {
			continue
		}
		key := entry.OperationID + "-" + op + ".log"
		if _, ok := cmData[key]; ok {
			return entry.OperationID
		}
	}
	return ""
}

// collectLogsForOperationID gathers all log files belonging to an operation ID,
// ordered chronologically by their position in history with separators between each.
func collectLogsForOperationID(cmData map[string]string, history []TerraformHistoryEntry, opID string) ([]string, string) {
	if opID == "" {
		return nil, ""
	}

	type opLog struct {
		operation string
		content   string
		index     int
	}

	var entries []opLog
	for i, entry := range history {
		if entry.OperationID != opID {
			continue
		}
		op := strings.ToLower(entry.Operation)
		key := entry.OperationID + "-" + op + ".log"
		if content, ok := cmData[key]; ok && strings.TrimSpace(content) != "" {
			entries = append(entries, opLog{operation: op, content: content, index: i})
		}
	}

	// Deduplicate by operation name — keep the later (more complete) one
	seen := make(map[string]int)
	var deduped []opLog
	for _, e := range entries {
		if idx, exists := seen[e.operation]; exists {
			deduped[idx] = e
		} else {
			seen[e.operation] = len(deduped)
			deduped = append(deduped, e)
		}
	}
	entries = deduped

	if len(entries) == 0 {
		return nil, ""
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].index < entries[j].index
	})

	// Build label from operation sequence
	shortID := opID
	if len(shortID) > 8 {
		shortID = shortID[:8]
	}
	var opNames []string
	for _, e := range entries {
		opNames = append(opNames, e.operation)
	}
	label := fmt.Sprintf("%s (%s)", shortID, strings.Join(opNames, " → "))

	// Stitch logs with separators
	var lines []string
	for i, e := range entries {
		if i > 0 {
			lines = append(lines, "")
		}
		lines = append(lines, fmt.Sprintf("─── %s ───", e.operation))
		lines = append(lines, "")
		contentLines := strings.Split(e.content, "\n")
		// Trim trailing empty lines to avoid phantom growth on each poll
		for len(contentLines) > 0 && strings.TrimSpace(contentLines[len(contentLines)-1]) == "" {
			contentLines = contentLines[:len(contentLines)-1]
		}
		lines = append(lines, contentLines...)
	}

	return lines, label
}

// fetchLogsFromConfigMap fetches all logs for the latest operation ID from the tf-state configmap.
func fetchLogsFromConfigMap(ctx context.Context, conn *k8sConnection, instanceID, resourceID string, history []TerraformHistoryEntry) ([]string, string, string, []TerraformHistoryEntry, error) {
	index, err := loadTerraformConfigMapIndex(ctx, conn.clientset, instanceID)
	if err != nil {
		return nil, "", "", history, err
	}
	if index == nil {
		return nil, "", "", history, nil
	}

	normalizedID := normalizeResourceIDForConfigMap(resourceID)
	var cmData map[string]string
	for _, key := range []string{
		normalizedID,
		resourceID,
		"tf-" + normalizedID,
		"tf-" + strings.ToLower(resourceID),
	} {
		if cm, ok := index.stateByResource[key]; ok && cm != nil {
			cmData = cm.Data
			break
		}
	}
	if cmData == nil {
		return nil, "", "", history, nil
	}

	// Re-read history from configmap for freshest data
	if historyJSON, ok := cmData["history"]; ok {
		var newHistory []TerraformHistoryEntry
		if jsonErr := json.Unmarshal([]byte(historyJSON), &newHistory); jsonErr == nil {
			history = newHistory
		}
	}

	opID := findLatestApplyDestroyOperationID(cmData, history)
	lines, label := collectLogsForOperationID(cmData, history, opID)
	return lines, label, opID, history, nil
}

// watchApplyDestroyLogs polls the configmap for log updates for the latest operation ID
// and sends new lines via the channel.
func watchApplyDestroyLogs(conn *k8sConnection, instanceID, resourceID string, history []TerraformHistoryEntry, ch chan logLineMsg) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		var prevLines []string
		prevOpID := ""

		for {
			lines, label, opID, newHistory, err := fetchLogsFromConfigMap(ctx, conn, instanceID, resourceID, history)
			if err != nil {
				close(ch)
				return logStreamDoneMsg{err: err}
			}
			history = newHistory

			if len(lines) > 0 {
				if opID != prevOpID && prevOpID != "" {
					// New operation ID — replace with separator and full content
					ch <- logLineMsg{
						lines:   append([]string{"", "═══ new operation ═══", ""}, lines...),
						label:   label,
						replace: true,
					}
					prevLines = lines
				} else if len(prevLines) == 0 {
					// First fetch — send full content
					ch <- logLineMsg{lines: lines, label: label}
					prevLines = lines
				} else if len(lines) > len(prevLines) {
					// Same operation, more content — send only new lines
					ch <- logLineMsg{
						lines: lines[len(prevLines):],
						label: label,
					}
					prevLines = lines
				} else if !slicesEqual(lines, prevLines) {
					// Content changed (e.g. new sub-operation appeared) — replace all
					ch <- logLineMsg{lines: lines, label: label, replace: true}
					prevLines = lines
				}
				prevOpID = opID
			}

			time.Sleep(logPollInterval)
		}
	}
}

// slicesEqual compares two string slices for equality.
func slicesEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// waitForLogLines blocks until the next batch of lines arrives on the channel.
func waitForLogLines(ch chan logLineMsg) tea.Cmd {
	return func() tea.Msg {
		msg, ok := <-ch
		if !ok {
			return logStreamDoneMsg{}
		}
		return msg
	}
}

// renderLogsTab renders the live log viewer
func (m terraformDetailModel) renderLogsTab() string {
	if m.loading {
		return fmt.Sprintf("\n  %s Fetching logs...", m.spinner.View())
	}
	if m.logErr != nil {
		errStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("203"))
		return fmt.Sprintf("\n  %s\n", errStyle.Render(fmt.Sprintf("Error: %v", m.logErr)))
	}
	if !m.logStreaming && !m.logDone && len(m.logLines) == 0 {
		subtleStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
		return fmt.Sprintf("\n  %s\n", subtleStyle.Render("No operation logs available for this resource."))
	}

	var b strings.Builder

	// Header
	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("255"))
	statusText := ""
	if m.logStreaming {
		statusText = fmt.Sprintf("  %s", m.spinner.View()) + lipgloss.NewStyle().Foreground(lipgloss.Color("82")).Render(" live")
	} else if m.logDone {
		statusText = lipgloss.NewStyle().Foreground(lipgloss.Color("245")).Render("  ○ ended")
	}
	followText := ""
	if m.logFollow {
		followText = lipgloss.NewStyle().Foreground(lipgloss.Color("82")).Render("  [follow]")
	}
	labelText := ""
	if m.logLabel != "" {
		labelText = lipgloss.NewStyle().Foreground(lipgloss.Color("245")).Render("  " + m.logLabel)
	}
	b.WriteString(fmt.Sprintf("  %s%s%s%s\n\n",
		headerStyle.Render(fmt.Sprintf("Operation Logs (%d lines)", len(m.logLines))),
		statusText,
		followText,
		labelText,
	))

	bodyH := m.bodyHeight() - 4
	if bodyH < 1 {
		bodyH = 1
	}

	totalLines := len(m.logLines)

	// Scroll position is managed in Update()
	scroll := m.logScroll
	// Clamp scroll
	maxScroll := totalLines - bodyH
	if maxScroll < 0 {
		maxScroll = 0
	}
	if scroll > maxScroll {
		scroll = maxScroll
	}
	if scroll < 0 {
		scroll = 0
	}

	end := scroll + bodyH
	if end > totalLines {
		end = totalLines
	}

	lineNumStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	maxCodeWidth := m.contentWidth() - 9
	if maxCodeWidth < 20 {
		maxCodeWidth = 20
	}

	for i := scroll; i < end; i++ {
		line := m.logLines[i]
		runes := []rune(line)
		if len(runes) > maxCodeWidth {
			line = string(runes[:maxCodeWidth-1]) + "…"
		}
		lineNum := lineNumStyle.Render(fmt.Sprintf("%4d", i+1))
		styled := highlightLogLine(line)
		b.WriteString(fmt.Sprintf("  %s │ %s\n", lineNum, styled))
	}

	// Pad remaining lines
	for i := end - scroll; i < bodyH; i++ {
		b.WriteString("\n")
	}

	// Scroll indicator
	if totalLines > bodyH {
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
			fmt.Sprintf("[%d/%d %s]", scroll+bodyH, totalLines, pos))))
	}

	return b.String()
}

// highlightLogLine applies basic coloring to terraform log output
func highlightLogLine(line string) string {
	errStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("203"))
	warnStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("220"))
	successStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("82"))
	changeStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("117"))
	dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("245"))

	lower := strings.ToLower(line)

	switch {
	case strings.Contains(lower, "error") || strings.Contains(lower, "fatal"):
		return errStyle.Render(line)
	case strings.Contains(lower, "warning") || strings.Contains(lower, "warn"):
		return warnStyle.Render(line)
	case strings.HasPrefix(line, "Apply complete") || strings.HasPrefix(line, "Destroy complete"):
		return successStyle.Render(line)
	case strings.HasPrefix(line, "  # ") || strings.HasPrefix(line, "  + ") || strings.HasPrefix(line, "  - ") || strings.HasPrefix(line, "  ~ "):
		return changeStyle.Render(line)
	case strings.HasPrefix(line, "Terraform") || strings.HasPrefix(line, "OpenTofu"):
		return lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("252")).Render(line)
	case strings.HasPrefix(line, "───") || strings.HasPrefix(line, "═══") || (strings.HasPrefix(line, "---") && strings.HasSuffix(line, "---")):
		return lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("220")).Render(line)
	case strings.TrimSpace(line) == "":
		return line
	case strings.HasPrefix(line, "  "):
		return dimStyle.Render(line)
	default:
		return line
	}
}
