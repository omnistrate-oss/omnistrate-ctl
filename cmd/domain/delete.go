package domain

import (
	"slices"
	"strings"

	"github.com/omnistrate-oss/omnistrate-ctl/cmd/common"

	"github.com/omnistrate-oss/omnistrate-ctl/internal/config"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/dataaccess"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/utils"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

var (
	deleteExample = `# Delete domain with name
omnistrate-ctl delete domain [domain-name]`
)

var deleteCmd = &cobra.Command{
	Use:          "delete [name]",
	Short:        "Delete a Custom Domain",
	Long:         `This command helps you delete a Custom Domain.`,
	Example:      deleteExample,
	RunE:         runDelete,
	SilenceUsage: true,
}

func init() {
	deleteCmd.Args = cobra.ExactArgs(1) // Require exactly one argument
}

func runDelete(cmd *cobra.Command, args []string) error {
	defer config.CleanupArgsAndFlags(cmd, &args)

	// Retrieve flags
	output, err := cmd.Flags().GetString("output")
	if err != nil {
		utils.PrintError(err)
		return err
	}

	token, err := common.GetTokenWithLogin()
	if err != nil {
		utils.PrintError(err)
		return err
	}

	environmentTypes := make([]string, 0)

	// List domains
	listRes, err := dataaccess.ListDomains(cmd.Context(), token)
	if err != nil {
		utils.PrintError(err)
		return err
	}

	// Filter domains by name
	found := make(map[string]int)
	for _, name := range args {
		found[name] = 0
	}

	for _, d := range listRes.CustomDomains {
		if slices.Contains(args, d.Name) {
			environmentTypes = append(environmentTypes, d.EnvironmentType)
			found[d.Name] += 1
		}
	}

	domainsNotFound := make([]string, 0)
	for name, count := range found {
		if count == 0 {
			domainsNotFound = append(domainsNotFound, name)
		}
	}

	if len(domainsNotFound) > 0 {
		err = errors.New("domain not found: " + strings.Join(domainsNotFound, ", "))
		utils.PrintError(err)
		return err
	}

	for name, count := range found {
		if count > 1 {
			utils.PrintWarning("Multiple domains found with name: " + name + ". Deleting all of them.")
		}
	}

	// Delete domain
	for _, environmentType := range environmentTypes {
		err = dataaccess.DeleteDomain(cmd.Context(), token, environmentType)
		if err != nil {
			utils.PrintError(err)
			return err
		}
	}

	if output != "json" {
		utils.PrintSuccess("Domain deleted successfully")
	}

	return nil
}
