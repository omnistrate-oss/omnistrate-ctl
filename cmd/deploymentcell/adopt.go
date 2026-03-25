package deploymentcell

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/omnistrate-oss/omnistrate-ctl/cmd/common"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/config"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/dataaccess"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/utils"
	"github.com/spf13/cobra"
)

var adoptCmd = &cobra.Command{
	Use:          "adopt",
	Short:        "Adopt a deployment cell",
	Long:         `Adopt a deployment cell with the specified parameters.`,
	RunE:         runAdopt,
	SilenceUsage: true,
}

func init() {
	adoptCmd.Flags().StringP("id", "i", "", "Deployment cell ID (required)")
	adoptCmd.Flags().StringP("cloud-provider", "c", "", "Cloud provider name (required)")
	adoptCmd.Flags().StringP("region", "r", "", "Region name (required)")
	adoptCmd.Flags().StringP("description", "d", "Deployment cell adopted via CLI", "Description for the deployment cell")
	adoptCmd.Flags().StringP("customer-email", "u", "", "Customer email to adopt the deployment cell for (optional)")

	_ = adoptCmd.MarkFlagRequired("id")
	_ = adoptCmd.MarkFlagRequired("cloud-provider")
	_ = adoptCmd.MarkFlagRequired("region")
}

func runAdopt(cmd *cobra.Command, args []string) error {
	defer config.CleanupArgsAndFlags(cmd, &args)

	id, err := cmd.Flags().GetString("id")
	if err != nil {
		utils.PrintError(err)
		return err
	}

	cloudProvider, err := cmd.Flags().GetString("cloud-provider")
	if err != nil {
		utils.PrintError(err)
		return err
	}

	region, err := cmd.Flags().GetString("region")
	if err != nil {
		utils.PrintError(err)
		return err
	}

	description, err := cmd.Flags().GetString("description")
	if err != nil {
		utils.PrintError(err)
		return err
	}

	userEmail, err := cmd.Flags().GetString("customer-email")
	if err != nil {
		utils.PrintError(err)
		return err
	}

	ctx := context.Background()
	token, err := common.GetTokenWithLogin()
	if err != nil {
		utils.PrintError(err)
		return err
	}

	fmt.Printf("Adopting deployment cell:\n")
	fmt.Printf("  ID: %s\n", id)
	fmt.Printf("  Cloud Provider: %s\n", cloudProvider)
	fmt.Printf("  Region: %s\n", region)
	fmt.Printf("  Description: %s\n", description)
	if userEmail != "" {
		fmt.Printf("  User Email: %s\n", userEmail)
	}

	// Perform the actual adoption using the SDK
	var userEmailPtr *string
	if userEmail != "" {
		userEmailPtr = &userEmail
	}

	result, err := dataaccess.AdoptHostCluster(ctx, token, id, cloudProvider, region, description, userEmailPtr)
	if err != nil {
		utils.PrintError(fmt.Errorf("failed to adopt deployment cell: %w", err))
		return err
	}

	utils.PrintSuccess("Deployment cell adoption initiated successfully!")
	fmt.Printf("Adoption result:\n")
	fmt.Printf("  Adoption Status: %s\n", result.GetAdoptionStatus())

	if result.GetAgentInstallationKit() != "" {
		// Save the installation kit as a tar file
		tarFileName := fmt.Sprintf("%s.tar", id)
		err := saveInstallationKit(result.GetAgentInstallationKit(), tarFileName)
		if err != nil {
			utils.PrintWarning(fmt.Sprintf("Failed to save installation kit: %v", err))
		} else {
			fmt.Printf("  Agent Installation Kit: Saved as %s\n", tarFileName)
			fmt.Printf("  Note: Use the agent installation kit to complete the adoption process\n")
		}
	} else {
		fmt.Printf("  Agent Installation Kit: Not provided\n")
	}

	return nil
}

func saveInstallationKit(base64EncodedKit string, fileName string) error {
	// Get current working directory
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	// Create the full path for the tar file
	fullPath := filepath.Join(cwd, fileName)

	// Create output file
	outFile, err := os.Create(fullPath)
	if err != nil {
		return fmt.Errorf("failed to create output file %s: %w", fullPath, err)
	}
	defer func() {
		_ = outFile.Close()
	}()

	// Decode the base64 encoded installation kit
	var kitData []byte
	if kitData, err = base64.StdEncoding.DecodeString(base64EncodedKit); err != nil {
		return fmt.Errorf("failed to decode base64 installation kit: %w", err)
	}

	// Create a reader for the installation kit
	kitFile := io.NopCloser(bytes.NewReader(kitData))

	// Copy the contents
	_, err = io.Copy(outFile, kitFile)
	if err != nil {
		return fmt.Errorf("failed to copy installation kit contents: %w", err)
	}

	return nil
}
