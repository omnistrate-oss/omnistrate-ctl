package docs

import (
	"fmt"
	"strings"

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
	searchCmd.Flags().IntP("limit", "l", 100, "Maximum number of results to return")
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

	err = utils.PrintTextTableJsonArrayOutput(output, results)
	if err != nil {
		utils.PrintError(err)
		return err
	}

	return nil
}
