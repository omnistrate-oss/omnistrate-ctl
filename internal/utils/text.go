package utils

import (
	"encoding/json"
	"fmt"
	"os"
	"slices"
	"strings"
	"text/tabwriter"
)

func PrintText(data []string) (err error) {
	if len(data) == 0 {
		return
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 1, ' ', tabwriter.TabIndent)

	// Unmarshal the first JSON data to get the columns
	var mappedData map[string]any
	if err = json.Unmarshal(json.RawMessage(data[0]), &mappedData); err != nil {
		return
	}

	columns := make([]any, 0, len(mappedData))
	for k := range mappedData {
		columns = append(columns, k)
	}

	// Sort the columns
	columnsAsStrings := make([]string, 0, len(columns))
	for _, column := range columns {
		columnsAsStrings = append(columnsAsStrings, fmt.Sprintf("%v", column))
	}
	slices.Sort(columnsAsStrings)

	// Print the header
	_, err = fmt.Fprintln(w, strings.Join(columnsAsStrings, "\t"))
	if err != nil {
		return
	}

	// Print the data
	for _, d := range data {
		mappedData = make(map[string]any)
		if err = json.Unmarshal(json.RawMessage(d), &mappedData); err != nil {
			return
		}

		var rowSlice []any
		for _, column := range columnsAsStrings {
			if mappedData[column] == nil {
				rowSlice = append(rowSlice, "")
				continue
			}
			rowSlice = append(rowSlice, mappedData[column])
		}

		_, err = fmt.Fprintf(w, strings.Repeat("%v\t", len(rowSlice)-1)+"%v\n", rowSlice...)
		if err != nil {
			return
		}
	}

	err = w.Flush()
	if err != nil {
		return
	}

	return
}
