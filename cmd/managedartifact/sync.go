package managedartifact

import (
	"fmt"

	"github.com/omnistrate-oss/omnistrate-ctl/cmd/common"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/config"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/dataaccess"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/model"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/utils"
	"github.com/spf13/cobra"
)

var syncCmd = &cobra.Command{
	Use:          "sync [operation] [flags]",
	Short:        "Inspect provisioner managed artifact synchronization",
	Run:          run,
	SilenceUsage: true,
}

var syncListCmd = &cobra.Command{
	Use:          "list [flags]",
	Short:        "List managed artifact synchronization records",
	Example:      "omnistrate-ctl managed-artifact sync list --status failed\nomnistrate-ctl managed-artifact sync list --target-id hc-123 -o json",
	Args:         cobra.NoArgs,
	RunE:         runSyncList,
	SilenceUsage: true,
}

var syncDescribeCmd = &cobra.Command{
	Use:          "describe [flags]",
	Short:        "Describe a managed artifact synchronization record",
	Example:      "omnistrate-ctl managed-artifact sync describe --id spabs-123",
	Args:         cobra.NoArgs,
	RunE:         runSyncDescribe,
	SilenceUsage: true,
}

type syncTableRow struct {
	ID                 string `json:"id"`
	BundleVersion      string `json:"bundle_version"`
	TargetID           string `json:"target_id"`
	TargetName         string `json:"target_name"`
	CloudProvider      string `json:"cloud_provider"`
	Region             string `json:"region"`
	Status             string `json:"status"`
	ArtifactProgress   string `json:"artifact_progress"`
	FailedArtifacts    int    `json:"failed_artifacts"`
	LastTransitionTime string `json:"last_transition_time"`
}

type syncArtifactTableRow struct {
	AmenityName     string `json:"amenity_name"`
	ArtifactKey     string `json:"artifact_key"`
	Version         string `json:"version"`
	Status          string `json:"status"`
	FailureCategory string `json:"failure_category"`
	FailureMessage  string `json:"failure_message"`
}

func init() {
	syncCmd.AddCommand(syncListCmd)
	syncCmd.AddCommand(syncDescribeCmd)

	syncListCmd.Flags().String("bundle-version", "", "Filter by bundle version, for example r0000020")
	syncListCmd.Flags().String("status", "", "Filter by status: pending, in_progress, ready, failed, or skipped")
	syncListCmd.Flags().String("target-id", "", "Filter by provisioner target ID")
	syncListCmd.Flags().String("updated-after", "", "Filter syncs updated at or after this RFC3339 timestamp")
	syncListCmd.Flags().String("updated-before", "", "Filter syncs updated before this RFC3339 timestamp")
	syncListCmd.Flags().Int("limit", 20, "Maximum number of syncs to return (1-100)")
	syncListCmd.Flags().String("next-page-token", "", "Opaque token returned by the previous page")

	syncDescribeCmd.Flags().String("id", "", "Managed artifact synchronization ID (required)")
	_ = syncDescribeCmd.MarkFlagRequired("id")
}

func runSyncList(cmd *cobra.Command, args []string) error {
	defer config.CleanupArgsAndFlags(cmd, &args)

	bundleVersion, _ := cmd.Flags().GetString("bundle-version")
	status, _ := cmd.Flags().GetString("status")
	targetID, _ := cmd.Flags().GetString("target-id")
	updatedAfter, _ := cmd.Flags().GetString("updated-after")
	updatedBefore, _ := cmd.Flags().GetString("updated-before")
	limit, _ := cmd.Flags().GetInt("limit")
	nextPageToken, _ := cmd.Flags().GetString("next-page-token")
	output, _ := cmd.Flags().GetString("output")

	if err := validateBundleVersion(bundleVersion); err != nil {
		return err
	}
	status, err := normalizeSyncStatus(status)
	if err != nil {
		return err
	}
	if err := validateRFC3339Flag("updated-after", updatedAfter); err != nil {
		return err
	}
	if err := validateRFC3339Flag("updated-before", updatedBefore); err != nil {
		return err
	}
	if err := validateLimit(limit); err != nil {
		return err
	}

	token, err := common.GetTokenWithLogin()
	if err != nil {
		return fmt.Errorf("failed to get user token: %w", err)
	}
	result, err := dataaccess.ListManagedArtifactSyncs(cmd.Context(), token, dataaccess.ListManagedArtifactSyncsOptions{
		BundleVersion: bundleVersion,
		Status:        status,
		TargetID:      targetID,
		UpdatedAfter:  updatedAfter,
		UpdatedBefore: updatedBefore,
		Limit:         limit,
		NextPageToken: nextPageToken,
	})
	if err != nil {
		return fmt.Errorf("failed to list managed artifact syncs: %w", err)
	}

	if output != "table" {
		return utils.PrintTextTableJsonOutput(output, result)
	}
	rows := make([]syncTableRow, 0, len(result.Syncs))
	for _, sync := range result.Syncs {
		rows = append(rows, syncSummary(sync))
	}
	if err := utils.PrintTextTableJsonArrayOutput(output, rows); err != nil {
		return err
	}
	if result.NextPageToken != "" {
		utils.PrintInfo(fmt.Sprintf("More syncs are available; pass --next-page-token %s to fetch the next page.", result.NextPageToken))
	}
	return nil
}

func runSyncDescribe(cmd *cobra.Command, args []string) error {
	defer config.CleanupArgsAndFlags(cmd, &args)

	syncID, _ := cmd.Flags().GetString("id")
	output, _ := cmd.Flags().GetString("output")
	if err := validateSyncID(syncID); err != nil {
		return err
	}

	token, err := common.GetTokenWithLogin()
	if err != nil {
		return fmt.Errorf("failed to get user token: %w", err)
	}
	sync, err := dataaccess.DescribeManagedArtifactSync(cmd.Context(), token, syncID)
	if err != nil {
		return fmt.Errorf("failed to describe managed artifact sync: %w", err)
	}

	if output != "table" {
		return utils.PrintTextTableJsonOutput(output, sync)
	}
	if err := utils.PrintTextTableJsonOutput(output, syncSummary(*sync)); err != nil {
		return err
	}
	artifacts := make([]syncArtifactTableRow, 0, len(sync.Artifacts))
	for _, artifact := range sync.Artifacts {
		artifacts = append(artifacts, syncArtifactTableRow{
			AmenityName:     artifact.AmenityName,
			ArtifactKey:     artifact.ArtifactKey,
			Version:         artifact.Version,
			Status:          artifact.Status,
			FailureCategory: artifact.FailureCategory,
			FailureMessage:  artifact.FailureMessage,
		})
	}
	return utils.PrintTextTableJsonArrayOutput(output, artifacts)
}

func syncSummary(sync model.ManagedArtifactSync) syncTableRow {
	return syncTableRow{
		ID:                 sync.ID,
		BundleVersion:      sync.BundleVersion,
		TargetID:           sync.Target.ID,
		TargetName:         sync.Target.Name,
		CloudProvider:      sync.Target.CloudProvider,
		Region:             sync.Target.Region,
		Status:             sync.Status,
		ArtifactProgress:   fmt.Sprintf("%d/%d", sync.CompletedArtifactCount, sync.ArtifactCount),
		FailedArtifacts:    sync.FailedArtifactCount,
		LastTransitionTime: sync.LastTransitionTime,
	}
}
