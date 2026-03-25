package utils

import (
	"bytes"
	"github.com/stretchr/testify/require"
	"io"
	"os"
	"testing"
)

func TestPrintText(t *testing.T) {
	require := require.New(t)

	tests := []struct {
		name      string
		inputData []string
		expected  string
		expectErr bool
	}{
		{
			name: "Basic Test",
			inputData: []string{
				`{"id":"1", "name":"Alice", "age":30}`,
				`{"id":"2", "name":"Bob", "age":25}`,
			},
			expected: "age id name\n30  1  Alice\n25  2  Bob\n",
		},
		{
			name: "Different Keys Order",
			inputData: []string{
				`{"name":"Alice", "age":30, "id":"1"}`,
				`{"id":"2", "age":25, "name":"Bob"}`,
			},
			expected: "age id name\n30  1  Alice\n25  2  Bob\n",
		},
		{
			name: "Missing Key in Second Row",
			inputData: []string{
				`{"id":"1", "name":"Alice", "age":30}`,
				`{"id":"2", "name":"Bob"}`, // Missing "age"
			},
			expected: "age id name\n30  1  Alice\n2  Bob\n",
		},
		{
			name: "Empty Data",
			inputData: []string{
				`{"id":"1", "name":"Alice", "age":30}`,
				`{}`, // Empty row
			},
			expected: "age id name\n30  1  Alice\n\n",
		},
		{
			name: "Invalid JSON",
			inputData: []string{
				`{"id":"1", "name":"Alice", "age":30}`,
				`{"id":2, "name":"Bob", "age":25`, // Invalid JSON
			},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Redirect stdout to a pipe
			old := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			// Run the test
			err := PrintText(tt.inputData)
			if tt.expectErr {
				require.Error(err)
			} else {
				require.NoError(err)
			}

			// Capture the output
			err = w.Close()
			if err != nil {
				return
			}
			os.Stdout = old
			var buf bytes.Buffer
			_, err = io.Copy(&buf, r)
			if err != nil {
				return
			}

			// Compare the output
			if !tt.expectErr {
				require.Equal(tt.expected, buf.String())
			}
		})
	}
}
