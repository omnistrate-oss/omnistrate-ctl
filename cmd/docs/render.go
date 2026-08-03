package docs

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/omnistrate-oss/omnistrate-ctl/internal/dataaccess"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/utils"
)

// The docs commands return documentation prose: paragraphs, markdown tables and
// fenced code blocks. A terminal table cannot represent that — the renderer emits
// one line per row and truncates each cell to the column width, so a table view of
// a doc section silently drops nearly all of it. These commands therefore default to
// markdown, which preserves the section body verbatim, and keep table/text/json
// available for callers that ask for them explicitly.
const (
	outputMarkdown = "markdown"
	outputJSON     = "json"

	docsOutputUsage   = "Output format (markdown|json|table|text). markdown preserves the full section text; table truncates it to one line per row"
	schemaOutputUsage = "Output format (json|markdown|table|text). json is the default because a JSON schema cannot be represented as a table"
)

// specSection is the shape shared by compose spec and plan spec search results.
type specSection struct {
	Tag     string
	URL     string
	Content string
}

func composeSectionsToSpecSections(results []dataaccess.ComposeSpecResult) []specSection {
	sections := make([]specSection, 0, len(results))
	for _, r := range results {
		sections = append(sections, specSection{Tag: r.Tag, URL: r.URL, Content: r.Content})
	}
	return sections
}

func planSectionsToSpecSections(results []dataaccess.PlanSpecResult) []specSection {
	sections := make([]specSection, 0, len(results))
	for _, r := range results {
		sections = append(sections, specSection{Tag: r.Tag, URL: r.URL, Content: r.Content})
	}
	return sections
}

// renderSpecSectionsMarkdown writes each matched section as markdown, keeping the
// body exactly as the documentation wrote it.
func renderSpecSectionsMarkdown(w io.Writer, sections []specSection) {
	var b strings.Builder
	for i, section := range sections {
		if i > 0 {
			b.WriteString("\n---\n\n")
		}
		fmt.Fprintf(&b, "## %s\n\n", section.Tag)
		if section.URL != "" {
			fmt.Fprintf(&b, "%s\n\n", section.URL)
		}
		if section.Content != "" {
			b.WriteString(strings.TrimRight(section.Content, "\n"))
			b.WriteString("\n")
		}
	}
	writeRendered(w, b.String())
}

// warnNoTagMatch tells the caller on stderr that their tag matched nothing and the
// listing below is a fallback. It goes to stderr so it is visible in every output
// mode — including `-o json` piped into jq — without corrupting stdout. Without it
// the fallback listing is indistinguishable from a real answer once piped.
func warnNoTagMatch(specName, tag string, available int) {
	if tag == "" {
		return
	}
	fmt.Fprintf(os.Stderr, "No %s section matched %q; listing the %d available sections instead.\n",
		specName, tag, available)
}

// renderAvailableTagsMarkdown writes the tag listing as a bullet list, with a note
// telling the caller how to read one of them.
func renderAvailableTagsMarkdown(w io.Writer, heading string, tags []string, readCmd string) {
	var b strings.Builder
	fmt.Fprintf(&b, "## %s (%d)\n\n", heading, len(tags))
	for _, tag := range tags {
		fmt.Fprintf(&b, "- %s\n", tag)
	}
	if readCmd != "" && len(tags) > 0 {
		fmt.Fprintf(&b, "\nRead one with: %s\n", fmt.Sprintf(readCmd, tags[0]))
	}
	writeRendered(w, b.String())
}

// renderSearchResultsMarkdown writes search hits as markdown, preserving each
// excerpt's own formatting.
func renderSearchResultsMarkdown(w io.Writer, results []dataaccess.DocumentationResult) {
	var b strings.Builder
	for i, result := range results {
		if i > 0 {
			b.WriteString("\n---\n\n")
		}
		heading := result.Title
		if result.Subtitle != "" {
			heading = fmt.Sprintf("%s — %s", result.Title, result.Subtitle)
		}
		fmt.Fprintf(&b, "## %s\n\n", heading)
		if result.Section != "" {
			fmt.Fprintf(&b, "*%s*\n\n", result.Section)
		}
		if result.URL != "" {
			fmt.Fprintf(&b, "%s\n\n", result.URL)
		}
		if result.Content != "" {
			b.WriteString(strings.TrimRight(result.Content, "\n"))
			b.WriteString("\n")
		}
	}
	writeRendered(w, b.String())
}

// renderValidationResultMarkdown writes the validation outcome as a readable report:
// one line per violation, each naming the spec path so it can be found and fixed.
func renderValidationResultMarkdown(w io.Writer, result *dataaccess.SpecValidationResult) {
	var b strings.Builder

	if result.Valid {
		fmt.Fprintf(&b, "%s is valid against the %s schema.\n", result.File, result.SpecType)
		writeRendered(w, b.String())
		return
	}

	fmt.Fprintf(&b, "%s has %d violation(s) of the %s schema:\n\n",
		result.File, len(result.Violations), result.SpecType)
	for _, v := range result.Violations {
		fmt.Fprintf(&b, "  %s\n      %s\n", v.Path, v.Message)
	}
	b.WriteString("\nPaths are JSON pointers into the spec — `/services/0/apiParameters/2/type`\n" +
		"means the `type` of the third API parameter of the first service.\n" +
		"\nNote: a violation here is a structural problem. Invalid *values* (a wrong\n" +
		"tenancyType, cloudProvider, or parameter type) are not detectable from the\n" +
		"schema — check those against `docs compose-spec` / `docs plan-spec`.\n")

	writeRendered(w, b.String())
}

// writeRendered emits rendered markdown and mirrors it into LastPrintedString so it
// stays consistent with the shared print helpers.
func writeRendered(w io.Writer, rendered string) {
	fmt.Fprint(w, rendered)
	utils.LastPrintedString = rendered
}

func stdout() io.Writer { return os.Stdout }

// schemaOutput maps the requested format onto one the shared print helpers accept
// for a JSON schema payload. markdown has no meaning for a schema, so it resolves to
// json rather than falling through to the unsupported-format error.
func schemaOutput(output string) string {
	if output == outputMarkdown {
		return outputJSON
	}
	return output
}
