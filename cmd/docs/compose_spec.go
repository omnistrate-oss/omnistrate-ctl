package docs

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/omnistrate-oss/omnistrate-ctl/internal/dataaccess"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/utils"
	"github.com/spf13/cobra"
)

const (
	composeSpecExample = `# List all H3 headers in the compose spec documentation
omctl docs compose-spec

# Search for a specific tag/section in the compose spec documentation
omctl docs compose-spec "volumes"

# Search for a specific tag with JSON output
omctl docs compose-spec "networks" --output json`
)

var composeSpecCmd = &cobra.Command{
	Use:          "compose-spec [tag]",
	Short:        "Compose spec documentation",
	Long:         "This command returns information about the Omnistrate Docker Compose specification. If no tag is provided, it lists all supported tags. If a tag is provided, it returns the information about the tag.",
	Example:      composeSpecExample,
	RunE:         runComposeSpec,
	SilenceUsage: true,
}

func init() {
	composeSpecCmd.Flags().StringP("output", "o", "table", "Output format (table|json)")
}

// ComposeSpecResult represents a compose spec search result
type ComposeSpecResult struct {
	Header  string `json:"header"`
	Content string `json:"content,omitempty"`
	URL     string `json:"url"`
}

func runComposeSpec(cmd *cobra.Command, args []string) error {
	// Retrieve flags
	output, err := cmd.Flags().GetString("output")
	if err != nil {
		utils.PrintError(err)
		return err
	}

	// Get the tag from args (optional)
	var tag string
	if len(args) > 0 {
		tag = strings.Join(args, " ")
	}

	// Fetch content from the compose spec documentation
	const composeSpecURL = "https://docs.omnistrate.com/spec-guides/compose-spec/index.md"
	content, err := dataaccess.FetchContentFromURL(composeSpecURL)
	if err != nil {
		utils.PrintError(fmt.Errorf("failed to fetch compose spec documentation: %w", err))
		return err
	}

	// Parse H3 headers and their content
	h3Sections, err := parseH3Sections(content)
	if err != nil {
		utils.PrintError(fmt.Errorf("failed to parse H3 sections: %w", err))
		return err
	}

	if len(h3Sections) == 0 {
		fmt.Println("No H3 headers found in the compose spec documentation")
		return nil
	}

	var results []ComposeSpecResult

	if tag == "" {
		// No tag provided, return list of all H3 headers
		for _, section := range h3Sections {
			results = append(results, ComposeSpecResult{
				Header: section.Header,
				URL:    composeSpecURL + "#" + strings.ToLower(strings.ReplaceAll(section.Header, " ", "-")),
			})
		}
	} else {
		// Tag provided, search for matching sections
		found := false
		for _, section := range h3Sections {
			if strings.Contains(strings.ToLower(section.Header), strings.ToLower(tag)) {
				results = append(results, ComposeSpecResult{
					Header:  section.Header,
					Content: section.Content,
					URL:     composeSpecURL + "#" + strings.ToLower(strings.ReplaceAll(section.Header, " ", "-")),
				})
				found = true
			}
		}

		if !found {
			fmt.Printf("No sections found matching tag: %s\n\nAvailable H3 headers:\n", tag)
			for _, section := range h3Sections {
				fmt.Printf("- %s\n", section.Header)
			}
			return nil
		}
	}

	// Print results
	err = utils.PrintTextTableJsonArrayOutput(output, results)
	if err != nil {
		utils.PrintError(err)
		return err
	}

	return nil
}

// H3Section represents a section of content under an H3 heading
type H3Section struct {
	Header  string
	Content string
}

// parseH3Sections parses markdown content and extracts H3 sections
func parseH3Sections(content string) ([]H3Section, error) {
	var sections []H3Section

	// Use regex to find H3 headings (### )
	h3Regex := regexp.MustCompile(`(?m)^### (.+)$`)
	matches := h3Regex.FindAllStringSubmatchIndex(content, -1)

	if len(matches) == 0 {
		return sections, nil
	}

	// Process each H3 section
	for i, match := range matches {
		// Extract the H3 title (first capture group)
		titleStart := match[2]
		titleEnd := match[3]
		header := strings.TrimSpace(content[titleStart:titleEnd])

		// Determine the content boundaries
		contentStart := match[1] // End of the H3 line
		var contentEnd int

		if i+1 < len(matches) {
			// Content ends at the start of the next H3 heading
			contentEnd = matches[i+1][0]
		} else {
			// This is the last section, content goes to the end
			contentEnd = len(content)
		}

		// Extract and clean the section content
		sectionContent := strings.TrimSpace(content[contentStart:contentEnd])

		// Add the section (even if content is empty)
		sections = append(sections, H3Section{
			Header:  header,
			Content: sectionContent,
		})
	}

	return sections, nil
}
