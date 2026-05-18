package customer

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/omnistrate-oss/omnistrate-ctl/cmd/common"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/config"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/dataaccess"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/utils"
	"github.com/spf13/cobra"
)

const createExample = `# Create a customer portal user
omnistrate-ctl customer create --email user@example.com --name "Jane Doe" --password "$PASSWORD" --legal-company-name "Example Inc"

# Create a customer portal user and auto-verify the account
omnistrate-ctl customer create --email user@example.com --name "Jane Doe" --password "$PASSWORD" --legal-company-name "Example Inc" --auto-verify

# Create a customer portal user with the password read from stdin
echo "$PASSWORD" | omnistrate-ctl customer create --email user@example.com --name "Jane Doe" --password-stdin --legal-company-name "Example Inc"`

var createCmd = &cobra.Command{
	Use:          "create [flags]",
	Short:        "Create a customer portal user",
	Long:         "This command creates a customer portal user.",
	Example:      createExample,
	RunE:         runCreate,
	SilenceUsage: true,
}

func init() {
	createCmd.Flags().String("email", "", "Customer user email")
	createCmd.Flags().String("name", "", "Customer user name")
	createCmd.Flags().String("password", "", "Customer user password")
	createCmd.Flags().Bool("password-stdin", false, "Reads the customer user password from stdin")
	createCmd.Flags().String("legal-company-name", "", "Customer legal company name")
	createCmd.Flags().String("company-url", "", "Customer company URL")
	createCmd.Flags().Bool("auto-verify", false, "Enable automatic verification for the customer user")
	createCmd.Flags().StringArray("attribute", []string{}, "Customer user attribute in key=value format. Can be repeated or comma-separated")
	_ = createCmd.MarkFlagRequired("email")
	_ = createCmd.MarkFlagRequired("name")
	_ = createCmd.MarkFlagRequired("legal-company-name")
	createCmd.MarkFlagsMutuallyExclusive("password", "password-stdin")
	createCmd.MarkFlagsOneRequired("password", "password-stdin")
}

func runCreate(cmd *cobra.Command, args []string) error {
	defer config.CleanupArgsAndFlags(cmd, &args)

	email, _ := cmd.Flags().GetString("email")
	name, _ := cmd.Flags().GetString("name")
	password, err := customerCreatePassword(cmd, os.Stdin, customerInputIsTerminal(os.Stdin))
	if err != nil {
		utils.PrintError(err)
		return err
	}
	legalCompanyName, _ := cmd.Flags().GetString("legal-company-name")
	companyURL, _ := cmd.Flags().GetString("company-url")
	autoVerify, _ := cmd.Flags().GetBool("auto-verify")
	attributeValues, _ := cmd.Flags().GetStringArray("attribute")

	attributes, err := parseAttributes(attributeValues)
	if err != nil {
		utils.PrintError(err)
		return err
	}
	if err = utils.ValidateEmail(strings.TrimSpace(email)); err != nil {
		err = fmt.Errorf("invalid --email value: %w", err)
		utils.PrintError(err)
		return err
	}

	token, err := common.GetTokenWithLogin()
	if err != nil {
		utils.PrintError(err)
		return err
	}

	userID, err := dataaccess.CreateCustomerUser(
		cmd.Context(),
		token,
		dataaccess.NewCustomerUserCreateRequest(
			strings.TrimSpace(email),
			strings.TrimSpace(name),
			password,
			strings.TrimSpace(legalCompanyName),
			strings.TrimSpace(companyURL),
			autoVerify,
			attributes,
		),
	)
	if err != nil {
		utils.PrintError(err)
		return err
	}

	fmt.Printf("Successfully created customer user %s\n", userID)
	return nil
}

func customerCreatePassword(cmd *cobra.Command, stdin io.Reader, stdinIsTerminal bool) (string, error) {
	password, _ := cmd.Flags().GetString("password")
	passwordStdin, _ := cmd.Flags().GetBool("password-stdin")
	if strings.TrimSpace(password) != "" && passwordStdin {
		return "", fmt.Errorf("--password and --password-stdin are mutually exclusive")
	}

	if passwordStdin {
		if stdinIsTerminal {
			return "", fmt.Errorf("--password-stdin requires piped or redirected input; for example: printf '%%s\\n' \"$PASSWORD\" | omnistrate-ctl customer create --email user@example.com --name \"Jane Doe\" --legal-company-name \"Example Inc\" --password-stdin")
		}

		passwordFromStdin, err := io.ReadAll(stdin)
		if err != nil {
			return "", err
		}
		password = string(passwordFromStdin)
	}

	password = strings.TrimSpace(password)
	if password == "" {
		return "", fmt.Errorf("must provide a non-empty password via --password or --password-stdin")
	}

	return password, nil
}

func customerInputIsTerminal(file *os.File) bool {
	info, err := file.Stat()
	if err != nil {
		return false
	}
	return (info.Mode() & os.ModeCharDevice) != 0
}
