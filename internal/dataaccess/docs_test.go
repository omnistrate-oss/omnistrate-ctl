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
			results, err := parseDocumentationContentForIndexing(test.input)

			if err != nil {
				assert.NoError(err, "parseDocumentationContentForIndexing() should not return an error")
				return
			}

			if len(results) != len(test.expected) {
				assert.Equal(len(test.expected), len(results), "parseDocumentationContentForIndexing() returned %d results, expected %d", len(results), len(test.expected))
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

func TestPerformDocumentationSearchWithBleve(t *testing.T) {
	// Clean up any existing index before test
	defer func() {
		_ = cleanupSearchIndex()
	}()

	// Test basic search functionality
	results, err := PerformDocumentationSearch("API", 3)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// We expect some results (assuming the documentation contains "API")
	if len(results) == 0 {
		t.Log("No results found for 'API' - this might be expected if the documentation source is not available")
		return
	}

	// Verify result structure
	for i, result := range results {
		if result.Title == "" {
			t.Errorf("Result %d has empty title", i)
		}
		if result.URL == "" {
			t.Errorf("Result %d has empty URL", i)
		}
		if result.Score <= 0 {
			t.Errorf("Result %d has invalid score: %f", i, result.Score)
		}
	}

	t.Logf("Found %d results for 'API' search", len(results))
}

func TestCleanupSearchIndex(t *testing.T) {
	// Initialize index first
	_, err := PerformDocumentationSearch("test", 1)
	if err != nil {
		t.Logf("Could not initialize index for cleanup test: %v", err)
		return
	}

	// Test cleanup
	err = cleanupSearchIndex()
	if err != nil {
		t.Errorf("Expected no error for cleanup, got: %v", err)
	}

	// Test cleanup again to ensure idempotency
	err = cleanupSearchIndex()
	if err != nil {
		t.Errorf("Expected no error for cleanup, got: %v", err)
	}
}

func TestRefreshSearchIndex(t *testing.T) {
	// Clean up any existing index before test
	defer func() {
		_ = cleanupSearchIndex()
	}()

	// Test refresh functionality
	err := refreshSearchIndex()
	if err != nil {
		t.Logf("RefreshSearchIndex failed (might be expected if documentation source is not available): %v", err)
		return
	}

	// Verify we can search after refresh
	results, err := PerformDocumentationSearch("test", 1)
	if err != nil {
		t.Errorf("Expected no error after refresh, got: %v", err)
	}

	t.Logf("RefreshSearchIndex completed successfully, found %d results for 'test'", len(results))
}
