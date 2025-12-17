package secret

import (
	"github.com/omnistrate-oss/omnistrate-ctl/cmd/common"
	"github.com/omnistrate-oss/omnistrate-ctl/cmd/environment"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/config"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/dataaccess"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/model"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/utils"
	"github.com/spf13/cobra"
)

const (
	// #nosec G101 -- This is just an example string for CLI help, not actual credentials
	secretGetExample = `# Get a secret in dev environment
omnistrate-ctl secret get dev my-secret

# Get a secret with JSON output
omnistrate-ctl secret get prod db-password --output json`
)

var secretGetCmd = &cobra.Command{
	Use:          "get [environment-type] [secret-name] [flags]",
	Short:        "Get an environment secret",
	Long:         `This command helps you get a specific secret for an environment type.`,
	Example:      secretGetExample,
	RunE:         runSecretGet,
	SilenceUsage: true,
}

func init() {
	secretGetCmd.Args = cobra.ExactArgs(2)
}

func runSecretGet(cmd *cobra.Command, args []string) error {
	defer config.CleanupArgsAndFlags(cmd, &args)

	environmentType := args[0]
	secretName := args[1]
	output, _ := cmd.Flags().GetString("output")

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

	// Get secret
	result, err := dataaccess.GetSecret(cmd.Context(), token, environmentType, secretName)
	if err != nil {
		utils.PrintError(err)
		return err
	}

	// Format response
	secret := model.Secret{
		EnvironmentType: result.GetEnvironmentType(),
		Name:            result.GetName(),
	}

	err = utils.PrintTextTableJsonOutput(output, secret)
	if err != nil {
		utils.PrintError(err)
		return err
	}

	return nil
}
