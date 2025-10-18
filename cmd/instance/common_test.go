package instance

import (
	"testing"

	openapiclientfleet "github.com/omnistrate-oss/omnistrate-sdk-go/fleet"
	"github.com/spf13/cobra"
)

func TestParseCustomTags(t *testing.T) {
	tests := []struct {
		name          string
		tagsFlag      string
		flagSet       bool
		expectedTags  []openapiclientfleet.CustomTag
		expectedSet   bool
		expectError   bool
		errorContains string
	}{
		{
			name:         "flag not set",
			tagsFlag:     "",
			flagSet:      false,
			expectedTags: nil,
			expectedSet:  false,
			expectError:  false,
		},
		{
			name:         "empty string when flag is set",
			tagsFlag:     "",
			flagSet:      true,
			expectedTags: []openapiclientfleet.CustomTag{},
			expectedSet:  true,
			expectError:  false,
		},
		{
			name:         "whitespace only when flag is set",
			tagsFlag:     "   ",
			flagSet:      true,
			expectedTags: []openapiclientfleet.CustomTag{},
			expectedSet:  true,
			expectError:  false,
		},
		{
			name:     "single tag",
			tagsFlag: "env=prod",
			flagSet:  true,
			expectedTags: []openapiclientfleet.CustomTag{
				{Key: "env", Value: "prod"},
			},
			expectedSet: true,
			expectError: false,
		},
		{
			name:     "multiple tags",
			tagsFlag: "env=prod,team=backend",
			flagSet:  true,
			expectedTags: []openapiclientfleet.CustomTag{
				{Key: "env", Value: "prod"},
				{Key: "team", Value: "backend"},
			},
			expectedSet: true,
			expectError: false,
		},
		{
			name:     "tags with spaces are trimmed",
			tagsFlag: " env = prod , team = backend ",
			flagSet:  true,
			expectedTags: []openapiclientfleet.CustomTag{
				{Key: "env", Value: "prod"},
				{Key: "team", Value: "backend"},
			},
			expectedSet: true,
			expectError: false,
		},
		{
			name:     "tags are sorted by key",
			tagsFlag: "z=last,a=first,m=middle",
			flagSet:  true,
			expectedTags: []openapiclientfleet.CustomTag{
				{Key: "a", Value: "first"},
				{Key: "m", Value: "middle"},
				{Key: "z", Value: "last"},
			},
			expectedSet: true,
			expectError: false,
		},
		{
			name:          "invalid format - no equals",
			tagsFlag:      "envprod",
			flagSet:       true,
			expectedTags:  nil,
			expectedSet:   false,
			expectError:   true,
			errorContains: "invalid tag",
		},
		{
			name:          "invalid format - empty tag pair",
			tagsFlag:      "env=prod,,team=backend",
			flagSet:       true,
			expectedTags:  nil,
			expectedSet:   false,
			expectError:   true,
			errorContains: "tag pair cannot be empty",
		},
		{
			name:          "invalid format - empty key",
			tagsFlag:      "=value",
			flagSet:       true,
			expectedTags:  nil,
			expectedSet:   false,
			expectError:   true,
			errorContains: "tag key cannot be empty",
		},
		{
			name:          "duplicate tag keys",
			tagsFlag:      "env=prod,env=dev",
			flagSet:       true,
			expectedTags:  nil,
			expectedSet:   false,
			expectError:   true,
			errorContains: "duplicate tag key",
		},
		{
			name:     "tag with empty value is allowed",
			tagsFlag: "env=",
			flagSet:  true,
			expectedTags: []openapiclientfleet.CustomTag{
				{Key: "env", Value: ""},
			},
			expectedSet: true,
			expectError: false,
		},
		{
			name:     "tag value can contain equals sign",
			tagsFlag: "formula=a=b+c",
			flagSet:  true,
			expectedTags: []openapiclientfleet.CustomTag{
				{Key: "formula", Value: "a=b+c"},
			},
			expectedSet: true,
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := &cobra.Command{}
			cmd.Flags().String("tags", "", "tags flag")

			if tt.flagSet {
				err := cmd.Flags().Set("tags", tt.tagsFlag)
				if err != nil {
					t.Fatalf("failed to set flag: %v", err)
				}
			}

			result, set, err := parseCustomTags(cmd)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
					return
				}
				if tt.errorContains != "" && !contains(err.Error(), tt.errorContains) {
					t.Errorf("expected error to contain %q, got %q", tt.errorContains, err.Error())
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if set != tt.expectedSet {
				t.Errorf("expected set=%v, got set=%v", tt.expectedSet, set)
			}

			if len(result) != len(tt.expectedTags) {
				t.Errorf("expected %d tags, got %d", len(tt.expectedTags), len(result))
				return
			}

			for i, expected := range tt.expectedTags {
				if result[i].Key != expected.Key || result[i].Value != expected.Value {
					t.Errorf("tag %d: expected {%s=%s}, got {%s=%s}",
						i, expected.Key, expected.Value, result[i].Key, result[i].Value)
				}
			}
		})
	}
}

func TestEnsureUniqueTagKeys(t *testing.T) {
	tests := []struct {
		name         string
		tags         []openapiclientfleet.CustomTag
		expectError  bool
		duplicateKey string
	}{
		{
			name:        "empty tags",
			tags:        []openapiclientfleet.CustomTag{},
			expectError: false,
		},
		{
			name: "single tag",
			tags: []openapiclientfleet.CustomTag{
				{Key: "env", Value: "prod"},
			},
			expectError: false,
		},
		{
			name: "multiple unique tags",
			tags: []openapiclientfleet.CustomTag{
				{Key: "env", Value: "prod"},
				{Key: "team", Value: "backend"},
				{Key: "region", Value: "us-east"},
			},
			expectError: false,
		},
		{
			name: "duplicate keys",
			tags: []openapiclientfleet.CustomTag{
				{Key: "env", Value: "prod"},
				{Key: "env", Value: "dev"},
			},
			expectError:  true,
			duplicateKey: "env",
		},
		{
			name: "duplicate keys with different values",
			tags: []openapiclientfleet.CustomTag{
				{Key: "env", Value: "prod"},
				{Key: "team", Value: "backend"},
				{Key: "env", Value: "staging"},
			},
			expectError:  true,
			duplicateKey: "env",
		},
		{
			name: "three identical keys",
			tags: []openapiclientfleet.CustomTag{
				{Key: "env", Value: "prod"},
				{Key: "env", Value: "dev"},
				{Key: "env", Value: "staging"},
			},
			expectError:  true,
			duplicateKey: "env",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ensureUniqueTagKeys(tt.tags)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
					return
				}
				if tt.duplicateKey != "" && !contains(err.Error(), tt.duplicateKey) {
					t.Errorf("expected error to mention duplicate key %q, got %q", tt.duplicateKey, err.Error())
				}
			} else if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestFormatTags(t *testing.T) {
	tests := []struct {
		name     string
		tags     []openapiclientfleet.CustomTag
		expected string
	}{
		{
			name:     "empty tags",
			tags:     []openapiclientfleet.CustomTag{},
			expected: "",
		},
		{
			name: "single tag",
			tags: []openapiclientfleet.CustomTag{
				{Key: "env", Value: "prod"},
			},
			expected: "env=prod",
		},
		{
			name: "multiple tags",
			tags: []openapiclientfleet.CustomTag{
				{Key: "env", Value: "prod"},
				{Key: "team", Value: "backend"},
			},
			expected: "env=prod,team=backend",
		},
		{
			name: "tag with empty value",
			tags: []openapiclientfleet.CustomTag{
				{Key: "env", Value: ""},
			},
			expected: "env=",
		},
		{
			name: "tag value with special characters",
			tags: []openapiclientfleet.CustomTag{
				{Key: "env", Value: "prod-us-east-1"},
				{Key: "owner", Value: "team@example.com"},
			},
			expected: "env=prod-us-east-1,owner=team@example.com",
		},
		{
			name: "tag value with equals sign",
			tags: []openapiclientfleet.CustomTag{
				{Key: "formula", Value: "a=b+c"},
			},
			expected: "formula=a=b+c",
		},
		{
			name: "three tags",
			tags: []openapiclientfleet.CustomTag{
				{Key: "env", Value: "prod"},
				{Key: "team", Value: "backend"},
				{Key: "region", Value: "us-east"},
			},
			expected: "env=prod,team=backend,region=us-east",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatTags(tt.tags)
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && indexOf(s, substr) >= 0))
}

func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

func TestParseTagFilters(t *testing.T) {
	tests := []struct {
		name        string
		input       []string
		expected    map[string]string
		expectError bool
	}{
		{
			name:        "empty filters",
			input:       []string{},
			expected:    map[string]string{},
			expectError: false,
		},
		{
			name:        "single tag filter",
			input:       []string{"env=prod"},
			expected:    map[string]string{"env": "prod"},
			expectError: false,
		},
		{
			name:        "multiple tag filters",
			input:       []string{"env=prod", "team=backend"},
			expected:    map[string]string{"env": "prod", "team": "backend"},
			expectError: false,
		},
		{
			name:        "filter with empty array marker",
			input:       []string{"[]", "env=prod"},
			expected:    map[string]string{"env": "prod"},
			expectError: false,
		},
		{
			name:        "invalid format - no equals",
			input:       []string{"envprod"},
			expected:    nil,
			expectError: true,
		},
		{
			name:        "multiple equals",
			input:       []string{"formular=a=b+c"},
			expected:    map[string]string{"formular": "a=b+c"},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseTagFilters(tt.input)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if len(result) != len(tt.expected) {
				t.Errorf("expected %d tags, got %d", len(tt.expected), len(result))
				return
			}

			for key, expectedValue := range tt.expected {
				if actualValue, exists := result[key]; !exists {
					t.Errorf("expected key %s not found in result", key)
				} else if actualValue != expectedValue {
					t.Errorf("for key %s, expected value %s, got %s", key, expectedValue, actualValue)
				}
			}
		})
	}
}

func TestMatchesTagFilters(t *testing.T) {
	tests := []struct {
		name          string
		instanceTags  string
		filterTags    map[string]string
		expectedMatch bool
	}{
		{
			name:          "no filters - should match",
			instanceTags:  "env=prod,team=backend",
			filterTags:    map[string]string{},
			expectedMatch: true,
		},
		{
			name:          "single matching tag",
			instanceTags:  "env=prod,team=backend",
			filterTags:    map[string]string{"env": "prod"},
			expectedMatch: true,
		},
		{
			name:          "multiple matching tags",
			instanceTags:  "env=prod,team=backend,region=us-east",
			filterTags:    map[string]string{"env": "prod", "team": "backend"},
			expectedMatch: true,
		},
		{
			name:          "single non-matching tag",
			instanceTags:  "env=prod,team=backend",
			filterTags:    map[string]string{"env": "dev"},
			expectedMatch: false,
		},
		{
			name:          "one matching, one non-matching tag",
			instanceTags:  "env=prod,team=backend",
			filterTags:    map[string]string{"env": "prod", "team": "frontend"},
			expectedMatch: false,
		},
		{
			name:          "filter tag not present in instance",
			instanceTags:  "env=prod,team=backend",
			filterTags:    map[string]string{"region": "us-east"},
			expectedMatch: false,
		},
		{
			name:          "empty instance tags with filters",
			instanceTags:  "",
			filterTags:    map[string]string{"env": "prod"},
			expectedMatch: false,
		},
		{
			name:          "instance with tags, empty filters",
			instanceTags:  "env=prod,team=backend",
			filterTags:    map[string]string{},
			expectedMatch: true,
		},
		{
			name:          "exact match all tags",
			instanceTags:  "env=prod",
			filterTags:    map[string]string{"env": "prod"},
			expectedMatch: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := matchesTagFilters(tt.instanceTags, tt.filterTags)
			if result != tt.expectedMatch {
				t.Errorf("expected match=%v, got match=%v", tt.expectedMatch, result)
			}
		})
	}
}
