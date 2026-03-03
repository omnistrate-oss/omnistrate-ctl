package instance

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// outputNode represents a node in the JSON tree view
type outputNode struct {
	key            string
	value          string // leaf value as string (or masked if sensitive)
	realValue      string // actual value when sensitive
	nodeType       string // "object", "array", "string", "number", "bool", "null"
	depth          int
	expandable     bool
	expanded       bool
	sensitive      bool // whether this is a sensitive terraform output
	sensitiveShown bool // whether the real value is currently revealed
	children       []*outputNode
}

// buildOutputTreeFromJSON parses the terraform output JSON and builds a navigable tree.
// The JSON format is: { "output_name": { "sensitive": bool, "type": ..., "value": ... }, ... }
func buildOutputTreeFromJSON(rawJSON string) []outputNode {
	if rawJSON == "" {
		return nil
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(rawJSON), &parsed); err != nil {
		// Not valid JSON — show as raw text
		return []outputNode{{
			key:      "output",
			value:    truncateValue(rawJSON, 500),
			nodeType: "string",
			depth:    0,
		}}
	}

	var roots []outputNode

	keys := make([]string, 0, len(parsed))
	for k := range parsed {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, key := range keys {
		val := parsed[key]
		// Each output is typically { "sensitive": bool, "type": ..., "value": ... }
		if obj, ok := val.(map[string]interface{}); ok {
			node := buildTerraformOutputNode(key, obj, 0)
			roots = append(roots, *node)
		} else {
			node := buildJSONNode(key, val, 0)
			roots = append(roots, *node)
		}
	}

	return roots
}

// buildTerraformOutputNode builds a node for a terraform output entry,
// showing the value prominently with type and sensitivity info
func buildTerraformOutputNode(key string, obj map[string]interface{}, depth int) *outputNode {
	sensitive, _ := obj["sensitive"].(bool)
	value := obj["value"]

	node := buildJSONNode(key, value, depth)

	if sensitive {
		node.sensitive = true
		node.realValue = node.value
		node.value = "••••••••  (sensitive, press enter to reveal)"
	}

	return node
}

// findLatestOutputLog finds the latest *-output.log content from the Files map,
// matching against the history to pick the most recent operation's output
func findLatestOutputLog(files map[string]string, history []TerraformHistoryEntry) string {
	if len(files) == 0 {
		return ""
	}

	// Try to find by latest history entry that has an output log
	for i := len(history) - 1; i >= 0; i-- {
		opID := history[i].OperationID
		if opID == "" {
			continue
		}
		key := opID + "-output.log"
		if content, ok := files[key]; ok {
			return content
		}
	}

	// Fallback: find any *-output.log key, pick the last one alphabetically
	var outputKeys []string
	for k := range files {
		if strings.HasSuffix(k, "-output.log") {
			outputKeys = append(outputKeys, k)
		}
	}
	if len(outputKeys) == 0 {
		return ""
	}
	sort.Strings(outputKeys)
	return files[outputKeys[len(outputKeys)-1]]
}

func buildJSONNode(key string, value interface{}, depth int) *outputNode {
	switch v := value.(type) {
	case map[string]interface{}:
		node := &outputNode{
			key:        key,
			nodeType:   "object",
			depth:      depth,
			expandable: true,
			expanded:   depth < 1, // auto-expand first level
		}
		// Sort keys
		keys := make([]string, 0, len(v))
		for k := range v {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			child := buildJSONNode(k, v[k], depth+1)
			node.children = append(node.children, child)
		}
		return node

	case []interface{}:
		node := &outputNode{
			key:        key,
			nodeType:   "array",
			depth:      depth,
			expandable: len(v) > 0,
			expanded:   depth < 1,
		}
		for i, item := range v {
			child := buildJSONNode(fmt.Sprintf("[%d]", i), item, depth+1)
			node.children = append(node.children, child)
		}
		return node

	case string:
		return &outputNode{
			key:      key,
			value:    v,
			nodeType: "string",
			depth:    depth,
		}

	case float64:
		s := fmt.Sprintf("%g", v)
		return &outputNode{
			key:      key,
			value:    s,
			nodeType: "number",
			depth:    depth,
		}

	case bool:
		return &outputNode{
			key:      key,
			value:    fmt.Sprintf("%t", v),
			nodeType: "bool",
			depth:    depth,
		}

	case nil:
		return &outputNode{
			key:      key,
			value:    "null",
			nodeType: "null",
			depth:    depth,
		}

	default:
		return &outputNode{
			key:      key,
			value:    fmt.Sprintf("%v", v),
			nodeType: "string",
			depth:    depth,
		}
	}
}

// flattenOutputTree returns a flat list of visible nodes for rendering
func flattenOutputTree(roots []outputNode) []*outputNode {
	var flat []*outputNode
	for i := range roots {
		flattenOutputNode(&roots[i], &flat)
	}
	return flat
}

func flattenOutputNode(node *outputNode, flat *[]*outputNode) {
	*flat = append(*flat, node)
	if node.expandable && node.expanded {
		for _, child := range node.children {
			flattenOutputNode(child, flat)
		}
	}
}

// renderTerraformOutputTab renders the terraform output logs as a JSON tree view
func (m terraformDetailModel) renderTerraformOutputTab() string {
	if m.loading {
		return fmt.Sprintf("\n  %s Fetching terraform output...", m.spinner.View())
	}
	if m.loadErr != nil {
		errStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("203"))
		return fmt.Sprintf("\n  %s\n", errStyle.Render(fmt.Sprintf("Error: %v", m.loadErr)))
	}

	if len(m.outputTree) == 0 {
		subtleStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
		return fmt.Sprintf("\n  %s\n", subtleStyle.Render("No terraform output data available for this resource."))
	}

	var b strings.Builder

	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("255"))
	b.WriteString(fmt.Sprintf("  %s\n\n", headerStyle.Render("Terraform Output")))

	visibleNodes := flattenOutputTree(m.outputTree)

	// Viewport clipping
	visibleRows := m.bodyHeight() - 4
	if visibleRows < 1 {
		visibleRows = 1
	}

	totalEntries := len(visibleNodes)

	// Auto-scroll to keep cursor visible
	scrollOffset := 0
	if m.outputCursor >= visibleRows {
		scrollOffset = m.outputCursor - visibleRows + 1
	}
	if scrollOffset > totalEntries-visibleRows {
		scrollOffset = totalEntries - visibleRows
	}
	if scrollOffset < 0 {
		scrollOffset = 0
	}

	end := scrollOffset + visibleRows
	if end > totalEntries {
		end = totalEntries
	}

	keyStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("117"))   // blue
	strStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("114"))   // green
	numStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("178"))   // yellow
	boolStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("178"))  // yellow
	nullStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241"))  // dim
	braceStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("245")) // dim
	selectedBg := lipgloss.NewStyle().Background(lipgloss.Color("236")) // subtle highlight

	maxValWidth := m.contentWidth() - 20
	if maxValWidth < 20 {
		maxValWidth = 20
	}

	for idx := scrollOffset; idx < end; idx++ {
		node := visibleNodes[idx]
		indent := strings.Repeat("  ", node.depth)

		cursor := "  "
		if idx == m.outputCursor {
			cursor = "▶ "
		}

		var line string
		if node.expandable {
			arrow := "▸"
			if node.expanded {
				arrow = "▾"
			}
			childCount := len(node.children)
			typeMark := braceStyle.Render("{}")
			if node.nodeType == "array" {
				typeMark = braceStyle.Render("[]")
			}
			countStr := lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Render(fmt.Sprintf("  %d items", childCount))
			line = fmt.Sprintf("%s%s %s %s%s", indent, arrow, keyStyle.Render(node.key), typeMark, countStr)
		} else {
			var styledVal string
			switch node.nodeType {
			case "string":
				val := node.value
				runes := []rune(val)
				if len(runes) > maxValWidth {
					val = string(runes[:maxValWidth-1]) + "…"
				}
				styledVal = strStyle.Render(fmt.Sprintf("%q", val))
			case "number":
				styledVal = numStyle.Render(node.value)
			case "bool":
				styledVal = boolStyle.Render(node.value)
			case "null":
				styledVal = nullStyle.Render("null")
			default:
				styledVal = lipgloss.NewStyle().Foreground(lipgloss.Color("252")).Render(node.value)
			}
			line = fmt.Sprintf("%s  %s: %s", indent, keyStyle.Render(node.key), styledVal)
		}

		if idx == m.outputCursor {
			line = selectedBg.Render(line)
		}

		b.WriteString(fmt.Sprintf("  %s%s\n", cursor, line))
	}

	// Scroll indicator
	if totalEntries > visibleRows {
		dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
		pos := ""
		if scrollOffset == 0 {
			pos = "top"
		} else if end >= totalEntries {
			pos = "end"
		} else {
			pct := (scrollOffset * 100) / (totalEntries - visibleRows)
			pos = fmt.Sprintf("%d%%", pct)
		}
		b.WriteString(fmt.Sprintf("\n  %s\n", dimStyle.Render(fmt.Sprintf("↑↓: navigate  enter: expand/collapse  [%d/%d %s]", m.outputCursor+1, totalEntries, pos))))
	} else {
		b.WriteString(fmt.Sprintf("\n  %s\n", lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Render("↑↓: navigate  enter: expand/collapse")))
	}

	return b.String()
}

func truncateValue(s string, maxLen int) string {
	runes := []rune(s)
	if len(runes) > maxLen {
		return string(runes[:maxLen-1]) + "…"
	}
	return s
}
