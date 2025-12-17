package instance

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/chelnak/ysmrr"
	"github.com/omnistrate-oss/omnistrate-ctl/cmd/common"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/config"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/dataaccess"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/utils"
	errors2 "github.com/pkg/errors"
	"github.com/spf13/cobra"
)

// Define structs to match the JSON structure
type TerraformResponse struct {
	SyncState string   `json:"syncState,omitempty"`
	Files     FileInfo `json:"files,omitempty"`
	SyncError string   `json:"syncError,omitempty"`
}

type FileInfo struct {
	FilesContents   map[string]string `json:"filesContents,omitempty"`
	Name            string            `json:"name,omitempty"`
	InstanceID      string            `json:"instanceID,omitempty"`
	ResourceVersion string            `json:"resourceVersion,omitempty"`
}

const (
	getDeploymentExample = `  # Get the deployment entity metadata of the instance
	  omnistrate-ctl instance get-deployment instance-abcd1234 --resource-name my-terraform-deployment --output-path /tmp`
)

var getDeploymentCmd = &cobra.Command{
	Use:          "get-deployment [instance-id] --resource-name <resource-name> --output-path <output-path>",
	Short:        "Get the deployment entity metadata of the instance",
	Long:         `This command helps you get the deployment entity metadata of the instance.`,
	Example:      getDeploymentExample,
	RunE:         runGetDeployment,
	SilenceUsage: true,
}

func init() {
	getDeploymentCmd.Flags().StringP("resource-name", "r", "", "Resource name")
	getDeploymentCmd.Flags().StringP("output-path", "p", "", "Output path")

	getDeploymentCmd.Args = cobra.ExactArgs(1) // Require exactly one argument
	getDeploymentCmd.Flags().StringP("output", "o", "json", "Output format. Only json is supported")

	var err error
	if err = getDeploymentCmd.MarkFlagRequired("resource-name"); err != nil {
		return
	}
}

func runGetDeployment(cmd *cobra.Command, args []string) error {
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
		msg := "Getting deployment entity metadata..."
		spinner = sm.AddSpinner(msg)
		sm.Start()
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

		var outputPath string
		outputPath, err = cmd.Flags().GetString("output-path")
		if err != nil {
			utils.PrintError(err)
			return err
		}

		if outputPath == "" {
			err = errors.New("output-path is required")
			utils.PrintError(err)
			return err
		}

		// Set local space
		err = setupTerraformWorkspace(response, outputPath)
		if err != nil {
			utils.PrintError(errors2.Errorf("Error setting up terraform workspace: %v\n", err))
			return err
		}

		utils.PrintInfo(fmt.Sprintf("Terraform workspace setup at: %s", outputPath))

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

	utils.HandleSpinnerSuccess(spinner, sm, "Successfully got deployment entity metadata")

	// Print output
	err = utils.PrintTextTableJsonOutput(output, deploymentEntity)
	if err != nil {
		utils.PrintError(err)
		return err
	}

	return nil
}

func setupTerraformWorkspace(response TerraformResponse, outputPath string) (err error) {
	// Create directory for files
	dirName := outputPath

	// check if directory exists
	if _, err = os.Stat(dirName); err != nil {
		if os.IsNotExist(err) {
			// Create the directory if it doesn't exist
			if err = os.MkdirAll(dirName, 0755); err != nil {
				return
			}
		}
	}

	// Decode and write each file
	for filename, content := range response.Files.FilesContents {
		var decoded []byte
		// Decode base64 content
		decoded, err = base64.StdEncoding.DecodeString(content)
		if err != nil {
			err = errors2.Wrap(err, fmt.Sprintf("Error decoding file %s", filename))
			return
		}

		// Create full file path
		filePath := filepath.Join(dirName, filename)

		// Get directory of the file
		dir := filepath.Dir(filePath)
		// Create the directory if it doesn't exist
		if err = os.MkdirAll(dir, 0755); err != nil {
			return
		}

		// Write file
		err = os.WriteFile(filePath, decoded, 0600)
		if err != nil {
			err = errors2.Wrap(err, fmt.Sprintf("Error writing file %s", filename))
			return
		}
	}

	return
}
