package secret

import (
	"fmt"
	"github.com/omnistrate-oss/omnistrate-ctl/cmd/common"
	"github.com/omnistrate-oss/omnistrate-ctl/cmd/environment"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/config"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/dataaccess"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/utils"
	"github.com/spf13/cobra"
)

const (
	// #nosec G101 -- This is just an example string for CLI help, not actual credentials
	secretDeleteExample = `# Delete a secret from dev environment
omnistrate-ctl environment secret delete dev my-secret

# Delete a secret from prod environment
omnistrate-ctl environment secret delete prod db-password`
)

var secretDeleteCmd = &cobra.Command{
	Use:          "delete [environment-type] [secret-name] [flags]",
	Short:        "Delete an environment secret",
	Long:         `This command helps you delete a secret from a specific environment type.`,
	Example:      secretDeleteExample,
	RunE:         runSecretDelete,
	SilenceUsage: true,
}

func init() {
	secretDeleteCmd.Args = cobra.ExactArgs(2)
}

func runSecretDelete(cmd *cobra.Command, args []string) error {
	defer config.CleanupArgsAndFlags(cmd, &args)

	environmentType := args[0]
	secretName := args[1]

	// Validate environment type
	if err := environment.ValidateEnvironmentType(environmentType); err != nil {
		utils.PrintError(err)
		return err
	}

	// Validate user login
	token, err := common.GetTokenWithLogin()
	if err != nil {
		utils.PrintError(err)
		return err
	}

	// Delete the secret
	err = dataaccess.DeleteSecret(cmd.Context(), token, environmentType, secretName)
	if err != nil {
		utils.PrintError(err)
		return err
	}

	fmt.Printf("Successfully deleted secret '%s' from environment type '%s'\n", secretName, environmentType)
	return nil
}
