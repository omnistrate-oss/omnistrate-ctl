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

var releaseCmd = &cobra.Command{
	Use:          "release [operation] [flags]",
	Short:        "Inspect managed artifact releases",
	Run:          run,
	SilenceUsage: true,
}

var releaseListCmd = &cobra.Command{
	Use:          "list [flags]",
	Short:        "List published managed artifact releases",
	Example:      "omnistrate-ctl managed-artifact release list --limit 20\nomnistrate-ctl managed-artifact release list --bundle-version r0000020 -o json",
	Args:         cobra.NoArgs,
	RunE:         runReleaseList,
	SilenceUsage: true,
}

var releaseDescribeCmd = &cobra.Command{
	Use:          "describe [flags]",
	Short:        "Describe a managed artifact release",
	Example:      "omnistrate-ctl managed-artifact release describe --bundle-version r0000020",
	Args:         cobra.NoArgs,
	RunE:         runReleaseDescribe,
	SilenceUsage: true,
}

type releaseTableRow struct {
	BundleVersion  string `json:"bundle_version"`
	ReleasedAt     string `json:"released_at"`
	AmenityCount   int    `json:"amenity_count"`
	ArtifactCount  int    `json:"artifact_count"`
	TargetCount    int    `json:"target_count"`
	ReadyTargets   int    `json:"ready_targets"`
	FailedTargets  int    `json:"failed_targets"`
	PendingTargets int    `json:"pending_targets"`
	RunningTargets int    `json:"in_progress_targets"`
	SkippedTargets int    `json:"skipped_targets"`
}

type releaseArtifactTableRow struct {
	AmenityName  string `json:"amenity_name"`
	ArtifactKey  string `json:"artifact_key"`
	Type         string `json:"type"`
	Version      string `json:"version"`
	SourceRef    string `json:"source_ref"`
	RelativePath string `json:"relative_path"`
}

func init() {
	releaseCmd.AddCommand(releaseListCmd)
	releaseCmd.AddCommand(releaseDescribeCmd)

	releaseListCmd.Flags().String("bundle-version", "", "Filter by bundle version, for example r0000020")
	releaseListCmd.Flags().String("released-after", "", "Filter releases at or after this RFC3339 timestamp")
	releaseListCmd.Flags().String("released-before", "", "Filter releases before this RFC3339 timestamp")
	releaseListCmd.Flags().Int("limit", 20, "Maximum number of releases to return (1-100)")
	releaseListCmd.Flags().String("next-page-token", "", "Opaque token returned by the previous page")

	releaseDescribeCmd.Flags().String("bundle-version", "", "Managed artifact bundle version (required)")
	_ = releaseDescribeCmd.MarkFlagRequired("bundle-version")
}

func runReleaseList(cmd *cobra.Command, args []string) error {
	defer config.CleanupArgsAndFlags(cmd, &args)

	bundleVersion, _ := cmd.Flags().GetString("bundle-version")
	releasedAfter, _ := cmd.Flags().GetString("released-after")
	releasedBefore, _ := cmd.Flags().GetString("released-before")
	limit, _ := cmd.Flags().GetInt("limit")
	nextPageToken, _ := cmd.Flags().GetString("next-page-token")
	output, _ := cmd.Flags().GetString("output")

	if err := validateBundleVersion(bundleVersion); err != nil {
		return err
	}
	if err := validateRFC3339Flag("released-after", releasedAfter); err != nil {
		return err
	}
	if err := validateRFC3339Flag("released-before", releasedBefore); err != nil {
		return err
	}
	if err := validateLimit(limit); err != nil {
		return err
	}

	token, err := common.GetTokenWithLogin()
	if err != nil {
		return fmt.Errorf("failed to get user token: %w", err)
	}
	result, err := dataaccess.ListManagedArtifactReleases(cmd.Context(), token, dataaccess.ListManagedArtifactReleasesOptions{
		BundleVersion:  bundleVersion,
		ReleasedAfter:  releasedAfter,
		ReleasedBefore: releasedBefore,
		Limit:          limit,
		NextPageToken:  nextPageToken,
	})
	if err != nil {
		return fmt.Errorf("failed to list managed artifact releases: %w", err)
	}

	if output != "table" {
		return utils.PrintTextTableJsonOutput(output, result)
	}
	rows := make([]releaseTableRow, 0, len(result.Releases))
	for _, release := range result.Releases {
		rows = append(rows, releaseSummary(release))
	}
	if err := utils.PrintTextTableJsonArrayOutput(output, rows); err != nil {
		return err
	}
	if result.NextPageToken != "" {
		utils.PrintInfo("More releases are available; pass --next-page-token with the token shown by -o json.")
	}
	return nil
}

func runReleaseDescribe(cmd *cobra.Command, args []string) error {
	defer config.CleanupArgsAndFlags(cmd, &args)

	bundleVersion, _ := cmd.Flags().GetString("bundle-version")
	output, _ := cmd.Flags().GetString("output")
	if err := validateBundleVersion(bundleVersion); err != nil {
		return err
	}

	token, err := common.GetTokenWithLogin()
	if err != nil {
		return fmt.Errorf("failed to get user token: %w", err)
	}
	release, err := dataaccess.DescribeManagedArtifactRelease(cmd.Context(), token, bundleVersion)
	if err != nil {
		return fmt.Errorf("failed to describe managed artifact release: %w", err)
	}

	if output != "table" {
		return utils.PrintTextTableJsonOutput(output, release)
	}
	if err := utils.PrintTextTableJsonOutput(output, releaseSummary(*release)); err != nil {
		return err
	}
	artifacts := make([]releaseArtifactTableRow, 0, len(release.Artifacts))
	for _, artifact := range release.Artifacts {
		artifacts = append(artifacts, releaseArtifactTableRow{
			AmenityName:  artifact.AmenityName,
			ArtifactKey:  artifact.ArtifactKey,
			Type:         artifact.Type,
			Version:      artifact.Version,
			SourceRef:    artifact.SourceRef,
			RelativePath: artifact.RelativePath,
		})
	}
	return utils.PrintTextTableJsonArrayOutput(output, artifacts)
}

func releaseSummary(release model.ManagedArtifactRelease) releaseTableRow {
	return releaseTableRow{
		BundleVersion:  release.BundleVersion,
		ReleasedAt:     release.ReleasedAt,
		AmenityCount:   release.AmenityCount,
		ArtifactCount:  release.ArtifactCount,
		TargetCount:    release.TargetStatus.TotalTargetCount,
		ReadyTargets:   release.TargetStatus.ReadyTargetCount,
		FailedTargets:  release.TargetStatus.FailedTargetCount,
		PendingTargets: release.TargetStatus.PendingTargetCount,
		RunningTargets: release.TargetStatus.InProgressTargetCount,
		SkippedTargets: release.TargetStatus.SkippedTargetCount,
	}
}
