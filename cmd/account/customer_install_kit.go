package account

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/omnistrate-oss/omnistrate-ctl/cmd/common"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/config"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/dataaccess"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/utils"
	"github.com/spf13/cobra"
)

const customerInstallKitExample = `# Download the BYOC On-Premise install kit for a customer onboarding instance
omnistrate-ctl account customer install-kit instance-abcd1234

# Download the install kit to a specific path
omnistrate-ctl account customer install-kit instance-abcd1234 --output-path /tmp/byoc-onprem-install-kit.tar`

var (
	downloadByocOnPremInstallKitFn = dataaccess.DownloadByocOnPremInstallKit
	mkdirAllInstallKitFn           = os.MkdirAll
	writeInstallKitFileFn          = os.WriteFile
	getTokenWithLoginFn            = common.GetTokenWithLogin
)

var customerInstallKitCmd = &cobra.Command{
	Use:          "install-kit [customer-account-instance-id]",
	Short:        "Download the BYOC On-Premise install kit",
	Long:         "This command downloads the BYOC On-Premise install kit for a customer onboarding instance.",
	Example:      customerInstallKitExample,
	RunE:         runCustomerInstallKit,
	SilenceUsage: true,
}

func init() {
	customerInstallKitCmd.Args = cobra.ExactArgs(1)
	customerInstallKitCmd.Flags().StringP("output-path", "p", "", "Output path for the install kit file")
}

func runCustomerInstallKit(cmd *cobra.Command, args []string) error {
	defer config.CleanupArgsAndFlags(cmd, &args)

	outputPath, _ := cmd.Flags().GetString("output-path")

	token, err := getTokenWithLoginFn()
	if err != nil {
		utils.PrintError(err)
		return err
	}

	sm := utils.NewSpinnerManager()
	spinner := sm.AddSpinner("Resolving customer BYOC On-Premise onboarding instance...")
	sm.Start()

	ref, err := resolveCustomerAccountInstanceByID(cmd.Context(), token, args[0])
	if err != nil {
		utils.HandleSpinnerError(spinner, sm, err)
		return err
	}

	instance, err := describeResourceInstanceFn(cmd.Context(), token, ref.ServiceID, ref.EnvironmentID, ref.InstanceID)
	if err != nil {
		utils.HandleSpinnerError(spinner, sm, err)
		return err
	}

	accountConfigID := extractCustomerAccountConfigID(instance)
	if accountConfigID == "" {
		err = fmt.Errorf("customer onboarding instance %s does not expose a backing account config ID", ref.InstanceID)
		utils.HandleSpinnerError(spinner, sm, err)
		return err
	}

	spinner.UpdateMessage("Downloading BYOC On-Premise install kit...")
	data, fileName, err := downloadByocOnPremInstallKitFn(cmd.Context(), token, accountConfigID)
	if err != nil {
		utils.HandleSpinnerError(spinner, sm, err)
		return err
	}

	if strings.TrimSpace(outputPath) == "" {
		outputPath = fileName
	}
	outputPath = filepath.Clean(outputPath)

	if dir := filepath.Dir(outputPath); dir != "." {
		if err = mkdirAllInstallKitFn(dir, 0755); err != nil {
			err = fmt.Errorf("failed to create install kit output directory %s: %w", dir, err)
			utils.HandleSpinnerError(spinner, sm, err)
			return err
		}
	}

	if err = writeInstallKitFileFn(outputPath, data, 0600); err != nil {
		err = fmt.Errorf("failed to write install kit to %s: %w", outputPath, err)
		utils.HandleSpinnerError(spinner, sm, err)
		return err
	}

	utils.HandleSpinnerSuccess(spinner, sm, fmt.Sprintf("Successfully downloaded install kit to %s", outputPath))

	fmt.Println()
	fmt.Println("Next steps:")
	fmt.Printf("  1. Extract the kit:  tar xf %s\n", outputPath)
	fmt.Println("  2. Run the installer from the extracted directory:  ./install.sh")
	fmt.Println("  3. Wait for the customer onboarding instance to become READY.")

	return nil
}
