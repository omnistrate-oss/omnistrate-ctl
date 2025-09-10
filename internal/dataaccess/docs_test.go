package dataaccess

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseDocumentationContent(t *testing.T) {
	assert := assert.New(t)

	tests := []struct {
		name     string
		input    string
		expected []DocumentationResult
	}{
		{
			name: "parse basic section with links",
			input: `## Introduction

- [Introduction](https://docs.omnistrate.com/index.md): Main introduction to Omnistrate
- [What is Omnistrate](https://docs.omnistrate.com/what-is-omnistrate/index.md): Overview of what Omnistrate is
- [What you can do](https://docs.omnistrate.com/what-you-can-do/index.md): What you can accomplish with Omnistrate
- [Cost vs. Benefits](https://docs.omnistrate.com/cost-benefits/index.md): Cost vs benefits analysis
- [Architecture](https://docs.omnistrate.com/architecture/index.md): Platform architecture overview`,
			expected: []DocumentationResult{
				{
					Title:       "Introduction",
					URL:         "https://docs.omnistrate.com/",
					Description: "Main introduction to Omnistrate",
					Section:     "Introduction",
				},
				{
					Title:       "What is Omnistrate",
					URL:         "https://docs.omnistrate.com/what-is-omnistrate/",
					Description: "Overview of what Omnistrate is",
					Section:     "Introduction",
				},
				{
					Title:       "What you can do",
					URL:         "https://docs.omnistrate.com/what-you-can-do/",
					Description: "What you can accomplish with Omnistrate",
					Section:     "Introduction",
				},
				{
					Title:       "Cost vs. Benefits",
					URL:         "https://docs.omnistrate.com/cost-benefits/",
					Description: "Cost vs benefits analysis",
					Section:     "Introduction",
				},
				{
					Title:       "Architecture",
					URL:         "https://docs.omnistrate.com/architecture/",
					Description: "Platform architecture overview",
					Section:     "Introduction",
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			results, err := parseDocumentationContent(test.input)

			if err != nil {
				assert.NoError(err, "parseDocumentationContent() should not return an error")
				return
			}

			if len(results) != len(test.expected) {
				assert.Equal(len(test.expected), len(results), "parseDocumentationContent() returned %d results, expected %d", len(results), len(test.expected))
				return
			}

			for i, result := range results {
				expected := test.expected[i]
				if result.Title != expected.Title {
					assert.Equal(expected.Title, result.Title, "Result %d: Title mismatch", i)
				}
				if result.URL != expected.URL {
					assert.Equal(expected.URL, result.URL, "Result %d: URL mismatch", i)
				}
				if result.Description != expected.Description {
					assert.Equal(expected.Description, result.Description, "Result %d: Description mismatch", i)
				}
				if result.Section != expected.Section {
					assert.Equal(expected.Section, result.Section, "Result %d: Section mismatch", i)
				}
				assert.NotEmpty(result.Content, "Result %d: Content should not be empty", i)
			}
		})
	}
}
