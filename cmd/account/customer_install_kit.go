package account

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/omnistrate-oss/omnistrate-ctl/cmd/common"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/config"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/dataaccess"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/utils"
	"github.com/spf13/cobra"
)

const customerInstallKitExample = `# Re-download the BYOC On-Premise install kit for a customer onboarding instance
omnistrate-ctl account customer install-kit instance-abcd1234

# Re-download the install kit to a specific path
omnistrate-ctl account customer install-kit instance-abcd1234 --output-path /tmp/byoc-onprem-install-kit.tar`

var (
	downloadByocOnPremInstallKitFn = dataaccess.DownloadByocOnPremInstallKit
	mkdirAllInstallKitFn           = os.MkdirAll
	openInstallKitFileFn           = os.OpenFile
	getTokenWithLoginFn            = common.GetTokenWithLogin
)

var customerInstallKitCmd = &cobra.Command{
	Use:          "install-kit [customer-account-instance-id]",
	Short:        "Re-download the BYOC On-Premise install kit",
	Long:         "This command re-downloads the BYOC On-Premise install kit for an existing customer onboarding instance. New BYOC On-Premise customer onboarding instances automatically download the generated install kit during account customer create.",
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

	if strings.TrimSpace(outputPath) == "" {
		outputPath = dataaccess.ByocOnPremInstallKitFileName(accountConfigID)
	}
	outputPath = filepath.Clean(outputPath)

	if dir := filepath.Dir(outputPath); dir != "." {
		if err = mkdirAllInstallKitFn(dir, 0755); err != nil {
			err = fmt.Errorf("failed to create install kit output directory %s: %w", dir, err)
			utils.HandleSpinnerError(spinner, sm, err)
			return err
		}
	}

	file, err := openInstallKitFileFn(outputPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		err = fmt.Errorf("failed to create install kit file %s: %w", outputPath, err)
		utils.HandleSpinnerError(spinner, sm, err)
		return err
	}
	defer func() {
		_ = file.Close()
	}()

	spinner.UpdateMessage("Downloading BYOC On-Premise install kit...")
	if err = downloadByocOnPremInstallKitFn(cmd.Context(), token, accountConfigID, file); err != nil {
		utils.HandleSpinnerError(spinner, sm, err)
		return err
	}
	if err = closeInstallKitWriter(file); err != nil {
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

func closeInstallKitWriter(closer io.Closer) error {
	if err := closer.Close(); err != nil {
		return fmt.Errorf("failed to close install kit file: %w", err)
	}
	return nil
}
