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
	lines      []string
	generation int
}

type appLogStreamDoneMsg struct {
	err        error
	generation int
}

type appLogPodStreams struct {
	name    string
	streams []dataaccess.LogsStream
}

type appLogsState struct {
	streams     []dataaccess.LogsStream
	pods        []appLogPodStreams
	selectedPod int
	lines       []string
	ch          chan appLogLineMsg
	cancel      context.CancelFunc

	scroll     int
	follow     bool
	streaming  bool
	done       bool
	err        error
	started    bool
	generation int
}

func newAppLogsState(streams []dataaccess.LogsStream) *appLogsState {
	return &appLogsState{
		streams: append([]dataaccess.LogsStream(nil), streams...),
		pods:    appLogPodStreamsForStreams(streams),
		ch:      make(chan appLogLineMsg, 100),
		follow:  true,
	}
}

func appLogPodStreamsForStreams(streams []dataaccess.LogsStream) []appLogPodStreams {
	if len(streams) == 0 {
		return nil
	}

	byPod := make(map[string][]dataaccess.LogsStream)
	for _, stream := range streams {
		podName := strings.TrimSpace(stream.PodName)
		if podName == "" {
			continue
		}
		byPod[podName] = append(byPod[podName], stream)
	}
	if len(byPod) == 0 {
		return nil
	}

	podNames := make([]string, 0, len(byPod))
	for podName := range byPod {
		podNames = append(podNames, podName)
	}
	sort.Strings(podNames)

	pods := make([]appLogPodStreams, 0, len(podNames))
	for _, podName := range podNames {
		podStreams := append([]dataaccess.LogsStream(nil), byPod[podName]...)
		sort.Slice(podStreams, func(i, j int) bool {
			return podStreams[i].ContainerName < podStreams[j].ContainerName
		})
		pods = append(pods, appLogPodStreams{name: podName, streams: podStreams})
	}
	return pods
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
	streams := s.selectedStreams()
	if len(streams) == 0 {
		return nil
	}

	ctx, cancel := context.WithCancel(context.Background()) //nolint:gosec // cancel is stored and called when leaving the view
	s.cancel = cancel
	s.streaming = true
	s.done = false
	s.err = nil
	s.ch = make(chan appLogLineMsg, 100)

	return tea.Batch(watchAppLogs(ctx, streams, s.ch, s.generation), waitForAppLogLines(s.ch, s.generation))
}

func (s *appLogsState) stop() {
	if s != nil && s.cancel != nil {
		s.cancel()
	}
}

func (s *appLogsState) selectedStreams() []dataaccess.LogsStream {
	if s == nil {
		return nil
	}
	if len(s.pods) == 0 {
		return s.streams
	}
	if s.selectedPod < 0 || s.selectedPod >= len(s.pods) {
		s.selectedPod = 0
	}
	return s.pods[s.selectedPod].streams
}

func (s *appLogsState) hasMultiplePods() bool {
	return s != nil && len(s.pods) > 1
}

func (s *appLogsState) selectPreviousPod() tea.Cmd {
	if !s.hasMultiplePods() {
		return nil
	}
	s.selectedPod = (s.selectedPod - 1 + len(s.pods)) % len(s.pods)
	return s.restartSelectedPod()
}

func (s *appLogsState) selectNextPod() tea.Cmd {
	if !s.hasMultiplePods() {
		return nil
	}
	s.selectedPod = (s.selectedPod + 1) % len(s.pods)
	return s.restartSelectedPod()
}

func (s *appLogsState) restartSelectedPod() tea.Cmd {
	if s == nil {
		return nil
	}
	s.stop()
	s.generation++
	s.lines = nil
	s.scroll = 0
	s.follow = true
	s.streaming = false
	s.done = false
	s.err = nil
	s.started = false
	s.cancel = nil
	s.ch = make(chan appLogLineMsg, 100)
	return s.start()
}

func (s *appLogsState) handleLine(msg appLogLineMsg, bodyHeight, contentWidth int) tea.Cmd {
	if s == nil {
		return nil
	}
	if msg.generation != s.generation {
		return nil
	}
	s.lines = append(s.lines, msg.lines...)
	if s.follow {
		s.scroll = appLogMaxScroll(s.lines, bodyHeight, contentWidth, s.selectorRows())
	}
	return waitForAppLogLines(s.ch, s.generation)
}

func (s *appLogsState) handleDone(msg appLogStreamDoneMsg) {
	if s == nil {
		return
	}
	if msg.generation != s.generation {
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
	if maxScroll := appLogMaxScroll(s.lines, bodyHeight, contentWidth, s.selectorRows()); s.scroll > maxScroll {
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
	if maxScroll := appLogMaxScroll(s.lines, bodyHeight, contentWidth, s.selectorRows()); s.scroll > maxScroll {
		s.scroll = maxScroll
	}
}

func (s *appLogsState) toggleFollow(bodyHeight, contentWidth int) {
	if s == nil {
		return
	}
	s.follow = !s.follow
	if s.follow {
		s.scroll = appLogMaxScroll(s.lines, bodyHeight, contentWidth, s.selectorRows())
	}
}

func (s *appLogsState) copyText() string {
	if s == nil || len(s.lines) == 0 {
		return ""
	}
	return strings.Join(s.lines, "\n")
}

func (s *appLogsState) selectorRows() int {
	if !s.hasMultiplePods() {
		return 0
	}
	return 2
}

func appLogsFooterHelp(state *appLogsState, includeTabSwitch bool) string {
	parts := []string{"↑↓/pgup/pgdn: scroll"}
	if state != nil && state.hasMultiplePods() {
		parts = append(parts, "←→: switch pods")
	}
	parts = append(parts, "f: toggle follow", "y: copy")
	if includeTabSwitch {
		parts = append(parts, "tab/shift+tab: switch tabs")
	}
	parts = append(parts, "esc: back", "q: quit")
	return strings.Join(parts, "  ")
}

func watchAppLogs(ctx context.Context, streams []dataaccess.LogsStream, ch chan appLogLineMsg, generation int) tea.Cmd {
	return func() tea.Msg {
		var wg sync.WaitGroup
		var mu sync.Mutex
		var errs []string
		streamCount := 0

		logsService := dataaccess.NewLogsService()
		for _, stream := range streams {
			stream := stream
			if strings.TrimSpace(stream.PodName) == "" {
				continue
			}

			streamCount++
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
					case ch <- appLogLineMsg{lines: lines, generation: generation}:
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
			return appLogStreamDoneMsg{generation: generation}
		}
		if len(errs) > 0 && len(errs) >= streamCount {
			return appLogStreamDoneMsg{err: errors.New(strings.Join(errs, "; ")), generation: generation}
		}
		return appLogStreamDoneMsg{generation: generation}
	}
}

func waitForAppLogLines(ch chan appLogLineMsg, generation int) tea.Cmd {
	return func() tea.Msg {
		msg, ok := <-ch
		if !ok {
			return appLogStreamDoneMsg{generation: generation}
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

func appLogMaxScroll(lines []string, bodyHeight, contentWidth, selectorRows int) int {
	bodyH := appLogViewportHeight(bodyHeight, selectorRows)
	maxCodeWidth := appLogMaxCodeWidth(contentWidth)
	vlines := expandLinesToVisual(lines, maxCodeWidth)
	maxScroll := len(vlines) - bodyH
	if maxScroll < 0 {
		maxScroll = 0
	}
	return maxScroll
}

func appLogViewportHeight(bodyHeight, selectorRows int) int {
	bodyH := bodyHeight - 4 - selectorRows
	if bodyH < 1 {
		bodyH = 1
	}
	return bodyH
}

func appLogMaxCodeWidth(contentWidth int) int {
	maxCodeWidth := contentWidth - 9
	if maxCodeWidth < 20 {
		maxCodeWidth = 20
	}
	return maxCodeWidth
}

func renderAppLogPodSelector(state *appLogsState) string {
	if state == nil || !state.hasMultiplePods() {
		return ""
	}

	var b strings.Builder
	dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	activeStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("230")).Background(lipgloss.Color("63")).Padding(0, 1)
	inactiveStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("252")).Background(lipgloss.Color("238")).Padding(0, 1)

	fmt.Fprintf(&b, "  %s\n", dimStyle.Render("Pods: use ←/→ or h/l to switch"))

	var rendered []string
	for i, pod := range state.pods {
		label := pod.name
		if len(pod.streams) > 1 {
			label = fmt.Sprintf("%s (%d)", pod.name, len(pod.streams))
		}
		if i == state.selectedPod {
			rendered = append(rendered, activeStyle.Render(label))
			continue
		}
		rendered = append(rendered, inactiveStyle.Render(label))
	}

	fmt.Fprintf(&b, "  %s\n", strings.Join(rendered, " "))
	return b.String()
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
		return renderAppLogPodSelector(state) + fmt.Sprintf("\n  %s\n", errStyle.Render(fmt.Sprintf("Error streaming app logs: %v", state.err)))
	}
	if !state.streaming && !state.done && len(state.lines) == 0 {
		return renderAppLogPodSelector(state) + fmt.Sprintf("\n  %s Connecting to app logs...", spinnerView)
	}
	if len(state.lines) == 0 {
		subtleStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
		return renderAppLogPodSelector(state) + fmt.Sprintf("\n  %s\n", subtleStyle.Render("Connected. Waiting for app logs..."))
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
	title := fmt.Sprintf("App Logs (%d lines)", len(state.lines))
	if len(state.pods) > 0 {
		title = fmt.Sprintf("App Logs: %s (%d lines)", state.pods[state.selectedPod].name, len(state.lines))
	}
	fmt.Fprintf(&b, "  %s%s%s\n", headerStyle.Render(title), statusText, followText)
	if selector := renderAppLogPodSelector(state); selector != "" {
		b.WriteString(selector)
	} else {
		b.WriteString("\n")
	}

	bodyH := appLogViewportHeight(bodyHeight, state.selectorRows())
	lineNumStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	maxCodeWidth := appLogMaxCodeWidth(contentWidth)

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
