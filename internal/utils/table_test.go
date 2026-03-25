package utils

import (
	"encoding/json"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestTablePrint(t *testing.T) {
	table := NewTable([]any{"Name", "Age", "Country"})
	table.AddRow([]any{"John Doe", "30", "USA"})
	table.AddRow([]any{"Jane Doe", "25", "Canada"})
	table.AddRow([]any{"John Smith", "40", "UK"})
	table.Print()
	// Output:
	// Name      Age  Country
	// -----------------------
	// John Doe  30   USA
	// Jane Doe  25   Canada
}

func TestTablePrintEmpty(t *testing.T) {
	table := NewTable([]any{"Name", "Age", "Country"})
	table.Print()
	// Output:
	// Name  Age  Country
	// ------------------
}

func TestTablePrintOneRow(t *testing.T) {
	table := NewTable([]any{"Name", "Age", "Country"})
	table.AddRow([]any{"John Doe", "30", "USA"})
	table.Print()
	// Output:
	// Name      Age  Country
	// -----------------------
	// John Doe  30   USA
}

func TestTableWithOneColumn(t *testing.T) {
	table := NewTable([]any{"Name"})
	table.AddRow([]any{"John Doe"})
	table.AddRow([]any{"Jane Doe"})
	table.Print()
	// Output:
	// Name
	// ----
	// John Doe
	// Jane Doe
}

func TestTableFromAJSON(t *testing.T) {
	dataItems := []json.RawMessage{
		[]byte(`{"Name": "John Doe", "Age": 30, "Country": "USA"}`),
		[]byte(`{"Name": "Jane Doe", "Age": 25, "Country": "Canada"}`),
		[]byte(`{"Name": "John Smith", "Age": 40, "Country": "UK"}`),
		[]byte(`{"Name": "Jane Smith", "Age": 35, "Country": "Australia"}`),
	}

	table, err := NewTableFromJSONTemplate(dataItems[0])
	require.NoError(t, err)

	for _, data := range dataItems {
		err := table.AddRowFromJSON(data)
		require.NoError(t, err)
	}

	table.Print()
	// Output:
	// Name        Age  Country
	// -------------------------
	// John Doe    30   USA
	// Jane Doe    25   Canada
	// John Smith  40   UK
	// Jane Smith  35   Australia
}
