package dataaccess

import (
	"testing"
)

func TestCleanupId(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{input: "\"se-fhw4UJW8G3\"", expected: "se-fhw4UJW8G3"},
		{input: "\"se-fhw4UJW8G3\"\n", expected: "se-fhw4UJW8G3"},
		{input: "\t\"se-fhw4UJW8G3\"\t", expected: "se-fhw4UJW8G3"},
		{input: "\n\"se-fhw4UJW8G3\"\n", expected: "se-fhw4UJW8G3"},
		{input: "se-fhw4UJW8G3", expected: "se-fhw4UJW8G3"},
	}

	for _, test := range tests {
		result := cleanupId(test.input)
		if result != test.expected {
			t.Errorf("cleanupId(%q) = %q; expected %q", test.input, result, test.expected)
		}
	}
}
