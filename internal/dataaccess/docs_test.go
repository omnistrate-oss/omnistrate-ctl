package dataaccess

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

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
func TestParseH2Sections(t *testing.T) {
	assert := assert.New(t)

	tests := []struct {
		name     string
		input    string
		expected []MarkupSection
	}{
		{
			name:     "empty content",
			input:    "",
			expected: []MarkupSection{},
		},
		{
			name:     "no h2 headings",
			input:    "This is some content without h2 headings.\nMore content here.",
			expected: []MarkupSection{},
		},
		{
			name: "single h2 section",
			input: `## Introduction

This is the introduction section.
It has multiple lines of content.`,
			expected: []MarkupSection{
				{
					Header:  "Introduction",
					Content: "This is the introduction section.\nIt has multiple lines of content.",
				},
			},
		},
		{
			name: "multiple h2 sections",
			input: `## Getting Started

Welcome to the getting started guide.
This section covers the basics.

## Configuration

Here we explain configuration options.
You can configure various settings.

## Advanced Topics

This section covers advanced features.`,
			expected: []MarkupSection{
				{
					Header:  "Getting Started",
					Content: "Welcome to the getting started guide.\nThis section covers the basics.",
				},
				{
					Header:  "Configuration",
					Content: "Here we explain configuration options.\nYou can configure various settings.",
				},
				{
					Header:  "Advanced Topics",
					Content: "This section covers advanced features.",
				},
			},
		},
		{
			name: "h2 with special characters",
			input: `## API Reference - v2.1

This section documents the API.

## FAQ & Troubleshooting

Common questions and answers.`,
			expected: []MarkupSection{
				{
					Header:  "API Reference - v2.1",
					Content: "This section documents the API.",
				},
				{
					Header:  "FAQ & Troubleshooting",
					Content: "Common questions and answers.",
				},
			},
		},
		{
			name: "h2 with mixed heading levels",
			input: `# Main Title

Some introduction text.

## First Section

Content for first section.

### Subsection

This is a subsection.

## Second Section

Content for second section.

#### Another subsection

More nested content.`,
			expected: []MarkupSection{
				{
					Header:  "First Section",
					Content: "Content for first section.\n\n### Subsection\n\nThis is a subsection.",
				},
				{
					Header:  "Second Section",
					Content: "Content for second section.\n\n#### Another subsection\n\nMore nested content.",
				},
			},
		},
		{
			name: "h2 with empty sections",
			input: `## Empty Section

## Another Section

Content here.

## Final Empty Section`,
			expected: []MarkupSection{
				{
					Header:  "Another Section",
					Content: "Content here.",
				},
			},
		},
		{
			name: "h2 with code blocks and formatting",
			input: `## Installation

To install the package:

` + "```bash" + `
npm install package
` + "```" + `

## Usage

Here's how to use it:

` + "```javascript" + `
const pkg = require('package');
pkg.run();
` + "```" + `

More usage examples.`,
			expected: []MarkupSection{
				{
					Header:  "Installation",
					Content: "To install the package:\n\n```bash\nnpm install package\n```",
				},
				{
					Header:  "Usage",
					Content: "Here's how to use it:\n\n```javascript\nconst pkg = require('package');\npkg.run();\n```\n\nMore usage examples.",
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			results := parseH2Sections(test.input)

			assert.Equal(len(test.expected), len(results), "Number of sections should match")

			for i, result := range results {
				if i < len(test.expected) {
					expected := test.expected[i]
					assert.Equal(expected.Header, result.Header, "Section %d: Header should match", i)
					assert.Equal(expected.Content, result.Content, "Section %d: Content should match", i)
				}
			}
		})
	}
}
