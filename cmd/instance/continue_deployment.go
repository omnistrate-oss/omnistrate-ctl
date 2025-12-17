package instance

import (
	"encoding/json"
	"errors"
	"strings"

	"github.com/chelnak/ysmrr"
	"github.com/omnistrate-oss/omnistrate-ctl/cmd/common"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/config"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/dataaccess"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/utils"
	errors2 "github.com/pkg/errors"
	"github.com/spf13/cobra"
)

const (
	continueDeploymentExample = `# Continue instance deployment
omnistrate-ctl instance continue-deployment instance-abcd1234 --resource-name my-terraform-deployment --deployment-action apply`
)

var continueDeploymentCmd = &cobra.Command{
	Use:          "continue-deployment [instance-id] --resource-name <resource-name> --deployment-action <deployment-action>",
	Short:        "Continue instance deployment",
	Long:         `This command helps you continue instance deployment.`,
	Example:      continueDeploymentExample,
	RunE:         runContinueDeployment,
	SilenceUsage: true,
}

func init() {
	continueDeploymentCmd.Flags().StringP("resource-name", "r", "", "Resource name")
	continueDeploymentCmd.Flags().StringP("deployment-action", "e", "", "Deployment action")

	continueDeploymentCmd.Args = cobra.ExactArgs(1) // Require exactly one argument
	continueDeploymentCmd.Flags().StringP("output", "o", "json", "Output format. Only json is supported")

	var err error
	if err = continueDeploymentCmd.MarkFlagRequired("resource-name"); err != nil {
		return
	}
	if err = continueDeploymentCmd.MarkFlagRequired("deployment-action"); err != nil {
		return
	}
}

func runContinueDeployment(cmd *cobra.Command, args []string) error {
	defer config.CleanupArgsAndFlags(cmd, &args)

	if len(args) == 0 {
		err := errors.New("instance id is required")
		utils.PrintError(err)
		return err
	}

	// Retrieve args
	instanceID := args[0]
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

	// Retrieve flags
	output, err := cmd.Flags().GetString("output")
	if err != nil {
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
		msg := "Resuming deployment..."
		spinner = sm.AddSpinner(msg)
		sm.Start()
	}

	resourceID, resourceType, err := getResourceFromInstance(cmd.Context(), token, instanceID, resourceName)
	if err != nil {
		utils.HandleSpinnerError(spinner, sm, err)
		return err
	}

	var deploymentAction string
	if resourceType == string(TerraformDeploymentType) {
		deploymentAction, err = cmd.Flags().GetString("deployment-action")
		if err != nil {
			utils.PrintError(err)
			return err
		}

		if deploymentAction == "" {
			err = errors.New("deployment action is required")
			utils.PrintError(err)
			return err
		}
	}

	if resourceType != string(TerraformDeploymentType) {
		err = errors.New("only terraform deployment type is supported")
		utils.PrintError(err)
		return err
	}

	var deploymentName string
	switch strings.ToLower(resourceType) {
	case string(TerraformDeploymentType):
		deploymentName = getTerraformDeploymentName(resourceID, instanceID)
	}

	// Get instance deployment
	_, err = dataaccess.GetInstanceDeploymentEntity(cmd.Context(), token, instanceID, resourceType, deploymentName)
	if err != nil {
		utils.HandleSpinnerError(spinner, sm, err)
		return err
	}

	// Resume instance deployment
	err = dataaccess.ResumeInstanceDeploymentEntity(cmd.Context(), token, instanceID, resourceType, deploymentName, deploymentAction)
	if err != nil {
		utils.HandleSpinnerError(spinner, sm, err)
		return err
	}

	utils.PrintWarning("You will need to upgrade your instance to a plan version with the appropriate fix first and then disable debug mode.")

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

	utils.HandleSpinnerSuccess(spinner, sm, "Successfully continued instance deployment")
	// Print output
	err = utils.PrintTextTableJsonOutput(output, deploymentEntity)
	if err != nil {
		utils.PrintError(err)
		return err
	}

	return nil

}
