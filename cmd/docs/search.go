package docs

import (
	"fmt"
	"strings"

	"github.com/omnistrate-oss/omnistrate-ctl/cmd/common"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/dataaccess"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/utils"
	"github.com/spf13/cobra"
)

const (
	searchExample = `# Search documentation for a specific term
omctl docs search "kubernetes"

# Search documentation with multiple terms
omctl docs search "service plan deployment"`
)

var searchCmd = &cobra.Command{
	Use:          "search [query]",
	Short:        "Search through Omnistrate documentation",
	Long:         "This command helps you search through Omnistrate documentation content for specific terms or topics.",
	Example:      searchExample,
	RunE:         runSearch,
	SilenceUsage: true,
}

func init() {
	searchCmd.Args = cobra.MinimumNArgs(1) // Require at least one argument (the search query)
	searchCmd.Flags().StringP("output", "o", "table", "Output format (table|json)")
	searchCmd.Flags().IntP("limit", "l", 10, "Maximum number of results to return")
}

func runSearch(cmd *cobra.Command, args []string) error {
	// Retrieve args
	query := strings.Join(args, " ")

	// Retrieve flags
	output, err := cmd.Flags().GetString("output")
	if err != nil {
		utils.PrintError(err)
		return err
	}

	limit, err := cmd.Flags().GetInt("limit")
	if err != nil {
		utils.PrintError(err)
		return err
	}

	// Validate user login
	_, err = common.GetTokenWithLogin()
	if err != nil {
		utils.PrintError(err)
		return err
	}

	// Perform the search
	results, err := dataaccess.PerformDocumentationSearch(query, limit)
	if err != nil {
		utils.PrintError(err)
		return err
	}

	// Print results
	if len(results) == 0 {
		fmt.Printf("No documentation found for query: %s\n", query)
		return nil
	}

	err = printSearchResults(results, output)
	if err != nil {
		utils.PrintError(err)
		return err
	}

	return nil
}

// printSearchResults prints the search results in the specified format
func printSearchResults(results []dataaccess.DocumentationResult, output string) error {
	switch output {
	case "json":
		return utils.PrintTextTableJsonOutput("json", results)
	case "table":
		fmt.Printf("Found %d documentation result(s):\n\n", len(results))
		for i, result := range results {
			fmt.Printf("%d. %s\n", i+1, result.Title)
			fmt.Printf("   Section: %s\n", result.Section)
			fmt.Printf("   URL: %s\n", result.URL)
			fmt.Printf("   Description: %s\n", result.Description)
			if i < len(results)-1 {
				fmt.Println()
			}
		}
		return nil
	default:
		return fmt.Errorf("unsupported output format: %s", output)
	}
}
