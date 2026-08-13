package utils

import (
	"encoding/json"
	"os"
	"strings"
	"unicode/utf8"

	"github.com/pkg/errors"
)

func CheckIfEmpty(parameter string) error {
	if strings.TrimSpace(parameter) == "" {
		return errors.New("parameter is empty")
	}
	return nil
}

func CheckIfNilOrEmpty(parameter *string) bool {
	if parameter == nil {
		return true
	}
	if strings.TrimSpace(*parameter) == "" {
		return true
	}
	return false
}

func TruncateString(str string, num int) string {
	bnoden := str
	if len(str) > num {
		if num > 3 {
			num -= 3
		}
		bnoden = str[0:num] + "..."
	}
	return bnoden
}

func CutString(str string, length int) string {
	if length <= 0 {
		return ""
	}

	if utf8.RuneCountInString(str) < length {
		return str
	}

	return string([]rune(str)[:length])
}

// ParseCommaSeparatedList parses a comma-separated string into a slice of strings
func ParseCommaSeparatedList(input string) []string {
	if strings.TrimSpace(input) == "" {
		return []string{}
	}

	parts := strings.Split(input, ",")
	result := make([]string, 0, len(parts))

	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}

	return result
}

// SplitEscapedCommaSeparatedList splits input on commas unless the comma is
// escaped as \,. A backslash only escapes comma and backslash; before any other
// character it is preserved.
func SplitEscapedCommaSeparatedList(input string) []string {
	result := make([]string, 0)
	var current strings.Builder
	escaping := false

	for _, r := range input {
		if escaping {
			if r != ',' && r != '\\' {
				current.WriteRune('\\')
			}
			current.WriteRune(r)
			escaping = false
			continue
		}

		switch r {
		case '\\':
			escaping = true
		case ',':
			result = append(result, current.String())
			current.Reset()
		default:
			current.WriteRune(r)
		}
	}

	if escaping {
		current.WriteRune('\\')
	}
	result = append(result, current.String())

	return result
}

// ReadFile reads the contents of a file
func ReadFile(filePath string) ([]byte, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to read file %s", filePath)
	}
	return data, nil
}

// FormatJSON formats an interface{} as pretty-printed JSON
func FormatJSON(data interface{}) (string, error) {
	jsonBytes, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return "", errors.Wrap(err, "failed to format JSON")
	}
	return string(jsonBytes), nil
}
