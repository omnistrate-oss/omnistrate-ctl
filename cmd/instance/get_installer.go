package instance

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/chelnak/ysmrr"
	"github.com/omnistrate-oss/omnistrate-ctl/cmd/common"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/config"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/dataaccess"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/utils"
	"github.com/spf13/cobra"
)

const (
	getInstallerExample = `# Get the installer for an instance
omctl instance get-installer instance-abcd1234

# Get the installer and save to a specific location
omctl instance get-installer instance-abcd1234 --output-path /tmp/installer.tar.gz`
)

var getInstallerCmd = &cobra.Command{
	Use:          "get-installer [instance-id]",
	Short:        "Download the installer for an instance",
	Long:         `This command downloads the installer for an instance and saves it locally.`,
	Example:      getInstallerExample,
	RunE:         runGetInstaller,
	SilenceUsage: true,
}

func init() {
	getInstallerCmd.Args = cobra.ExactArgs(1) // Require exactly one argument
	getInstallerCmd.Flags().StringP("output-path", "p", "", "Output path for the installer file (default: ./installer.tar.gz)")
}

func runGetInstaller(cmd *cobra.Command, args []string) error {
	defer config.CleanupArgsAndFlags(cmd, &args)

	// Retrieve args
	instanceID := args[0]

	// Retrieve flags
	outputPath, err := cmd.Flags().GetString("output-path")
	if err != nil {
		utils.PrintError(err)
		return err
	}

	// Set default output path if not provided
	if outputPath == "" {
		outputPath = "./installer.tar.gz"
	}

	// Validate user login
	token, err := common.GetTokenWithLogin()
	if err != nil {
		utils.PrintError(err)
		return err
	}

	// Initialize spinner
	sm := ysmrr.NewSpinnerManager()
	spinner := sm.AddSpinner("Getting installer information...")
	sm.Start()

	// Check if instance exists and get serviceID, environmentID
	serviceID, environmentID, _, _, err := getInstance(cmd.Context(), token, instanceID)
	if err != nil {
		utils.HandleSpinnerError(spinner, sm, err)
		return err
	}

	// Get installer information
	installerInfo, err := dataaccess.DescribeResourceInstanceInstaller(cmd.Context(), token, serviceID, environmentID, instanceID)
	if err != nil {
		utils.HandleSpinnerError(spinner, sm, err)
		return err
	}

	// Check if installer URL exists
	if !installerInfo.HasInstallerURL() || installerInfo.GetInstallerURL() == "" {
		err = fmt.Errorf("no installer exists for instance %s", instanceID)
		utils.HandleSpinnerError(spinner, sm, err)
		return err
	}

	installerURL := installerInfo.GetInstallerURL()
	spinner.UpdateMessage("Downloading installer...")

	// Download the installer
	err = downloadFile(installerURL, outputPath)
	if err != nil {
		utils.HandleSpinnerError(spinner, sm, fmt.Errorf("failed to download installer: %w", err))
		return err
	}

	utils.HandleSpinnerSuccess(spinner, sm, fmt.Sprintf("Successfully downloaded installer to %s", outputPath))
	return nil
}

func downloadFile(url, filepath string) error {
	// Create the directory if it doesn't exist
	dir := filepath_dirname(filepath)
	if dir != "" {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory: %w", err)
		}
	}

	// Create the file
	out, err := os.Create(filepath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer out.Close()

	// Get the data
	resp, err := http.Get(url) // nolint:gosec // URL is from API response, intentional download
	if err != nil {
		return fmt.Errorf("failed to download from URL: %w", err)
	}
	defer resp.Body.Close()

	// Check server response
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("bad status: %s", resp.Status)
	}

	// Write the body to file
	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	// Set appropriate file permissions
	if strings.HasSuffix(filepath, ".tar.gz") || strings.HasSuffix(filepath, ".tgz") {
		if err := os.Chmod(filepath, 0644); err != nil {
			return fmt.Errorf("failed to set file permissions: %w", err)
		}
	}

	return nil
}

func filepath_dirname(path string) string {
	dir := filepath.Dir(path)
	if dir == "." {
		return ""
	}
	return dir
}