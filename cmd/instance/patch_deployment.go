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
	openapiclientfleet "github.com/omnistrate-oss/omnistrate-sdk-go/fleet"
	errors2 "github.com/pkg/errors"
	"github.com/spf13/cobra"
)

const (
	patchDeploymentExample = `# Patch deployment for an instance deployment
omctl instance patch-deployment instance-abcd1234 --resource-name my-terraform-deployment --deployment-action apply --patch-files /patchedFiles`
)

var patchDeploymentCmd = &cobra.Command{
	Use:          "patch-deployment [instance-id] --resource-name <resource-name> --deployment-action <deployment-action> --patch-files <patch-files>",
	Short:        "Patch deployment for an instance deployment",
	Long:         `This command helps you patch the deployment for an instance deployment.`,
	Example:      patchDeploymentExample,
	RunE:         runPatchDeployment,
	SilenceUsage: true,
}

func init() {
	patchDeploymentCmd.Flags().StringP("resource-name", "r", "", "Resource name")
	patchDeploymentCmd.Flags().StringP("deployment-action", "e", "", "Deployment action")
	patchDeploymentCmd.Flags().StringP("patch-files", "p", "", "Patch files")

	patchDeploymentCmd.Args = cobra.ExactArgs(1) // Require exactly one argument
	patchDeploymentCmd.Flags().StringP("output", "o", "json", "Output format. Only json is supported")

	var err error
	if err = patchDeploymentCmd.MarkFlagRequired("resource-name"); err != nil {
		return
	}
	if err = patchDeploymentCmd.MarkFlagRequired("deployment-action"); err != nil {
		return
	}
	if err = patchDeploymentCmd.MarkFlagRequired("patch-files"); err != nil {
		return
	}
}

func runPatchDeployment(cmd *cobra.Command, args []string) error {
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

	// Validate user login
	token, err := common.GetTokenWithLogin()
	if err != nil {
		utils.PrintError(err)
		return err
	}

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

	// Initialize spinner if output is not JSON
	var sm ysmrr.SpinnerManager
	var spinner *ysmrr.Spinner
	if output != "json" {
		sm = ysmrr.NewSpinnerManager()
		msg := "Patching deployment..."
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

	var deploymentAction string
	if resourceType == string(TerraformDeploymentType) {
		deploymentAction, err = cmd.Flags().GetString("deployment-action")
		if err != nil {
			utils.PrintError(err)
			return err
		}
	}

	if resourceType != string(TerraformDeploymentType) {
		err = errors.New("only terraform deployment type is supported")
		utils.PrintError(err)
		return err
	}

	patchedFilePath, err := cmd.Flags().GetString("patch-files")
	if err != nil {
		utils.PrintError(err)
		return err
	}

	// validate patch files
	if patchedFilePath == "" {
		err = errors.New("patch files cannot be empty")
		utils.PrintError(err)
		return err
	}

	// Describe instance
	var instance *openapiclientfleet.ResourceInstance
	instance, err = dataaccess.DescribeResourceInstance(cmd.Context(), token, serviceID, environmentID, instanceID, true)
	if err != nil {
		utils.HandleSpinnerError(spinner, sm, err)
		return err
	}

	if instance.ManualOverride == nil {
		err = errors.New("debug mode is not enabled for this instance")
		utils.PrintError(err)
		return err
	}

	err = dataaccess.PatchInstanceDeploymentEntity(cmd.Context(), token, instanceID, resourceType, deploymentName, patchedFilePath, deploymentAction)
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

	utils.HandleSpinnerSuccess(spinner, sm, "Successfully enabled override for instance deployment")
	// Print output
	err = utils.PrintTextTableJsonOutput(output, deploymentEntity)
	if err != nil {
		utils.PrintError(err)
		return err
	}

	return nil
}
