package utils

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"slices"

	btable "github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/lipgloss"
)

// Table wraps a bubbles table for rendering structured data.
type Table struct {
	columns []string
	rows    [][]string
}

func NewTable(columns []any) (t *Table) {
	t = &Table{}

	columnsAsStrings := make([]string, 0, len(columns))
	for _, column := range columns {
		columnsAsStrings = append(columnsAsStrings, fmt.Sprintf("%v", column))
	}

	// Sort the columns with "name" first, then alphabetically
	t.columns = columnsAsStrings
	slices.SortFunc(t.columns, func(a, b string) int {
		// "name" always comes first
		if a == "name" {
			return -1
		}
		if b == "name" {
			return 1
		}
		if a < b {
			return -1
		} else if a > b {
			return 1
		}
		return 0
	})

	return
}

func NewTableFromJSONTemplate(data json.RawMessage) (t *Table, err error) {
	var mappedData map[string]any
	if err = json.Unmarshal(data, &mappedData); err != nil {
		return
	}

	columns := make([]any, 0, len(mappedData))
	for k := range mappedData {
		columns = append(columns, k)
	}

	t = NewTable(columns)
	return
}

func (t *Table) AddRow(row []any) {
	strRow := make([]string, len(row))
	for i, v := range row {
		if v == nil {
			strRow[i] = ""
		} else {
			strRow[i] = fmt.Sprintf("%v", v)
		}
	}
	t.rows = append(t.rows, strRow)
}

func (t *Table) AddRowFromJSON(data json.RawMessage) error {
	var row map[string]any
	err := json.Unmarshal(data, &row)
	if err != nil {
		return err
	}

	var rowSlice []any
	for _, column := range t.columns {
		rowSlice = append(rowSlice, row[fmt.Sprintf("%v", column)])
	}

	t.AddRow(rowSlice)
	return nil
}

func (t Table) Print() {
	t.PrintToWriter(os.Stdout)
}

func (t Table) PrintToWriter(w io.Writer) {
	if len(t.columns) == 0 {
		return
	}

	// Calculate optimal column widths from content
	widths := make([]int, len(t.columns))
	for i, col := range t.columns {
		widths[i] = lipgloss.Width(col)
	}
	for _, row := range t.rows {
		for i, cell := range row {
			if i < len(widths) {
				if cw := lipgloss.Width(cell); cw > widths[i] {
					widths[i] = cw
				}
			}
		}
	}

	// Build bubbles/table columns and rows
	cols := make([]btable.Column, len(t.columns))
	for i, name := range t.columns {
		cols[i] = btable.Column{Title: name, Width: widths[i]}
	}

	rows := make([]btable.Row, len(t.rows))
	for i, row := range t.rows {
		rows[i] = btable.Row(row)
	}

	bt := btable.New(
		btable.WithColumns(cols),
		btable.WithRows(rows),
		btable.WithHeight(max(len(rows)+1, 2)),
	)

	s := btable.DefaultStyles()
	s.Header = s.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("240")).
		BorderBottom(true).
		Bold(true).
		Foreground(lipgloss.Color("99"))
	s.Selected = lipgloss.NewStyle()
	bt.SetStyles(s)

	baseStyle := lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("240"))

	fmt.Fprintln(w, baseStyle.Render(bt.View()))
}

func PrintTable(jsonData []string) (err error) {
	if len(jsonData) == 0 {
		return
	}

	var tableWriter *Table
	if tableWriter, err = NewTableFromJSONTemplate(json.RawMessage(jsonData[0])); err != nil {
		// Just print the JSON directly and return
		fmt.Printf("%+v\n", jsonData)
		return err
	}

	for _, data := range jsonData {
		if err = tableWriter.AddRowFromJSON(json.RawMessage(data)); err != nil {
			// Just print the JSON directly and return
			fmt.Printf("%+v\n", jsonData)
			return err
		}
	}

	tableWriter.Print()

	return
}
