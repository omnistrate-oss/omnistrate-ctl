package instance

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/omnistrate-oss/omnistrate-ctl/cmd/common"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/dataaccess"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/utils"
	"github.com/spf13/cobra"

	"github.com/chelnak/ysmrr"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/config"
)

const (
	evaluateExample = `# Evaluate an expression for an instance
omnistrate-ctl instance evaluate instance-abcd1234 my-resource-key --expression "$var.username + {{ $sys.id }}"

# Evaluate expressions from a JSON file
omnistrate-ctl instance evaluate instance-abcd1234 my-resource-key --expression-file expressions.json`
)

var evaluateCmd = &cobra.Command{
	Use:          "evaluate [instance-id] [resource-key]",
	Short:        "Evaluate an expression in the context of an instance",
	Long:         `This command helps you evaluate expressions using instance parameters and system variables.`,
	Example:      evaluateExample,
	RunE:         runEvaluate,
	SilenceUsage: true,
}

func init() {
	evaluateCmd.Args = cobra.ExactArgs(2) // Require exactly two arguments
	evaluateCmd.Flags().StringP("expression", "e", "", "Expression string to evaluate")
	evaluateCmd.Flags().StringP("expression-file", "f", "", "Path to JSON file containing expressions mapped to expressionMap field")
	evaluateCmd.MarkFlagsMutuallyExclusive("expression", "expression-file")
}

func runEvaluate(cmd *cobra.Command, args []string) error {
	defer config.CleanupArgsAndFlags(cmd, &args)

	// Retrieve args
	instanceID := args[0]
	resourceKey := args[1]

	// Retrieve flags
	output, err := cmd.Flags().GetString("output")
	if err != nil {
		utils.PrintError(err)
		return err
	}

	expression, err := cmd.Flags().GetString("expression")
	if err != nil {
		utils.PrintError(err)
		return err
	}

	expressionFile, err := cmd.Flags().GetString("expression-file")
	if err != nil {
		utils.PrintError(err)
		return err
	}

	// Validate that either expression or expression-file is provided
	if expression == "" && expressionFile == "" {
		err = errors.New("either --expression or --expression-file must be provided")
		utils.PrintError(err)
		return err
	}

	// Validate user login
	token, err := common.GetTokenWithLogin()
	if err != nil {
		utils.PrintError(err)
		return err
	}

	// Initialize spinner if output is not JSON
	var sm ysmrr.SpinnerManager
	var spinner *ysmrr.Spinner
	if output != common.OutputTypeJson {
		sm = ysmrr.NewSpinnerManager()
		msg := "Evaluating expression..."
		spinner = sm.AddSpinner(msg)
		sm.Start()
	}

	// Get instance details to extract service ID and product tier ID
	serviceID, _, productTierID, _, err := getInstance(cmd.Context(), token, instanceID)
	if err != nil {
		utils.HandleSpinnerError(spinner, sm, err)
		return err
	}

	// Prepare the request
	var expressionMap map[string]interface{}
	if expressionFile != "" {
		// Load expressions from file
		expressionMap, err = loadExpressionsFromFile(expressionFile)
		if err != nil {
			utils.HandleSpinnerError(spinner, sm, err)
			return err
		}
	}

	// Call the evaluate API
	result, err := dataaccess.EvaluateExpression(cmd.Context(), token, serviceID, productTierID, instanceID, resourceKey, expression, expressionMap)
	if err != nil {
		utils.HandleSpinnerError(spinner, sm, err)
		return err
	}

	utils.HandleSpinnerSuccess(spinner, sm, "Successfully evaluated expression")

	// Create a result structure for consistent JSON output
	evaluationResult := struct {
		Result interface{} `json:"result"`
	}{
		Result: result,
	}

	// Print output using standard utility function
	err = utils.PrintTextTableJsonOutput(output, evaluationResult)
	if err != nil {
		utils.PrintError(err)
		return err
	}

	return nil
}

func loadExpressionsFromFile(filePath string) (map[string]interface{}, error) {
	// Check if file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return nil, fmt.Errorf("expression file not found: %s", filePath)
	}

	// Open and read the file
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open expression file: %w", err)
	}
	defer file.Close()

	content, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("failed to read expression file: %w", err)
	}

	// Parse JSON content
	var expressionMap map[string]interface{}
	err = json.Unmarshal(content, &expressionMap)
	if err != nil {
		return nil, fmt.Errorf("failed to parse JSON expression file: %w", err)
	}

	return expressionMap, nil
}
