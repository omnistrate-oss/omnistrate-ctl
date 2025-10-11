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
func TestParseH3Sections(t *testing.T) {
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
			name:     "no h3 headings",
			input:    "This is some content without h3 headings.\nMore content here.",
			expected: []MarkupSection{},
		},
		{
			name: "single h3 section",
			input: `### Configuration

This is the configuration section.
It has multiple lines of content.`,
			expected: []MarkupSection{
				{
					Header:  "Configuration",
					Content: "This is the configuration section.\nIt has multiple lines of content.",
				},
			},
		},
		{
			name: "multiple h3 sections",
			input: `### Getting Started

Welcome to the getting started guide.
This section covers the basics.

### Configuration

Here we explain configuration options.
You can configure various settings.

### Advanced Topics

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
			name: "h3 with special characters",
			input: `### API Reference - v2.1

This section documents the API.

### FAQ & Troubleshooting

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
			name: "h3 with mixed heading levels",
			input: `# Main Title

Some introduction text.

## Major Section

Content for major section.

### First Subsection

Content for first subsection.

#### Deep subsection

This is deeply nested.

### Second Subsection

Content for second subsection.`,
			expected: []MarkupSection{
				{
					Header:  "First Subsection",
					Content: "Content for first subsection.\n\n#### Deep subsection\n\nThis is deeply nested.",
				},
				{
					Header:  "Second Subsection",
					Content: "Content for second subsection.",
				},
			},
		},
		{
			name: "h3 bounded by h2 sections",
			input: `## Section One

Intro to section one.

### Subsection A

Content for subsection A.

### Subsection B

Content for subsection B.

## Section Two

Intro to section two.

### Subsection C

Content for subsection C.`,
			expected: []MarkupSection{
				{
					Header:  "Subsection A",
					Content: "Content for subsection A.",
				},
				{
					Header:  "Subsection B",
					Content: "Content for subsection B.",
				},
				{
					Header:  "Subsection C",
					Content: "Content for subsection C.",
				},
			},
		},
		{
			name: "h3 with empty sections",
			input: `### Empty Section

### Another Section

Content here.

### Final Empty Section`,
			expected: []MarkupSection{
				{
					Header:  "Empty Section",
					Content: "",
				},
				{
					Header:  "Another Section",
					Content: "Content here.",
				},
				{
					Header:  "Final Empty Section",
					Content: "",
				},
			},
		},
		{
			name: "h3 with code blocks and formatting",
			input: `### Installation

To install the package:

` + "```bash" + `
npm install package
` + "```" + `

### Usage

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
		{
			name: "h3 terminated by h2",
			input: `### First Subsection

Content for first subsection.
More content here.

## New Major Section

This ends the h3 section above.

### Another Subsection

Content after h2.`,
			expected: []MarkupSection{
				{
					Header:  "First Subsection",
					Content: "Content for first subsection.\nMore content here.",
				},
				{
					Header:  "Another Subsection",
					Content: "Content after h2.",
				},
			},
		},
		{
			name: "complex mixed headings",
			input: `# Document Title

## Overview

Some overview content.

### Overview Details

Details about the overview.

### Installation Steps

Step by step installation.

## Configuration

Main configuration section.

### Basic Config

Basic configuration options.

### Advanced Config

Advanced configuration options.

## Usage

Usage section.`,
			expected: []MarkupSection{
				{
					Header:  "Overview Details",
					Content: "Details about the overview.",
				},
				{
					Header:  "Installation Steps",
					Content: "Step by step installation.",
				},
				{
					Header:  "Basic Config",
					Content: "Basic configuration options.",
				},
				{
					Header:  "Advanced Config",
					Content: "Advanced configuration options.",
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			results, err := ParseH3Sections(test.input)

			assert.NoError(err, "ParseH3Sections should not return an error")
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
