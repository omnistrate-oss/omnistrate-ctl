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
	secretSetExample = `# Set a secret for dev environment
omnistrate-ctl secret set dev my-secret my-value

# Set a secret for prod environment
omnistrate-ctl secret set prod db-password secret123`
)

var secretSetCmd = &cobra.Command{
	Use:          "set [environment-type] [secret-name] [secret-value] [flags]",
	Short:        "Set an environment secret",
	Long:         `This command helps you create or update a secret for a specific environment type.`,
	Example:      secretSetExample,
	RunE:         runSecretSet,
	SilenceUsage: true,
}

func init() {
	secretSetCmd.Args = cobra.ExactArgs(3)
}

func runSecretSet(cmd *cobra.Command, args []string) error {
	defer config.CleanupArgsAndFlags(cmd, &args)

	environmentType := args[0]
	secretName := args[1]
	secretValue := args[2]

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

	// Set the secret
	err = dataaccess.SetSecret(cmd.Context(), token, environmentType, secretName, secretValue)
	if err != nil {
		utils.PrintError(err)
		return err
	}

	fmt.Printf("Successfully set secret '%s' for environment type '%s'\n", secretName, environmentType)
	return nil
}
