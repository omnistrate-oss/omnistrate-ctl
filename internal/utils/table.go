package utils

import (
	"encoding/json"
	"fmt"
	prettytable "github.com/jedib0t/go-pretty/v6/table"
	"io"
	"os"
	"slices"
)

type Table struct {
	tableWriter prettytable.Writer
	columns     []string
}

func NewTable(columns []any) (t *Table) {
	t = &Table{
		tableWriter: prettytable.NewWriter(),
	}

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
		// Then sort alphabetically
		if a < b {
			return -1
		} else if a > b {
			return 1
		}
		return 0
	})

	// Convert back to any
	var columnsAsAny []any

	for _, column := range t.columns {
		columnsAsAny = append(columnsAsAny, column)
	}

	t.tableWriter.AppendHeader(columnsAsAny)

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
	t.tableWriter.AppendRow(row)
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

func (t Table) PrintToWriter(finalW io.Writer) {
	t.tableWriter.SetOutputMirror(finalW)
	t.tableWriter.Render()
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
