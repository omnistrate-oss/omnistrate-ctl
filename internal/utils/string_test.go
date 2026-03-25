package utils

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/stretchr/testify/assert"
)

func TestCheckIfEmpty(t *testing.T) {
	t.Parallel()

	require := require.New(t)
	tests := []struct {
		name      string
		parameter string
		wantErr   bool
	}{
		{
			name:      "Empty string",
			parameter: "",
			wantErr:   true,
		},
		{
			name:      "Non-empty string",
			parameter: "hello",
			wantErr:   false,
		},
		{
			name:      "Whitespace string",
			parameter: " ",
			wantErr:   true,
		},
		{
			name:      "Whitespace string with leading and trailing spaces",
			parameter: "  ",
			wantErr:   true,
		},
		{
			name:      "Non-empty string with leading and trailing spaces",
			parameter: " hello ",
			wantErr:   false,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := CheckIfEmpty(tc.parameter)
			if tc.wantErr {
				require.Error(err)
			} else {
				require.NoError(err)
			}
		})
	}
}

func TestCheckIfNilOrEmpty(t *testing.T) {
	t.Parallel()

	require := require.New(t)
	tests := []struct {
		name      string
		parameter *string
		want      bool
	}{
		{
			name:      "Nil pointer",
			parameter: nil,
			want:      true,
		},
		{
			name:      "Empty string",
			parameter: ToPtr(""),
			want:      true,
		},
		{
			name:      "Non-empty string",
			parameter: ToPtr("hello"),
			want:      false,
		},
		{
			name:      "Whitespace string",
			parameter: ToPtr(" "),
			want:      true,
		},
		{
			name:      "Whitespace string with leading and trailing spaces",
			parameter: ToPtr("  "),
			want:      true,
		},
		{
			name:      "Non-empty string with leading and trailing spaces",
			parameter: ToPtr(" hello "),
			want:      false,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := CheckIfNilOrEmpty(tc.parameter)
			require.Equal(tc.want, got)
		})
	}
}

func TestTruncateString(t *testing.T) {
	t.Parallel()

	assert := assert.New(t)
	str1 := "A-tisket a-tasket A green and yellow basket"
	str2 := "Peter Piper picked a peck of pickled peppers"

	assert.Equal("A-tisket...", TruncateString(str1, 11))
	assert.Equal("Peter Piper...", TruncateString(str2, 14))
	assert.Equal("A-tisket a-tasket A green and yellow basket", TruncateString(str1, len(str1)))
	assert.Equal("A-tisket a-tasket A green and yellow basket", TruncateString(str1, len(str1)+2))
	assert.Equal("A...", TruncateString("A-", 1))
	assert.Equal("Ab...", TruncateString("Absolutely Longer", 2))
}

func TestTruncateStringAndMax(t *testing.T) {
	t.Parallel()

	require := require.New(t)
	tests := []struct {
		name     string
		input    string
		max      int
		expected string
	}{
		{
			name:     "Empty string",
			input:    "",
			max:      10,
			expected: "",
		},
		{
			name:     "Short string, no truncation needed",
			input:    "Short",
			max:      10,
			expected: "Short",
		},
		{
			name:     "Exact length, no truncation needed",
			input:    "Exact length",
			max:      12,
			expected: "Exact length",
		},
		{
			name:     "Truncation at word boundary",
			input:    "This is a long sentence that should be truncated.",
			max:      26,
			expected: "This is a long sentence...",
		},
		{
			name:     "Truncation of a long word without spaces",
			input:    "Thisisaverylongwordwithoutspaces",
			max:      10,
			expected: "Thisisa...",
		},
		{
			name:     "No truncation needed, sentence ends with punctuation",
			input:    "This sentence ends with punctuation!",
			max:      36,
			expected: "This sentence ends with punctuation!",
		},
		{
			name:     "Truncation with punctuation",
			input:    "Another sentence; with punctuation.",
			max:      25,
			expected: "Another sentence; with...",
		},
		{
			name:     "Truncation with trailing spaces and punctuation",
			input:    "Trailing spaces and punctuations; ",
			max:      30,
			expected: "Trailing spaces and punctua...",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			require.LessOrEqual(len(test.expected), test.max)
			got := TruncateString(test.input, test.max)
			require.LessOrEqual(len(got), test.max, "TruncateString(%q, %d) = %q; length %d; max %d", test.input, test.max, got, len(got), test.max)
			require.Equal(test.expected, got, "TruncateString(%q, %d) = %q; want %q", test.input, test.max, got, test.expected)
		})
	}
}

func TestCutString(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		str    string
		length int
		want   string
	}{
		// Basic cases
		{"empty string", "", 5, ""},
		{"truncate to zero", "hello", 0, ""},
		{"negative length", "hello", -1, ""},
		{"no truncation needed", "hello", 5, "hello"},
		{"truncate to less", "hello", 3, "hel"},
		{"longer than string", "hi", 10, "hi"},

		// Unicode and special characters
		{"unicode characters", "こんにちは", 3, "こんに"},
		{"mixed ascii and unicode", "hello😊world", 8, "hello😊wo"},

		// Edge cases
		{"exact length", "hey", 3, "hey"},
		{"single character", "a", 1, "a"},
		{"truncate to one", "world", 1, "w"},
		{"empty string, zero length", "", 0, ""},
		{"empty string, negative length", "", -1, ""},
		{"non-zero string, zero length", "hello", 0, ""},
		{"non-zero string, negative length", "hello", -1, ""},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := CutString(tc.str, tc.length); got != tc.want {
				t.Errorf("CutString(%q, %d) = %q; want %q", tc.str, tc.length, got, tc.want)
			}
		})
	}
}
