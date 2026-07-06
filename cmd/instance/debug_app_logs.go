package instance

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/dataaccess"
)

type appLogLineMsg struct {
	lines []string
}

type appLogStreamDoneMsg struct {
	err error
}

type appLogsState struct {
	streams []dataaccess.LogsStream
	lines   []string
	ch      chan appLogLineMsg
	cancel  context.CancelFunc

	scroll    int
	follow    bool
	streaming bool
	done      bool
	err       error
	started   bool
}

func newAppLogsState(streams []dataaccess.LogsStream) *appLogsState {
	return &appLogsState{
		streams: append([]dataaccess.LogsStream(nil), streams...),
		ch:      make(chan appLogLineMsg, 100),
		follow:  true,
	}
}

func appLogStreamsForResource(data DebugData, node PlanDAGNode) []dataaccess.LogsStream {
	if len(data.AppLogStreams) == 0 {
		return nil
	}

	keys := []string{node.Key, node.ID, node.Name}
	for _, key := range keys {
		if key == "" {
			continue
		}
		if streams := appLogStreamsForKey(data.AppLogStreams, key); len(streams) > 0 {
			copied := append([]dataaccess.LogsStream(nil), streams...)
			sort.Slice(copied, func(i, j int) bool {
				return copied[i].PodName < copied[j].PodName
			})
			return copied
		}
	}
	return nil
}

func appLogStreamsForKey(streamsByKey map[string][]dataaccess.LogsStream, key string) []dataaccess.LogsStream {
	if streams := streamsByKey[key]; len(streams) > 0 {
		return streams
	}
	normalizedKey := strings.ToLower(strings.TrimSpace(key))
	if normalizedKey == "" {
		return nil
	}
	for candidate, streams := range streamsByKey {
		if strings.ToLower(strings.TrimSpace(candidate)) == normalizedKey && len(streams) > 0 {
			return streams
		}
	}
	return nil
}

func appLogsUnavailableMessage(data DebugData, node PlanDAGNode) string {
	if strings.TrimSpace(data.AppLogsError) != "" {
		return data.AppLogsError
	}

	selected := appLogSelectedResourceNames(node)
	available := appLogAvailableStreamKeys(data.AppLogStreams)
	if len(available) == 0 {
		if len(selected) == 0 {
			return "No app log streams available for this resource. No Kubernetes pod log streams were discovered for this instance."
		}
		return fmt.Sprintf("No app log streams available for this resource. Selected resource: %s. No Kubernetes pod log streams were discovered for this instance.", strings.Join(selected, ", "))
	}

	msg := "No app log streams available for this resource."
	if len(selected) > 0 {
		msg += " Selected resource: " + strings.Join(selected, ", ") + "."
	}
	msg += " Logs are available for other resource keys: " + strings.Join(available, ", ") + ". Select one of those resources to view its app logs."
	return msg
}

func appLogSelectedResourceNames(node PlanDAGNode) []string {
	seen := make(map[string]struct{})
	var result []string
	for _, value := range []string{node.Key, node.ID, node.Name} {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
}

func appLogAvailableStreamKeys(streamsByKey map[string][]dataaccess.LogsStream) []string {
	if len(streamsByKey) == 0 {
		return nil
	}

	keys := make([]string, 0, len(streamsByKey))
	for key, streams := range streamsByKey {
		if strings.TrimSpace(key) == "" || len(streams) == 0 {
			continue
		}
		keys = append(keys, key)
	}
	sort.Strings(keys)
	if len(keys) > 12 {
		keys = append(keys[:12], fmt.Sprintf("... %d more", len(keys)-12))
	}
	return keys
}

func (s *appLogsState) start() tea.Cmd {
	if s == nil || s.started {
		return nil
	}
	s.started = true
	if len(s.streams) == 0 {
		return nil
	}

	ctx, cancel := context.WithCancel(context.Background()) //nolint:gosec // cancel is stored and called when leaving the view
	s.cancel = cancel
	s.streaming = true
	s.done = false
	s.err = nil
	s.ch = make(chan appLogLineMsg, 100)

	return tea.Batch(watchAppLogs(ctx, s.streams, s.ch), waitForAppLogLines(s.ch))
}

func (s *appLogsState) stop() {
	if s != nil && s.cancel != nil {
		s.cancel()
	}
}

func (s *appLogsState) handleLine(msg appLogLineMsg, bodyHeight, contentWidth int) tea.Cmd {
	if s == nil {
		return nil
	}
	s.lines = append(s.lines, msg.lines...)
	if s.follow {
		s.scroll = appLogMaxScroll(s.lines, bodyHeight, contentWidth)
	}
	return waitForAppLogLines(s.ch)
}

func (s *appLogsState) handleDone(msg appLogStreamDoneMsg) {
	if s == nil {
		return
	}
	s.streaming = false
	s.done = true
	if msg.err != nil {
		s.err = msg.err
	}
}

func (s *appLogsState) scrollUp() {
	if s == nil {
		return
	}
	s.follow = false
	if s.scroll > 0 {
		s.scroll--
	}
}

func (s *appLogsState) scrollDown(bodyHeight, contentWidth int) {
	if s == nil {
		return
	}
	s.follow = false
	s.scroll++
	if maxScroll := appLogMaxScroll(s.lines, bodyHeight, contentWidth); s.scroll > maxScroll {
		s.scroll = maxScroll
	}
}

func (s *appLogsState) pageUp(bodyHeight int) {
	if s == nil {
		return
	}
	s.follow = false
	s.scroll -= bodyHeight
	if s.scroll < 0 {
		s.scroll = 0
	}
}

func (s *appLogsState) pageDown(bodyHeight, contentWidth int) {
	if s == nil {
		return
	}
	s.follow = false
	s.scroll += bodyHeight
	if maxScroll := appLogMaxScroll(s.lines, bodyHeight, contentWidth); s.scroll > maxScroll {
		s.scroll = maxScroll
	}
}

func (s *appLogsState) toggleFollow(bodyHeight, contentWidth int) {
	if s == nil {
		return
	}
	s.follow = !s.follow
	if s.follow {
		s.scroll = appLogMaxScroll(s.lines, bodyHeight, contentWidth)
	}
}

func (s *appLogsState) copyText() string {
	if s == nil || len(s.lines) == 0 {
		return ""
	}
	return strings.Join(s.lines, "\n")
}

func watchAppLogs(ctx context.Context, streams []dataaccess.LogsStream, ch chan appLogLineMsg) tea.Cmd {
	return func() tea.Msg {
		var wg sync.WaitGroup
		var mu sync.Mutex
		var errs []string

		logsService := dataaccess.NewLogsService()
		for _, stream := range streams {
			stream := stream
			if strings.TrimSpace(stream.PodName) == "" {
				continue
			}

			wg.Add(1)
			go func() {
				defer wg.Done()

				reader, err := logsService.OpenLogStream(ctx, stream)
				if err != nil {
					mu.Lock()
					errs = append(errs, fmt.Sprintf("%s: %v", appLogSourceName(stream), err))
					mu.Unlock()
					return
				}
				defer reader.Close()

				scanner := bufio.NewScanner(reader)
				scanner.Buffer(make([]byte, 1024), 1024*1024)
				for scanner.Scan() {
					select {
					case <-ctx.Done():
						return
					default:
					}

					lines := formatAppLogPayload(appLogSourceName(stream), scanner.Text())
					if len(lines) == 0 {
						continue
					}

					select {
					case ch <- appLogLineMsg{lines: lines}:
					case <-ctx.Done():
						return
					}
				}
				if scanErr := scanner.Err(); scanErr != nil && ctx.Err() == nil {
					mu.Lock()
					errs = append(errs, fmt.Sprintf("%s: %v", appLogSourceName(stream), scanErr))
					mu.Unlock()
				}
			}()
		}

		wg.Wait()
		close(ch)
		if ctx.Err() != nil {
			return appLogStreamDoneMsg{}
		}
		if len(errs) > 0 && len(errs) >= len(streams) {
			return appLogStreamDoneMsg{err: errors.New(strings.Join(errs, "; "))}
		}
		return appLogStreamDoneMsg{}
	}
}

func waitForAppLogLines(ch chan appLogLineMsg) tea.Cmd {
	return func() tea.Msg {
		msg, ok := <-ch
		if !ok {
			return appLogStreamDoneMsg{}
		}
		return msg
	}
}

func appLogSourceName(stream dataaccess.LogsStream) string {
	if strings.TrimSpace(stream.PodName) != "" {
		if strings.TrimSpace(stream.ContainerName) != "" {
			return strings.TrimSpace(stream.PodName) + "/" + strings.TrimSpace(stream.ContainerName)
		}
		return strings.TrimSpace(stream.PodName)
	}
	return "app"
}

func formatAppLogPayload(source, payload string) []string {
	payload = strings.TrimRight(payload, "\r\n")
	if payload == "" {
		return nil
	}

	rawLines := strings.Split(payload, "\n")
	lines := make([]string, 0, len(rawLines))
	for _, line := range rawLines {
		line = strings.TrimRight(line, "\r")
		if line == "" {
			continue
		}
		lines = append(lines, fmt.Sprintf("[%s] %s", source, line))
	}
	return lines
}

func appLogMaxScroll(lines []string, bodyHeight, contentWidth int) int {
	bodyH := bodyHeight - 4
	if bodyH < 1 {
		bodyH = 1
	}
	maxCodeWidth := contentWidth - 9
	if maxCodeWidth < 20 {
		maxCodeWidth = 20
	}
	vlines := expandLinesToVisual(lines, maxCodeWidth)
	maxScroll := len(vlines) - bodyH
	if maxScroll < 0 {
		maxScroll = 0
	}
	return maxScroll
}

func renderAppLogsTab(state *appLogsState, logsFeatureError, spinnerView string, bodyHeight, contentWidth int) string {
	if state == nil || len(state.streams) == 0 {
		subtleStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
		msg := "No app log streams available for this resource."
		if logsFeatureError != "" {
			msg = logsFeatureError
		}
		return fmt.Sprintf("\n  %s\n", subtleStyle.Render(msg))
	}
	if state.err != nil && len(state.lines) == 0 {
		errStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("203"))
		return fmt.Sprintf("\n  %s\n", errStyle.Render(fmt.Sprintf("Error streaming app logs: %v", state.err)))
	}
	if !state.streaming && !state.done && len(state.lines) == 0 {
		return fmt.Sprintf("\n  %s Connecting to app logs...", spinnerView)
	}
	if len(state.lines) == 0 {
		subtleStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
		return fmt.Sprintf("\n  %s\n", subtleStyle.Render("Connected. Waiting for app logs..."))
	}

	var b strings.Builder
	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("255"))
	statusText := ""
	if state.streaming {
		statusText = " " + lipgloss.NewStyle().Foreground(lipgloss.Color("82")).Render("live")
	} else if state.done {
		statusText = " " + lipgloss.NewStyle().Foreground(lipgloss.Color("245")).Render("ended")
	}
	followText := ""
	if state.follow {
		followText = " " + lipgloss.NewStyle().Foreground(lipgloss.Color("82")).Render("[follow]")
	}
	fmt.Fprintf(&b, "  %s%s%s\n\n", headerStyle.Render(fmt.Sprintf("App Logs (%d lines)", len(state.lines))), statusText, followText)

	bodyH := bodyHeight - 4
	if bodyH < 1 {
		bodyH = 1
	}
	lineNumStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	maxCodeWidth := contentWidth - 9
	if maxCodeWidth < 20 {
		maxCodeWidth = 20
	}

	vlines := expandLinesToVisual(state.lines, maxCodeWidth)
	totalLines := len(vlines)
	scroll := state.scroll
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

	for i := scroll; i < end; i++ {
		vl := vlines[i]
		styled := highlightLogLine(vl.text)
		if vl.sourceNum > 0 {
			lineNum := lineNumStyle.Render(fmt.Sprintf("%4d", vl.sourceNum))
			fmt.Fprintf(&b, "  %s | %s\n", lineNum, styled)
		} else {
			fmt.Fprintf(&b, "       %s\n", styled)
		}
	}

	for i := end - scroll; i < bodyH; i++ {
		b.WriteString("\n")
	}

	if totalLines > bodyH {
		dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
		pos := "top"
		if scroll >= maxScroll {
			pos = "end"
		} else if scroll > 0 {
			pos = fmt.Sprintf("%d%%", (scroll*100)/maxScroll)
		}
		fmt.Fprintf(&b, "  %s\n", dimStyle.Render(fmt.Sprintf("[%d/%d %s]", end, totalLines, pos)))
	}

	return b.String()
}
