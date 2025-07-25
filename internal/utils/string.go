package utils

import (
	"encoding/json"
	"io/ioutil"
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

// ReadFile reads the contents of a file
func ReadFile(filePath string) ([]byte, error) {
	data, err := ioutil.ReadFile(filePath)
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
