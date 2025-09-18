package instance

import (
	"fmt"

	"github.com/chelnak/ysmrr"
	"github.com/omnistrate-oss/omnistrate-ctl/cmd/common"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/config"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/dataaccess"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/utils"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

const (
	modifyExample = `# Modify an instance deployment
omctl instance modify instance-abcd1234 --network-type PUBLIC / INTERNAL --param '{"databaseName":"default","password":"a_secure_password","rootPassword":"a_secure_root_password","username":"user"}'

# Modify an instance deployment using a parameter file
omctl instance modify instance-abcd1234 --param-file /path/to/param.json

# Modify an instance deployment and wait for completion with progress tracking
omctl instance modify instance-abcd1234 --param-file /path/to/param.json --wait`
)

var modifyCmd = &cobra.Command{
	Use:          "modify [instance-id]",
	Short:        "Modify an instance deployment for your service",
	Long:         `This command helps you modify the instance for your service.`,
	Example:      modifyExample,
	RunE:         runModify,
	SilenceUsage: true,
}

func init() {
	modifyCmd.Flags().String("network-type", "", "Optional network type change for the instance deployment (PUBLIC / INTERNAL)")
	modifyCmd.Flags().String("param", "", "Parameters for the instance deployment")
	modifyCmd.Flags().String("param-file", "", "Json file containing parameters for the instance deployment")
	modifyCmd.Flags().Bool("wait", false, "Wait for modification to complete and show progress")

	if err := modifyCmd.MarkFlagFilename("param-file"); err != nil {
		return
	}

	modifyCmd.Args = cobra.ExactArgs(1) // Require exactly one argument
}

func runModify(cmd *cobra.Command, args []string) error {
	defer config.CleanupArgsAndFlags(cmd, &args)

	// Retrieve args
	instanceID := args[0]

	// Retrieve flags
	output, err := cmd.Flags().GetString("output")
	if err != nil {
		utils.PrintError(err)
		return err
	}
	param, err := cmd.Flags().GetString("param")
	if err != nil {
		utils.PrintError(err)
		return err
	}
	paramFile, err := cmd.Flags().GetString("param-file")
	if err != nil {
		utils.PrintError(err)
		return err
	}
	networkType, err := cmd.Flags().GetString("network-type")
	if err != nil {
		utils.PrintError(err)
		return err
	}
	waitFlag, err := cmd.Flags().GetBool("wait")
	if err != nil {
		utils.PrintError(err)
		return err
	}

	if len(param) == 0 && len(paramFile) == 0 && len(networkType) == 0 {
		err = errors.New("at least one of --param, --param-file or --network-type must be provided")
		utils.PrintError(err)
		return err
	}

	if len(param) > 0 && len(paramFile) > 0 {
		err = errors.New("only one of --param or --param-file can be provided")
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
		msg := "Modify instance..."
		spinner = sm.AddSpinner(msg)
		sm.Start()
	}

	// Check if instance exists
	serviceID, environmentID, _, resourceID, err := getInstance(cmd.Context(), token, instanceID)
	if err != nil {
		utils.PrintError(err)
		return err
	}

	// Format parameters
	formattedParams, err := common.FormatParams(param, paramFile)
	if err != nil {
		utils.HandleSpinnerError(spinner, sm, err)
		return err
	}

	// Modify instance
	err = dataaccess.UpdateResourceInstance(cmd.Context(), token,
		serviceID,
		environmentID,
		instanceID,
		resourceID,
		utils.ToPtr(networkType),
		formattedParams,
	)
	if err != nil {
		utils.HandleSpinnerError(spinner, sm, err)
		return err
	}

	utils.HandleSpinnerSuccess(spinner, sm, "Successfully modified instance")

	// Search for the instance
	searchRes, err := dataaccess.SearchInventory(cmd.Context(), token, fmt.Sprintf("resourceinstance:%s", instanceID))
	if err != nil {
		utils.PrintError(err)
		return err
	}

	if len(searchRes.ResourceInstanceResults) == 0 {
		err = errors.New("failed to find the modified instance")
		utils.PrintError(err)
		return err
	}

	// Format instance
	formattedInstance := formatInstance(&searchRes.ResourceInstanceResults[0], false)

	// Print output
	if err = utils.PrintTextTableJsonOutput(output, formattedInstance); err != nil {
		return err
	}

	// Display workflow resource-wise data if output is not JSON and wait flag is enabled
	if output != "json" && waitFlag {
		fmt.Println("🔄 Deployment progress...")
		err = displayWorkflowResourceDataWithSpinners(cmd.Context(), token, formattedInstance.InstanceID, "modify")
		if err != nil {
			// Handle spinner error if deployment monitoring fails
			fmt.Printf("❌ Deployment failed %s\n", err)
		} else {
			fmt.Println("✅ Deployment successful")
		}
	}

	return nil
}
