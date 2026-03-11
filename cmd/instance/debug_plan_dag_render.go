package instance

import (
	"fmt"
	"math"
	"sort"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

type planDAGPlacement struct {
	col    int
	x      int
	y      int
	width  int
	height int
}

// dagRenderResult bundles the rendered lines with node placement metadata
// so the caller can auto-scroll to keep the selected node visible.
type dagRenderResult struct {
	lines      []string
	placements map[string]planDAGPlacement
	prefixRows int // number of prefix lines (errors, warnings) before the diagram
}

type planDAGLayout struct {
	levels [][]string
	pos    map[string]int
}

const (
	boxTopLeft     = '╭'
	boxTopRight    = '╮'
	boxBottomLeft  = '╰'
	boxBottomRight = '╯'
	boxHLine       = '─'
	boxVLine       = '│'
	boxCross       = '┼'
	arrowHead      = '▶'
	dotRune        = '·'

	// Connector-specific corners — heavy weight for visibility
	connHLine     = '━'
	connVLine     = '┃'
	connDownRight = '┏'
	connDownLeft  = '┓'
	connUpRight   = '┗'
	connUpLeft    = '┛'
	connCross     = '╋'
)

func orderPlanLevels(plan *PlanDAG) planDAGLayout {
	levels := make([][]string, len(plan.Levels))
	for i, level := range plan.Levels {
		ids := append([]string{}, level...)
		sort.Slice(ids, func(i, j int) bool {
			return labelForNode(plan, ids[i]) < labelForNode(plan, ids[j])
		})
		levels[i] = ids
	}

	incoming, outgoing := buildPlanAdjacency(plan.Edges)
	pos := make(map[string]int)
	updatePositions(levels, pos)

	for pass := 0; pass < 2; pass++ {
		for levelIdx := 1; levelIdx < len(levels); levelIdx++ {
			levels[levelIdx] = sortLevelByBarycenter(levels[levelIdx], incoming, pos, plan)
			updatePositions(levels, pos)
		}
		for levelIdx := len(levels) - 2; levelIdx >= 0; levelIdx-- {
			levels[levelIdx] = sortLevelByBarycenter(levels[levelIdx], outgoing, pos, plan)
			updatePositions(levels, pos)
		}
	}

	return planDAGLayout{
		levels: levels,
		pos:    pos,
	}
}

func buildPlanAdjacency(edges []PlanDAGEdge) (map[string][]string, map[string][]string) {
	incoming := make(map[string][]string)
	outgoing := make(map[string][]string)
	for _, edge := range edges {
		incoming[edge.To] = append(incoming[edge.To], edge.From)
		outgoing[edge.From] = append(outgoing[edge.From], edge.To)
	}
	return incoming, outgoing
}

func updatePositions(levels [][]string, pos map[string]int) {
	for _, level := range levels {
		for i, id := range level {
			pos[id] = i
		}
	}
}

func sortLevelByBarycenter(level []string, adjacency map[string][]string, pos map[string]int, plan *PlanDAG) []string {
	type nodeOrder struct {
		id    string
		bary  float64
		label string
	}

	orders := make([]nodeOrder, 0, len(level))
	for idx, id := range level {
		neighbors := adjacency[id]
		bary := float64(idx)
		if len(neighbors) > 0 {
			var sum float64
			count := 0
			for _, neighbor := range neighbors {
				if neighborPos, ok := pos[neighbor]; ok {
					sum += float64(neighborPos)
					count++
				}
			}
			if count > 0 {
				bary = sum / float64(count)
			}
		}
		orders = append(orders, nodeOrder{
			id:    id,
			bary:  bary,
			label: labelForNode(plan, id),
		})
	}

	sort.SliceStable(orders, func(i, j int) bool {
		if orders[i].bary == orders[j].bary {
			return orders[i].label < orders[j].label
		}
		return orders[i].bary < orders[j].bary
	})

	sorted := make([]string, len(orders))
	for i, order := range orders {
		sorted[i] = order.id
	}
	return sorted
}

func labelForNode(plan *PlanDAG, id string) string {
	if node, ok := plan.Nodes[id]; ok {
		return nodeLabel(node)
	}
	return id
}

func progressForNode(plan *PlanDAG, node PlanDAGNode) (ResourceProgress, bool) {
	if plan == nil {
		return ResourceProgress{}, false
	}
	if plan.ProgressByID != nil {
		if progress, ok := plan.ProgressByID[node.ID]; ok {
			return progress, true
		}
	}
	if node.Key != "" && plan.ProgressByKey != nil {
		if progress, ok := plan.ProgressByKey[node.Key]; ok {
			return progress, true
		}
	}
	if node.Name != "" && plan.ProgressByName != nil {
		if progress, ok := plan.ProgressByName[node.Name]; ok {
			return progress, true
		}
	}
	return ResourceProgress{}, false
}

func wrapText(text string, width int) []string {
	if width <= 0 {
		return []string{text}
	}
	words := strings.Fields(text)
	if len(words) == 0 {
		return []string{""}
	}
	lines := []string{}
	line := words[0]
	for _, word := range words[1:] {
		if len(line)+1+len(word) > width {
			lines = append(lines, line)
			line = word
			continue
		}
		line += " " + word
	}
	lines = append(lines, line)
	return lines
}

type styledCell struct {
	ch    rune
	style lipgloss.Style
}

type dagCanvas struct {
	width  int
	height int
	cells  [][]styledCell
}

type nodeCard struct {
	title            string
	meta1            string
	meta2            string
	icon             rune
	iconStyle        lipgloss.Style
	keyLabel         string
	keyValue         string
	breakpointStatus string
	theme            cardTheme
	progress         ResourceProgress
	hasProgress      bool
	progressLoading  bool
	spinnerRune      rune
	deps             []depEntry // parent dependencies with status
	expanded         bool       // whether to show deps checklist
}

type depEntry struct {
	name   string
	status string // "completed", "running", "pending", "failed", etc.
}

type cardTheme struct {
	bg     string
	border string
	title  string
	label  string
	value  string
	icon   string
}

func renderPlanDAGStyledWithSelection(plan *PlanDAG, width int, selectedNodeID string, expandedNodes map[string]bool, highlightDeps bool) dagRenderResult {
	if plan == nil {
		style := lipgloss.NewStyle().Foreground(lipgloss.Color("1")).Bold(true)
		return dagRenderResult{lines: []string{style.Render("Deployment plan unavailable")}}
	}

	if width <= 0 {
		width = 120
	}

	subtleStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	warnStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("1")).Bold(true)

	var prefixLines []string

	if len(plan.Errors) > 0 {
		prefixLines = append(prefixLines, warnStyle.Render("Warnings:"))
		for _, err := range plan.Errors {
			for _, line := range wrapText(err, width-4) {
				prefixLines = append(prefixLines, "  "+subtleStyle.Render(line))
			}
		}
		prefixLines = append(prefixLines, "")
	}

	if plan.HasCycle {
		prefixLines = append(prefixLines, warnStyle.Render("Cycle detected in dependencies; layout may be incomplete."))
		prefixLines = append(prefixLines, "")
	}

	result := drawPlanDAGStyled(plan, width, selectedNodeID, expandedNodes, highlightDeps)
	result.prefixRows = len(prefixLines)
	result.lines = append(prefixLines, result.lines...)

	return result
}

func drawPlanDAGStyled(plan *PlanDAG, _ int, selectedNodeID string, expandedNodes map[string]bool, highlightDeps bool) dagRenderResult {
	layout := orderPlanLevels(plan)
	levels := layout.levels
	if len(levels) == 0 {
		return dagRenderResult{lines: []string{"No resources found for this plan version."}}
	}

	if expandedNodes == nil {
		expandedNodes = make(map[string]bool)
	}

	// Build reverse dependency map: nodeID → list of parent node IDs
	parentIDs := make(map[string][]string)
	for _, edge := range plan.Edges {
		parentIDs[edge.To] = append(parentIDs[edge.To], edge.From)
	}

	cards := make(map[string]nodeCard)
	maxInner := 0
	for id, node := range plan.Nodes {
		progress, ok := progressForNode(plan, node)
		breakpointStatus, _ := breakpointStatusForNode(plan, node)
		card := buildNodeCard(node, progress, ok, breakpointStatus)

		// Build dependency entries from parents
		for _, parentID := range parentIDs[id] {
			parentNode, exists := plan.Nodes[parentID]
			if !exists {
				continue
			}
			dep := depEntry{name: nodeLabel(parentNode), status: "pending"}
			if parentProgress, pOk := progressForNode(plan, parentNode); pOk {
				dep.status = strings.ToLower(parentProgress.Status)
			}
			card.deps = append(card.deps, dep)
		}
		// Sort deps by name for stable display
		sort.Slice(card.deps, func(i, j int) bool {
			return card.deps[i].name < card.deps[j].name
		})

		card.expanded = expandedNodes[id]

		// Mark all nodes as loading while progress fetch is in flight
		if plan.ProgressLoading {
			card.progressLoading = true
			card.hasProgress = true
			frames := []rune{'⠋', '⠙', '⠹', '⠸', '⠼', '⠴', '⠦', '⠧', '⠇', '⠏'}
			card.spinnerRune = frames[plan.SpinnerTick%len(frames)]
		}
		cards[id] = card

		line1 := 2 + len([]rune(card.title))
		line2 := len([]rune(card.meta1))
		line3 := len([]rune(card.meta2))
		maxLine := maxInt(line1, maxInt(line2, line3))
		if maxLine > maxInner {
			maxInner = maxLine
		}
	}

	innerWidth := clampInt(maxInner, 22, 36)
	cardWidth := innerWidth + 2
	anyProgress := false
	for _, card := range cards {
		if card.hasProgress {
			anyProgress = true
			break
		}
	}
	baseCardHeight := 5
	if anyProgress {
		baseCardHeight = 6
	}
	hGap := 6
	vGap := 2
	if len(levels) > 4 {
		hGap = 4
	}

	// Compute per-node card height (base + expanded dep lines + collapsed indicator)
	nodeHeight := make(map[string]int)
	for id, card := range cards {
		h := baseCardHeight
		if hasHitBreakpoint(card.breakpointStatus) {
			h++
		}
		if len(card.deps) > 0 && !card.expanded {
			h++ // collapsed: one line showing "▸ N dependencies"
		} else if card.expanded && len(card.deps) > 0 {
			h += len(card.deps) + 1 // header line + one line per dep
		}
		nodeHeight[id] = h
	}

	// Outer border padding
	outerPadX := 2
	outerPadY := 1

	// Compute max column height (sum of node heights + gaps in that column)
	maxColHeight := 0
	for _, level := range levels {
		colH := 0
		for i, nodeID := range level {
			colH += nodeHeight[nodeID]
			if i > 0 {
				colH += vGap
			}
		}
		if colH > maxColHeight {
			maxColHeight = colH
		}
	}

	innerTotalWidth := len(levels)*cardWidth + (len(levels)-1)*hGap
	numSeparators := len(levels) - 1

	totalWidth := innerTotalWidth + 2*outerPadX + 2 + numSeparators
	totalHeight := maxColHeight + 2*outerPadY + 2
	if totalWidth < cardWidth+2*outerPadX+2 {
		totalWidth = cardWidth + 2*outerPadX + 2
	}
	if totalHeight < baseCardHeight+2*outerPadY+2 {
		totalHeight = baseCardHeight + 2*outerPadY + 2
	}

	canvas := newDagCanvas(totalWidth, totalHeight)
	canvas.fillDots()

	// Draw outer border
	outerBorderStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	canvas.drawBorder(0, 0, totalWidth, totalHeight, outerBorderStyle)

	// Compute x-offset for each level column
	offsetX := outerPadX + 1
	offsetY := outerPadY + 1

	levelX := make([]int, len(levels))
	for col := range levels {
		levelX[col] = offsetX + col*(cardWidth+hGap) + col
	}

	// Place nodes with cumulative Y per column (dynamic heights)
	placements := make(map[string]planDAGPlacement)
	for col, level := range levels {
		curY := offsetY
		for _, nodeID := range level {
			placements[nodeID] = planDAGPlacement{col: col, x: levelX[col], y: curY, width: cardWidth, height: nodeHeight[nodeID]}
			curY += nodeHeight[nodeID] + vGap
		}
	}

	// Draw level separator dotted lines
	separatorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("238"))
	for col := 0; col < len(levels)-1; col++ {
		sepX := levelX[col] + cardWidth + hGap/2
		for y := 1; y < totalHeight-1; y++ {
			canvas.set(sepX, y, '┊', separatorStyle)
		}
	}

	// Compute ancestor highlight set: selected node + all its ancestors
	highlightSet := make(map[string]bool)
	if highlightDeps && selectedNodeID != "" {
		highlightSet[selectedNodeID] = true
		for id := range findAncestors(selectedNodeID, parentIDs) {
			highlightSet[id] = true
		}
	}

	connectorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("172"))
	dimConnStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("239"))

	// Draw dimmed connectors first, then highlighted ones on top,
	// so highlighted lines are never overwritten by dimmed ones.
	for pass := 0; pass < 2; pass++ {
		for _, edge := range plan.Edges {
			from, okFrom := placements[edge.From]
			to, okTo := placements[edge.To]
			if !okFrom || !okTo {
				continue
			}
			if from.col >= to.col {
				continue
			}
			fromH := nodeHeight[edge.From]
			toH := nodeHeight[edge.To]
			isHighlighted := len(highlightSet) == 0 || (highlightSet[edge.From] && highlightSet[edge.To])
			// pass 0: dimmed only, pass 1: highlighted only
			if pass == 0 && isHighlighted {
				continue
			}
			if pass == 1 && !isHighlighted {
				continue
			}
			style := connectorStyle
			if !isHighlighted {
				style = dimConnStyle
			}
			drawConnectorDynamic(canvas, from, to, cardWidth, fromH, toH, style)
		}
	}

	for _, level := range levels {
		for _, nodeID := range level {
			pos := placements[nodeID]
			card := cards[nodeID]
			h := nodeHeight[nodeID]
			isSelected := nodeID == selectedNodeID
			isDimmed := len(highlightSet) > 0 && !highlightSet[nodeID]
			drawCard(canvas, pos.x, pos.y, cardWidth, h, card, isSelected, isDimmed)
		}
	}

	return dagRenderResult{lines: canvas.render(), placements: placements}
}

func buildNodeCard(node PlanDAGNode, progress ResourceProgress, hasProgress bool, breakpointStatus string) nodeCard {
	label := nodeLabel(node)
	typeTag := formatTypeTag(node.Type)
	theme := themeForType(typeTag)
	iconRune, iconStyle := iconForType(typeTag, theme)

	meta1 := fmt.Sprintf("Type: %s", typeTag)
	keyLabel := "Key"
	keyValue := node.Key
	if keyValue == "" {
		keyLabel = "ID"
		keyValue = shortID(node.ID)
	}
	meta2 := fmt.Sprintf("%s: %s", keyLabel, keyValue)

	return nodeCard{
		title:            label,
		meta1:            meta1,
		meta2:            meta2,
		icon:             iconRune,
		iconStyle:        iconStyle,
		keyLabel:         keyLabel,
		keyValue:         keyValue,
		breakpointStatus: breakpointStatus,
		theme:            theme,
		progress:         progress,
		hasProgress:      hasProgress,
	}
}

func formatTypeTag(resourceType string) string {
	if resourceType == "" {
		return "Resource"
	}
	lower := strings.ToLower(resourceType)
	switch {
	case strings.Contains(lower, "helm"):
		return "Helm"
	case strings.Contains(lower, "terraform"):
		return "Terraform"
	case strings.Contains(lower, "kustomize"):
		return "Kustomize"
	default:
		if len(lower) == 0 {
			return "Resource"
		}
		return strings.ToUpper(lower[:1]) + lower[1:]
	}
}

func themeForType(_ string) cardTheme {
	return cardTheme{
		bg:     "",
		border: "245",
		title:  "255",
		label:  "245",
		value:  "252",
		icon:   "255",
	}
}

func iconForType(tag string, theme cardTheme) (rune, lipgloss.Style) {
	style := lipgloss.NewStyle().Foreground(lipgloss.Color(theme.icon)).Bold(true)
	switch tag {
	case "Helm":
		return 'H', style
	case "Terraform":
		return 'T', style
	case "Kustomize":
		return 'K', style
	default:
		return 'R', style
	}
}

// findAncestors returns the set of all ancestor node IDs for the given node (not including itself).
func findAncestors(nodeID string, parentIDs map[string][]string) map[string]bool {
	ancestors := make(map[string]bool)
	queue := []string{nodeID}
	visited := map[string]bool{nodeID: true}
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		for _, parent := range parentIDs[current] {
			if !visited[parent] {
				visited[parent] = true
				ancestors[parent] = true
				queue = append(queue, parent)
			}
		}
	}
	return ancestors
}

func drawCard(canvas *dagCanvas, x, y, width, height int, card nodeCard, selected bool, dimmed bool) {
	if width < 4 || height < 3 {
		return
	}

	inner := width - 2
	noStyle := lipgloss.NewStyle()
	borderStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(card.theme.border))
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(card.theme.title))
	labelStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(card.theme.label))
	valueStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(card.theme.value))

	if dimmed {
		dimColor := lipgloss.Color("239")
		borderStyle = lipgloss.NewStyle().Foreground(dimColor)
		titleStyle = lipgloss.NewStyle().Foreground(dimColor)
		labelStyle = lipgloss.NewStyle().Foreground(dimColor)
		valueStyle = lipgloss.NewStyle().Foreground(dimColor)
	}

	if selected {
		borderStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("205")).Bold(true)
		titleStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("230"))
		// Draw thick outer glow border 1 cell outside the card
		glowStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("205")).Bold(true)
		gx, gy, gw, gh := x-1, y-1, width+2, height+2
		// Heavy horizontal lines top/bottom
		for col := 0; col < gw; col++ {
			canvas.set(gx+col, gy, '━', glowStyle)
			canvas.set(gx+col, gy+gh-1, '━', glowStyle)
		}
		// Heavy vertical lines left/right
		for row := 1; row < gh-1; row++ {
			canvas.set(gx, gy+row, '┃', glowStyle)
			canvas.set(gx+gw-1, gy+row, '┃', glowStyle)
		}
		// Heavy corners
		canvas.set(gx, gy, '┏', glowStyle)
		canvas.set(gx+gw-1, gy, '┓', glowStyle)
		canvas.set(gx, gy+gh-1, '┗', glowStyle)
		canvas.set(gx+gw-1, gy+gh-1, '┛', glowStyle)
	}

	canvas.drawBorder(x, y, width, height, borderStyle)

	for row := 1; row < height-1; row++ {
		for col := 1; col < width-1; col++ {
			canvas.set(x+col, y+row, ' ', noStyle)
		}
	}

	offset := 0
	if card.hasProgress {
		offset = 1
		drawProgressBar(canvas, x+1, y+1, inner, card, noStyle, dimmed)
	}

	titleY := y + 1 + offset
	typeY := y + 2 + offset
	keyY := y + 3 + offset

	contentX := x + 1
	remaining := inner

	isFailed := card.hasProgress && (card.progress.Status == "failed" || card.progress.Status == "error")
	if isFailed && remaining >= 2 {
		failStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("203")).Bold(true)
		if dimmed {
			failStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("239"))
		}
		canvas.set(contentX, titleY, '!', failStyle)
		contentX++
		remaining--
		canvas.set(contentX, titleY, ' ', noStyle)
		contentX++
		remaining--
	}

	if remaining > 0 {
		titleWidth := remaining
		if badgeText, badgeStyle, showBadge := breakpointBadge(card.breakpointStatus, dimmed); showBadge {
			badgeWidth := len([]rune(badgeText))
			if badgeWidth < inner {
				badgeX := x + 1 + inner - badgeWidth
				writeText(canvas, badgeX, titleY, badgeText, badgeStyle, badgeWidth)

				if badgeWidth+1 < titleWidth {
					titleWidth -= badgeWidth + 1
				} else {
					titleWidth = 0
				}
			}
		}

		if titleWidth > 0 {
			writeText(canvas, contentX, titleY, card.title, titleStyle, titleWidth)
		}
	}

	writeLabelValue(canvas, x+1, typeY, "Type:", strings.TrimPrefix(card.meta1, "Type: "), inner, labelStyle, valueStyle)
	writeLabelValue(canvas, x+1, keyY, card.keyLabel+":", card.keyValue, inner, labelStyle, valueStyle)

	// Dependency checklist
	dimColor := lipgloss.Color("239")
	depsStartY := keyY + 1
	if hasHitBreakpoint(card.breakpointStatus) {
		breakpointStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("203")).Bold(true)
		if dimmed {
			breakpointStyle = lipgloss.NewStyle().Foreground(dimColor)
		}
		writeText(canvas, x+1, depsStartY, breakpointPopDownLine(inner), breakpointStyle, inner)
		depsStartY++
	}

	if len(card.deps) > 0 {
		if card.expanded {
			// Header line
			depsHeaderStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("245")).Bold(true)
			if dimmed {
				depsHeaderStyle = lipgloss.NewStyle().Foreground(dimColor)
			}
			writeText(canvas, x+1, depsStartY, "▾ Depends on:", depsHeaderStyle, inner)
			depsStartY++
			// One line per dependency with status icon
			for i, dep := range card.deps {
				depY := depsStartY + i
				if depY >= y+height-1 {
					break
				}
				icon, iconSt := depStatusIcon(dep.status)
				if dimmed {
					iconSt = lipgloss.NewStyle().Foreground(dimColor)
				}
				canvas.set(x+2, depY, icon, iconSt)
				depNameStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
				if dimmed {
					depNameStyle = lipgloss.NewStyle().Foreground(dimColor)
				}
				writeText(canvas, x+4, depY, dep.name, depNameStyle, inner-3)
			}
		} else {
			// Collapsed: show summary
			collapsedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
			summary := fmt.Sprintf("▸ %d deps", len(card.deps))
			// Show count of completed vs total
			done := 0
			for _, dep := range card.deps {
				if dep.status == "completed" || dep.status == "success" {
					done++
				}
			}
			if dimmed {
				collapsedStyle = lipgloss.NewStyle().Foreground(dimColor)
			} else if done == len(card.deps) {
				summary = fmt.Sprintf("▸ %d deps ✓", len(card.deps))
				collapsedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("82"))
			} else {
				summary = fmt.Sprintf("▸ %d/%d deps ready", done, len(card.deps))
			}
			writeText(canvas, x+1, depsStartY, summary, collapsedStyle, inner)
		}
	}
}

func depStatusIcon(status string) (rune, lipgloss.Style) {
	switch strings.ToLower(status) {
	case "completed", "success":
		return '✓', lipgloss.NewStyle().Foreground(lipgloss.Color("82"))
	case "failed", "error":
		return '✗', lipgloss.NewStyle().Foreground(lipgloss.Color("203"))
	case "running", "in_progress", "in-progress", "started":
		return '●', lipgloss.NewStyle().Foreground(lipgloss.Color("220"))
	default:
		return '○', lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	}
}

func breakpointBadge(status string, dimmed bool) (string, lipgloss.Style, bool) {
	if !hasHitBreakpoint(status) {
		return "", lipgloss.NewStyle(), false
	}

	style := lipgloss.NewStyle().Foreground(lipgloss.Color("220")).Bold(true)
	if dimmed {
		style = lipgloss.NewStyle().Foreground(lipgloss.Color("239"))
	}
	return "BREAKPOINT", style, true
}

func breakpointPopDownLine(width int) string {
	const label = "breakpoint hit"
	if width <= len(label) {
		return label
	}

	// Format: "└─ breakpoint hit ───┘"
	core := "─ " + label + " "
	if width <= len(core)+1 {
		return label
	}

	remaining := width - 1 - len(core) - 1
	if remaining < 1 {
		remaining = 1
	}

	return "└" + core + strings.Repeat("─", remaining) + "┘"
}

func drawProgressBar(canvas *dagCanvas, x, y, width int, card nodeCard, bgStyle lipgloss.Style, dimmed bool) {
	if width <= 0 {
		return
	}

	if !card.hasProgress {
		return
	}

	// Show loading spinner while progress is being fetched
	if card.progressLoading {
		shimmerStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
		spinnerStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("205"))
		if dimmed {
			dimColor := lipgloss.Color("239")
			shimmerStyle = lipgloss.NewStyle().Foreground(dimColor)
			spinnerStyle = lipgloss.NewStyle().Foreground(dimColor)
		}
		r := card.spinnerRune
		if r == 0 {
			r = '⠋'
		}
		// bar of ░ then space then spinner rune
		barWidth := width - 2
		if barWidth < 1 {
			barWidth = 1
		}
		for i := 0; i < barWidth; i++ {
			canvas.set(x+i, y, '░', shimmerStyle)
		}
		canvas.set(x+barWidth, y, ' ', bgStyle)
		canvas.set(x+barWidth+1, y, r, spinnerStyle)
		return
	}

	progress := card.progress
	fillStyle, emptyStyle, textStyle := progressStyles(card.theme, progress.Status)

	if dimmed {
		dimColor := lipgloss.Color("239")
		fillStyle = lipgloss.NewStyle().Foreground(dimColor)
		emptyStyle = lipgloss.NewStyle().Foreground(dimColor)
		textStyle = lipgloss.NewStyle().Foreground(dimColor)
	}

	label := fmt.Sprintf("%3d%%", clampInt(progress.Percent, 0, 100))
	labelWidth := len([]rune(label))

	if width <= labelWidth+1 {
		writeText(canvas, x, y, label, textStyle, width)
		return
	}

	barWidth := width - labelWidth - 1
	filled := int(math.Round(float64(barWidth) * float64(progress.Percent) / 100))
	if filled < 0 {
		filled = 0
	}
	if filled > barWidth {
		filled = barWidth
	}

	for i := 0; i < barWidth; i++ {
		ch := '░'
		style := emptyStyle
		if i < filled {
			ch = '█'
			style = fillStyle
		}
		canvas.set(x+i, y, ch, style)
	}
	canvas.set(x+barWidth, y, ' ', bgStyle)
	writeText(canvas, x+barWidth+1, y, label, textStyle, labelWidth)
}

func progressStyles(theme cardTheme, status string) (lipgloss.Style, lipgloss.Style, lipgloss.Style) {
	base := lipgloss.NewStyle()
	empty := base.Foreground(lipgloss.Color(theme.label))
	text := base.Foreground(lipgloss.Color(theme.title)).Bold(true)

	fillColor := theme.value
	switch strings.ToLower(status) {
	case "completed", "success":
		fillColor = "82"
	case "failed", "error", "cancelled", "canceled":
		fillColor = "203"
	case "running", "in_progress", "in-progress", "started":
		fillColor = "220"
	case "pending":
		fillColor = theme.value
	}
	fill := base.Foreground(lipgloss.Color(fillColor))
	return fill, empty, text
}

func writeLabelValue(canvas *dagCanvas, x, y int, label, value string, maxWidth int, labelStyle, valueStyle lipgloss.Style) {
	used := writeText(canvas, x, y, label, labelStyle, maxWidth)
	x += used
	maxWidth -= used
	if maxWidth <= 0 {
		return
	}
	canvas.set(x, y, ' ', lipgloss.Style{})
	x++
	maxWidth--
	if maxWidth <= 0 {
		return
	}
	writeText(canvas, x, y, value, valueStyle, maxWidth)
}

func writeText(canvas *dagCanvas, x, y int, text string, style lipgloss.Style, maxWidth int) int {
	runes := []rune(text)
	if maxWidth > 0 && len(runes) > maxWidth {
		runes = runes[:maxWidth]
	}
	for i, r := range runes {
		canvas.set(x+i, y, r, style)
	}
	return len(runes)
}

func drawConnectorDynamic(canvas *dagCanvas, from, to planDAGPlacement, cardWidth, fromHeight, toHeight int, style lipgloss.Style) {
	fromY := from.y + fromHeight/2
	toY := to.y + toHeight/2
	startX := from.x + cardWidth
	endX := to.x - 1
	if endX < startX {
		endX = startX
	}
	midX := startX
	if endX > startX {
		midX = startX + (endX-startX)/2
	}

	if fromY == toY {
		drawConnHorizontal(canvas, fromY, startX, endX-1, style)
		canvas.set(endX, toY, arrowHead, style)
		return
	}

	if midX > startX {
		drawConnHorizontal(canvas, fromY, startX, midX-1, style)
	}

	if fromY < toY {
		canvas.set(midX, fromY, connDownLeft, style)
	} else {
		canvas.set(midX, fromY, connUpLeft, style)
	}

	minY, maxY := fromY, toY
	if maxY < minY {
		minY, maxY = maxY, minY
	}
	if maxY-minY > 1 {
		drawConnVertical(canvas, midX, minY+1, maxY-1, style)
	}

	if fromY < toY {
		canvas.set(midX, toY, connUpRight, style)
	} else {
		canvas.set(midX, toY, connDownRight, style)
	}

	if endX-1 > midX {
		drawConnHorizontal(canvas, toY, midX+1, endX-1, style)
	}
	canvas.set(endX, toY, arrowHead, style)
}

func drawConnHorizontal(canvas *dagCanvas, y, x1, x2 int, style lipgloss.Style) {
	if y < 0 || y >= canvas.height {
		return
	}
	if x2 < x1 {
		x1, x2 = x2, x1
	}
	for x := x1; x <= x2; x++ {
		drawConnRuneStyled(canvas, x, y, connHLine, style)
	}
}

func drawConnVertical(canvas *dagCanvas, x, y1, y2 int, style lipgloss.Style) {
	if x < 0 || x >= canvas.width {
		return
	}
	if y2 < y1 {
		y1, y2 = y2, y1
	}
	for y := y1; y <= y2; y++ {
		drawConnRuneStyled(canvas, x, y, connVLine, style)
	}
}

func drawConnRuneStyled(canvas *dagCanvas, x, y int, r rune, style lipgloss.Style) {
	if x < 0 || x >= canvas.width || y < 0 || y >= canvas.height {
		return
	}
	current := canvas.cells[y][x].ch
	if isBorderRune(current) {
		return
	}
	if current == 0 || current == ' ' || current == dotRune {
		canvas.set(x, y, r, style)
		return
	}
	if current == connCross {
		return
	}
	if (current == connHLine && r == connVLine) || (current == connVLine && r == connHLine) {
		canvas.set(x, y, connCross, style)
		return
	}
	if current == r {
		// Allow re-drawing same rune with new style (e.g., highlighted over dimmed)
		canvas.set(x, y, r, style)
		return
	}
	canvas.set(x, y, r, style)
}

func isBorderRune(r rune) bool {
	if r == boxHLine || r == boxVLine {
		return true
	}
	if r == boxTopLeft || r == boxTopRight || r == boxBottomLeft || r == boxBottomRight {
		return true
	}
	return false
}

func shortID(id string) string {
	runes := []rune(id)
	if len(runes) <= 8 {
		return id
	}
	return string(runes[:8]) + "..."
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func clampInt(value, lo, hi int) int {
	if value < lo {
		return lo
	}
	if value > hi {
		return hi
	}
	return value
}

func newDagCanvas(width, height int) *dagCanvas {
	cells := make([][]styledCell, height)
	for y := range cells {
		cells[y] = make([]styledCell, width)
	}
	return &dagCanvas{width: width, height: height, cells: cells}
}

func (c *dagCanvas) set(x, y int, ch rune, style lipgloss.Style) {
	if x < 0 || x >= c.width || y < 0 || y >= c.height {
		return
	}
	c.cells[y][x] = styledCell{ch: ch, style: style}
}

func (c *dagCanvas) drawBorder(x, y, width, height int, style lipgloss.Style) {
	for col := 0; col < width; col++ {
		c.set(x+col, y, boxHLine, style)
		c.set(x+col, y+height-1, boxHLine, style)
	}
	for row := 0; row < height; row++ {
		c.set(x, y+row, boxVLine, style)
		c.set(x+width-1, y+row, boxVLine, style)
	}
	c.set(x, y, boxTopLeft, style)
	c.set(x+width-1, y, boxTopRight, style)
	c.set(x, y+height-1, boxBottomLeft, style)
	c.set(x+width-1, y+height-1, boxBottomRight, style)
}

func (c *dagCanvas) fillDots() {
	dotStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("238"))
	for y := 0; y < c.height; y++ {
		for x := 0; x < c.width; x++ {
			if x%2 == 0 && y%2 == 0 {
				c.set(x, y, dotRune, dotStyle)
			}
		}
	}
}

func (c *dagCanvas) render() []string {
	lines := make([]string, c.height)
	for y := 0; y < c.height; y++ {
		var b strings.Builder
		for x := 0; x < c.width; x++ {
			cell := c.cells[y][x]
			ch := cell.ch
			if ch == 0 {
				ch = ' '
			}
			b.WriteString(cell.style.Render(string(ch)))
		}
		lines[y] = strings.TrimRight(b.String(), " ")
	}
	return lines
}
