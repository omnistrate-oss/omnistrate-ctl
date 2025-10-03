package instance

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

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
	getInstallerCmd.Flags().StringP("output-path", "p", "", "Output path for the installer file (default: ./installer-{instance-id}.tar.gz)")
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
		outputPath = fmt.Sprintf("./installer-%s.tar.gz", instanceID)
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
	err = downloadFile(installerURL, outputPath, spinner)
	if err != nil {
		utils.HandleSpinnerError(spinner, sm, fmt.Errorf("failed to download installer: %w", err))
		return err
	}

	utils.HandleSpinnerSuccess(spinner, sm, fmt.Sprintf("Successfully downloaded installer to %s", outputPath))
	return nil
}

// Progress reader wrapper to track download progress
type ProgressReader struct {
	io.Reader
	Total      int64
	Downloaded int64
	Spinner    *ysmrr.Spinner
	LastUpdate time.Time
}

func (pr *ProgressReader) Read(p []byte) (int, error) {
	n, err := pr.Reader.Read(p)
	pr.Downloaded += int64(n)

	// Update spinner message every 100ms to avoid overwhelming the display
	now := time.Now()
	if now.Sub(pr.LastUpdate) > 100*time.Millisecond {
		percentage := float64(pr.Downloaded) / float64(pr.Total) * 100
		downloadedMB := float64(pr.Downloaded) / (1024 * 1024)
		totalMB := float64(pr.Total) / (1024 * 1024)

		pr.Spinner.UpdateMessage(fmt.Sprintf("Downloading installer... %.1f%% (%.1f MB / %.1f MB)",
			percentage, downloadedMB, totalMB))
		pr.LastUpdate = now
	}

	return n, err
}

func downloadFile(url, filepath string, spinner *ysmrr.Spinner) error {
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
	defer func() {
		if closeErr := out.Close(); closeErr != nil {
			// Log the error but don't override the main error
			fmt.Printf("Warning: failed to close file: %v\n", closeErr)
		}
	}()

	// Get the data
	resp, err := http.Get(url) // nolint:gosec // URL is from API response, intentional download
	if err != nil {
		return fmt.Errorf("failed to download from URL: %w", err)
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			// Log the error but don't override the main error
			fmt.Printf("Warning: failed to close response body: %v\n", closeErr)
		}
	}()

	// Check server response
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("bad status: %s", resp.Status)
	}

	// Get content length for progress calculation
	contentLength := resp.ContentLength
	if contentLength > 0 {
		// Create progress reader with known content length
		pr := &ProgressReader{
			Reader:     resp.Body,
			Total:      contentLength,
			Downloaded: 0,
			Spinner:    spinner,
			LastUpdate: time.Now(),
		}

		// Update initial message with file size
		totalMB := float64(contentLength) / (1024 * 1024)
		spinner.UpdateMessage(fmt.Sprintf("Downloading installer... 0.0%% (0.0 MB / %.1f MB)", totalMB))

		// Copy with progress tracking
		_, err = io.Copy(out, pr)
		if err != nil {
			return fmt.Errorf("failed to write file: %w", err)
		}
	} else {
		// Fallback for unknown content length
		spinner.UpdateMessage("Downloading installer (size unknown)...")
		_, err = io.Copy(out, resp.Body)
		if err != nil {
			return fmt.Errorf("failed to write file: %w", err)
		}
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
