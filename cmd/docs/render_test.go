package docs

import (
	"strings"
	"testing"

	"github.com/omnistrate-oss/omnistrate-ctl/internal/dataaccess"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The markdown table inside a doc section is the case the table renderer destroyed:
// it collapsed the rows onto one line and truncated the rest away.
const rootSchemaContent = `The root of the schema includes the following properties:

| Property     | Type    | Description           |
| ------------ | ------- | --------------------- |
| ` + "`name`" + `       | string  | The name of the Plan. |
| ` + "`services`" + `   | array   | **Required**.         |`

func TestRenderSpecSectionsMarkdownPreservesContentVerbatim(t *testing.T) {
	var out strings.Builder
	renderSpecSectionsMarkdown(&out, []specSection{{
		Tag:     "Root schema",
		URL:     "https://docs.omnistrate.com/spec-guides/plan-spec/#root-schema",
		Content: rootSchemaContent,
	}})

	rendered := out.String()
	assert.Contains(t, rendered, "## Root schema")
	assert.Contains(t, rendered, "https://docs.omnistrate.com/spec-guides/plan-spec/#root-schema")
	// Every content line survives, on its own line, untruncated.
	for _, line := range strings.Split(rootSchemaContent, "\n") {
		assert.Contains(t, rendered, line)
	}
	assert.NotContains(t, rendered, "…", "content must not be truncated")
	assert.Equal(t, strings.Count(rootSchemaContent, "\n")+1+4,
		strings.Count(strings.TrimRight(rendered, "\n"), "\n")+1,
		"heading + blank + url + blank + body lines")
}

func TestRenderSpecSectionsMarkdownSeparatesMultipleSections(t *testing.T) {
	var out strings.Builder
	renderSpecSectionsMarkdown(&out, []specSection{
		{Tag: "x-omnistrate-compute", URL: "https://example.com/#a", Content: "Compute config."},
		{Tag: "x-omnistrate-storage", URL: "https://example.com/#b", Content: "Storage config."},
	})

	rendered := out.String()
	assert.Contains(t, rendered, "## x-omnistrate-compute")
	assert.Contains(t, rendered, "## x-omnistrate-storage")
	assert.Contains(t, rendered, "\n---\n", "sections should be visually separated")
}

func TestRenderSpecSectionsMarkdownHandlesEmptyContent(t *testing.T) {
	var out strings.Builder
	renderSpecSectionsMarkdown(&out, []specSection{{Tag: "Empty", URL: "https://example.com/#e"}})

	rendered := out.String()
	assert.Contains(t, rendered, "## Empty")
	assert.NotContains(t, rendered, "<nil>")
}

func TestRenderAvailableTagsMarkdown(t *testing.T) {
	var out strings.Builder
	renderAvailableTagsMarkdown(&out, "Available plan spec sections",
		[]string{"Root schema", "Deployment schema"},
		`omnistrate-ctl docs plan-spec "%s"`)

	rendered := out.String()
	assert.Contains(t, rendered, "## Available plan spec sections (2)")
	assert.Contains(t, rendered, "- Root schema")
	assert.Contains(t, rendered, "- Deployment schema")
	// The listing has to teach the next step, since an unmatched tag lands here.
	assert.Contains(t, rendered, `omnistrate-ctl docs plan-spec "Root schema"`)
}

func TestRenderAvailableTagsMarkdownWithNoTags(t *testing.T) {
	var out strings.Builder
	renderAvailableTagsMarkdown(&out, "Available compose spec tags", nil,
		`omnistrate-ctl docs compose-spec "%s"`)

	rendered := out.String()
	assert.Contains(t, rendered, "(0)")
	assert.NotContains(t, rendered, "Read one with", "no example when there is nothing to read")
}

func TestRenderSearchResultsMarkdown(t *testing.T) {
	var out strings.Builder
	renderSearchResultsMarkdown(&out, []dataaccess.DocumentationResult{{
		Title:    "Helm Chart Customization",
		Subtitle: "How to customize Helm Chart Values",
		Section:  "Build Guides",
		URL:      "https://docs.omnistrate.com/build-guides/helm-charts-customize/#how-to-customize-helm-chart-values",
		Content:  "First line.\n\nSecond line.",
	}})

	rendered := out.String()
	assert.Contains(t, rendered, "## Helm Chart Customization — How to customize Helm Chart Values")
	assert.Contains(t, rendered, "*Build Guides*")
	assert.Contains(t, rendered, "First line.")
	assert.Contains(t, rendered, "Second line.")
}

func TestRenderSearchResultsMarkdownWithoutSubtitle(t *testing.T) {
	var out strings.Builder
	renderSearchResultsMarkdown(&out, []dataaccess.DocumentationResult{{
		Title:   "Installing ctl",
		URL:     "https://docs.omnistrate.com/getting-started/installing-ctl/",
		Content: "Install it.",
	}})

	rendered := out.String()
	assert.Contains(t, rendered, "## Installing ctl")
	assert.NotContains(t, rendered, "—", "no em dash when there is no subtitle")
}

func TestSchemaOutputResolvesMarkdownToJSON(t *testing.T) {
	// A schema has no markdown form, and markdown is not a format the shared print
	// helpers accept, so it has to resolve rather than error.
	assert.Equal(t, "json", schemaOutput(outputMarkdown))
	assert.Equal(t, "json", schemaOutput("json"))
	assert.Equal(t, "table", schemaOutput("table"))
	assert.Equal(t, "text", schemaOutput("text"))
}

func TestSpecSectionConverters(t *testing.T) {
	compose := composeSectionsToSpecSections([]dataaccess.ComposeSpecResult{
		{Tag: "x-omnistrate-compute", URL: "u1", Content: "c1"},
	})
	require.Len(t, compose, 1)
	assert.Equal(t, specSection{Tag: "x-omnistrate-compute", URL: "u1", Content: "c1"}, compose[0])

	plan := planSectionsToSpecSections([]dataaccess.PlanSpecResult{
		{Tag: "Root schema", URL: "u2", Content: "c2"},
	})
	require.Len(t, plan, 1)
	assert.Equal(t, specSection{Tag: "Root schema", URL: "u2", Content: "c2"}, plan[0])
}
