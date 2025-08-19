package deploymentcell

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/omnistrate-oss/omnistrate-ctl/cmd/common"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/dataaccess"
)

var outputDir string

var debugCmd = &cobra.Command{
	Use:     "debug [deployment-cell-id]",
	Short:   "Debug deployment cell resources and retrieve custom helm execution logs",
	Long:    `Debug deployment cell resources with custom helm execution logs and save them to a specified output directory.`,
	Args:    cobra.ExactArgs(1),
	RunE:    runDebugDeploymentCell,
	Example: `  omnistrate-ctl deployment-cell debug <deployment-cell-id> --output-dir ./debug-output`,
}

func init() {
	debugCmd.Flags().StringVarP(&outputDir, "output-dir", "o", "./debug-output", "Output directory to save debug logs")
}

func runDebugDeploymentCell(_ *cobra.Command, args []string) error {
	deploymentCellID := args[0]

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
		for serviceName, logs := range *debugResult.CustomHelmExecutionLogs {
			// Create a safe filename from service name
			safeServiceName := strings.ReplaceAll(serviceName, "/", "_")
			safeServiceName = strings.ReplaceAll(safeServiceName, ":", "_")

			filename := fmt.Sprintf("helm-logs-%s-%s.json", safeServiceName, time.Now().Format("20060102-150405"))
			filePath := filepath.Join(outputDir, filename)

			// Convert logs to JSON for better readability
			logsJSON, err := json.MarshalIndent(logs, "", "  ")
			if err != nil {
				fmt.Printf("Warning: Failed to marshal logs for service %s: %v\n", serviceName, err)
				continue
			}

			if err := os.WriteFile(filePath, logsJSON, 0644); err != nil {
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
