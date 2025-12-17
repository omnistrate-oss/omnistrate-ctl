package instance

import (
	"encoding/json"
	"errors"
	"strings"

	"github.com/cqroot/prompt"
	"github.com/cqroot/prompt/choose"

	"github.com/chelnak/ysmrr"
	"github.com/omnistrate-oss/omnistrate-ctl/cmd/common"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/config"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/dataaccess"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/utils"
	errors2 "github.com/pkg/errors"
	"github.com/spf13/cobra"
)

const (
	disableDebugModeExample = `# Disable debug mode for an instance deployment
omnistrate-ctl instance disable-debug-mode i-1234 --resource-name terraform --force`
)

var disableDebugModeCmd = &cobra.Command{
	Use:          "disable-debug-mode [instance-id] --resource-name [resource-name] --force",
	Short:        "Disable debug mode for an instance deployment",
	Long:         `This command helps you disable debug mode for an instance deployment`,
	Example:      disableDebugModeExample,
	RunE:         runDisableDebug,
	SilenceUsage: true,
}

func init() {
	disableDebugModeCmd.Flags().StringP("resource-name", "r", "", "Resource name")
	disableDebugModeCmd.Flags().BoolP("force", "f", false, "Force enable debug mode")

	disableDebugModeCmd.Args = cobra.ExactArgs(1) // Require exactly one argument
	disableDebugModeCmd.Flags().StringP("output", "o", "json", "Output format. Only json is supported")

	var err error
	if err = disableDebugModeCmd.MarkFlagRequired("resource-name"); err != nil {
		return
	}
}

func runDisableDebug(cmd *cobra.Command, args []string) error {
	defer config.CleanupArgsAndFlags(cmd, &args)

	if len(args) == 0 {
		err := errors.New("instance id is required")
		utils.PrintError(err)
		return err
	}

	// Retrieve args
	instanceID := args[0]

	// Retrieve flags
	output, err := cmd.Flags().GetString("output")
	if err != nil {
		utils.PrintError(err)
		return err
	}

	// Validate output flag
	if output != "json" {
		err = errors.New("only json output is supported")
		utils.PrintError(err)
		return err
	}

	isForce, err := cmd.Flags().GetBool("force")
	if err != nil {
		utils.PrintError(err)
		return err
	}

	if !isForce {
		// Prompt user to confirm
		var choice string
		choice, err = prompt.New().Ask("Please verify that your instance has been upgraded to the plan version with the appropriate fix. Continue to proceed?").
			Choose([]string{
				"Yes",
				"No",
			}, choose.WithTheme(choose.ThemeArrow))
		if err != nil {
			utils.PrintError(err)
			return err
		}

		switch choice {
		case "Yes":
			break
		case "No":
			utils.PrintInfo("Operation cancelled")
			return nil
		}
	}

	// Retrieve flags
	resourceName, err := cmd.Flags().GetString("resource-name")
	if err != nil {
		utils.PrintError(err)
		return err
	}

	if resourceName == "" {
		err = errors.New("resource name is required")
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
	if output != "json" {
		sm = ysmrr.NewSpinnerManager()
		msg := "Disabling debug mode for instance deployment..."
		spinner = sm.AddSpinner(msg)
		sm.Start()
	}

	// Check if instance exists
	serviceID, environmentID, _, _, err := getInstance(cmd.Context(), token, instanceID)
	if err != nil {
		utils.HandleSpinnerError(spinner, sm, err)
		return err
	}

	resourceID, resourceType, err := getResourceFromInstance(cmd.Context(), token, instanceID, resourceName)
	if err != nil {
		utils.HandleSpinnerError(spinner, sm, err)
		return err
	}

	// Validate deployment type
	if strings.ToLower(resourceType) != string(TerraformDeploymentType) {
		err = errors.New("only terraform deployment type is supported")
		utils.PrintError(err)
		return err
	}

	var deploymentName string
	switch strings.ToLower(resourceType) {
	case string(TerraformDeploymentType):
		deploymentName = getTerraformDeploymentName(resourceID, instanceID)
	}

	_, err = dataaccess.GetInstanceDeploymentEntity(cmd.Context(), token, instanceID, resourceType, deploymentName)
	if err != nil {
		utils.HandleSpinnerError(spinner, sm, err)
		return err
	}

	// Enable debug mode
	err = dataaccess.UpdateResourceInstanceDebugMode(cmd.Context(), token, serviceID, environmentID, instanceID, false)
	if err != nil {
		utils.HandleSpinnerError(spinner, sm, err)
		return err
	}

	// Describe deployment entity
	deploymentEntity, err := dataaccess.GetInstanceDeploymentEntity(cmd.Context(), token, instanceID, resourceType, deploymentName)
	if err != nil {
		utils.HandleSpinnerError(spinner, sm, err)
		return err
	}

	switch resourceType {
	case string(TerraformDeploymentType):
		// Parse JSON
		var response TerraformResponse
		err = json.Unmarshal([]byte(deploymentEntity), &response)
		if err != nil {
			utils.PrintError(errors2.Errorf("Error parsing instance deployment entity response: %v\n", err))
			return err
		}

		displayResource := TerraformResponse{}
		displayResource.Files = response.Files
		displayResource.Files.FilesContents = nil
		displayResource.SyncState = response.SyncState
		displayResource.SyncError = response.SyncError

		// Convert to JSON
		var displayOutput []byte
		displayOutput, err = json.MarshalIndent(displayResource, "", "  ")
		if err != nil {
			utils.PrintError(errors2.Errorf("Error converting instance deployment entity response to JSON: %v\n", err))
			return err
		}

		deploymentEntity = string(displayOutput)
	}

	utils.HandleSpinnerSuccess(spinner, sm, "Successfully disabled debug mode for instance deployment")
	// Print output
	err = utils.PrintTextTableJsonOutput(output, deploymentEntity)
	if err != nil {
		utils.PrintError(err)
		return err
	}

	return nil
}
