package deploymentcell

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/omnistrate-oss/omnistrate-ctl/cmd/common"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/config"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/dataaccess"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/utils"
)

var outputDir string

var debugCmd = &cobra.Command{
	Use:          "debug",
	Short:        "Debug deployment cell resources and retrieve custom helm execution logs",
	Long:         `Debug deployment cell resources with custom helm execution logs and save them to a specified output directory.`,
	RunE:         runDebugDeploymentCell,
	SilenceUsage: true,
	Example:      `  omnistrate-ctl deployment-cell debug --id <deployment-cell-id> --output-dir ./debug-output`,
}

func init() {
	debugCmd.Flags().StringP("id", "i", "", "Deployment cell ID (required)")
	debugCmd.Flags().StringVarP(&outputDir, "output-dir", "d", "./debug-output", "Output directory to save debug logs")
	_ = debugCmd.MarkFlagRequired("id")
}

func runDebugDeploymentCell(cmd *cobra.Command, args []string) error {
	defer config.CleanupArgsAndFlags(cmd, &args)

	deploymentCellID, err := cmd.Flags().GetString("id")
	if err != nil {
		utils.PrintError(err)
		return err
	}

	token, err := common.GetTokenWithLogin()
	if err != nil {
		return fmt.Errorf("failed to get token: %w", err)
	}

	ctx := context.Background()
	// Retrieve debug data from API
	debugResult, err := dataaccess.DebugHostCluster(ctx, token, deploymentCellID)
	if err != nil {
		fmt.Printf("Warning: Failed to retrieve debug data from API: %v\n", err)
		return fmt.Errorf("failed to retrieve debug data: %w", err)
	}

	// Create output directory if it doesn't exist
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Save CustomHelmExecutionLogs to files
	if debugResult.CustomHelmExecutionLogs != nil && len(*debugResult.CustomHelmExecutionLogs) > 0 {
		for serviceName, logFile := range *debugResult.CustomHelmExecutionLogs {
			// Create a safe filename from service name
			safeServiceName := strings.ReplaceAll(serviceName, "/", "_")
			safeServiceName = strings.ReplaceAll(safeServiceName, ":", "_")

			filename := fmt.Sprintf("helm-logs-%s-%s.txt", safeServiceName, time.Now().Format("20060102-150405"))
			filePath := filepath.Join(outputDir, filename)

			// Read the log content from the file and decode base64 if needed
			var logContent []byte

			// Read content from the file
			if logFile != nil {
				// Reset file pointer to beginning
				if _, err := logFile.Seek(0, 0); err != nil {
					fmt.Printf("Warning: Failed to seek to beginning of log file for service %s: %v\n", serviceName, err)
					continue
				}

				// Read all content from the file
				content, err := os.ReadFile(logFile.Name())
				if err != nil {
					fmt.Printf("Warning: Failed to read log file for service %s: %v\n", serviceName, err)
					continue
				}

				// Try to decode as base64 first
				if decoded, decodeErr := base64.StdEncoding.DecodeString(string(content)); decodeErr == nil {
					logContent = decoded
				} else {
					// If not base64, use content as-is
					logContent = content
				}

				// Try to parse as JSON for pretty formatting
				var parsedContent interface{}
				if err := json.Unmarshal(logContent, &parsedContent); err == nil {
					// Successfully parsed as JSON, format it nicely
					if formattedJSON, err := json.MarshalIndent(parsedContent, "", "  "); err == nil {
						logContent = formattedJSON
						// Change extension to .json for formatted JSON content
						filename = fmt.Sprintf("helm-logs-%s-%s.json", safeServiceName, time.Now().Format("20060102-150405"))
						filePath = filepath.Join(outputDir, filename)
					}
				}
			} else {
				fmt.Printf("Warning: Log file is nil for service %s\n", serviceName)
				continue
			}

			if err := os.WriteFile(filePath, logContent, 0600); err != nil {
				fmt.Printf("Warning: Failed to write logs for service %s: %v\n", serviceName, err)
				continue
			}

			fmt.Printf("Saved custom helm execution logs for service '%s' to: %s\n", serviceName, filePath)
		}

		fmt.Printf("\nDebug logs successfully saved to directory: %s\n", outputDir)
	} else {
		fmt.Println("No custom helm execution logs found in debug result")
	}

	return nil
}
