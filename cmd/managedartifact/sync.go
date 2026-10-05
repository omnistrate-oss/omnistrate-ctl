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
	Short:        "Inspect private artifact syncs and public ECR publications",
	Run:          run,
	SilenceUsage: true,
}

var syncListCmd = &cobra.Command{
	Use:   "list [flags]",
	Short: "List managed artifact synchronization records",
	Long: `List private ECR synchronization records or public ECR publications.

Omitting --registry-type preserves private ECR behavior. --destination-account-id
filters either registry type by its destination AWS account. When --target-id is
also supplied, both filters must match. Public publications may have no execution
target yet. These filters require the public ECR managed-artifact API update.
Returned records must match the requested registry and destination account; an
empty result alone cannot establish backend support. When paging, keep all
filters unchanged, including --registry-type and --destination-account-id.`,
	Example:      "omnistrate-ctl managed-artifact sync list --status failed\nomnistrate-ctl managed-artifact sync list --registry-type public_ecr --destination-account-id 123456789012 -o json\nomnistrate-ctl managed-artifact sync list --target-id hc-123 -o json",
	Args:         cobra.NoArgs,
	RunE:         runSyncList,
	SilenceUsage: true,
}

var syncDescribeCmd = &cobra.Command{
	Use:          "describe [flags]",
	Short:        "Describe a managed artifact synchronization record",
	Long:         "Describe a private synchronization or public publication. Use --output json for source references, digests, completion timestamps, and full execution-target metadata.",
	Example:      "omnistrate-ctl managed-artifact sync describe --id spabs-123\nomnistrate-ctl managed-artifact sync describe --id sppap-123 -o json",
	Args:         cobra.NoArgs,
	RunE:         runSyncDescribe,
	SilenceUsage: true,
}

type syncTableRow struct {
	ID                   string `json:"id"`
	BundleVersion        string `json:"bundle_version"`
	RegistryType         string `json:"registry_type"`
	DestinationAccountID string `json:"destination_account_id"`
	DestinationRegistry  string `json:"destination_registry"`
	ExecutionTargetID    string `json:"execution_target_id"`
	ExecutionRegion      string `json:"execution_region"`
	Status               string `json:"status"`
	ArtifactProgress     string `json:"artifact_progress"`
	FailedArtifacts      int    `json:"failed_artifacts"`
	LastTransitionTime   string `json:"last_transition_time"`
}

type syncArtifactTableRow struct {
	AmenityName     string `json:"amenity_name"`
	ArtifactKey     string `json:"artifact_key"`
	Version         string `json:"version"`
	Status          string `json:"status"`
	DestinationRef  string `json:"destination_ref"`
	FailureCategory string `json:"failure_category"`
	FailureMessage  string `json:"failure_message"`
}

func init() {
	syncCmd.AddCommand(syncListCmd)
	syncCmd.AddCommand(syncDescribeCmd)

	syncListCmd.Flags().String("bundle-version", "", "Filter by bundle version, for example r0000020")
	syncListCmd.Flags().String("status", "", "Filter by status: pending, in_progress, ready, failed, or skipped")
	syncListCmd.Flags().String("registry-type", "", "Registry type: private_ecr or public_ecr (omitted defaults to private ECR)")
	syncListCmd.Flags().String("destination-account-id", "", "Filter either ECR registry type by 12-digit destination AWS account ID")
	syncListCmd.Flags().String("target-id", "", "Filter by execution provisioner ID; must also match any destination account filter")
	syncListCmd.Flags().String("updated-after", "", "Filter syncs updated at or after this RFC3339 timestamp")
	syncListCmd.Flags().String("updated-before", "", "Filter syncs updated before this RFC3339 timestamp")
	syncListCmd.Flags().Int("limit", 20, "Maximum number of syncs to return (1-100)")
	syncListCmd.Flags().String("next-page-token", "", "Opaque token returned by the previous page")

	syncDescribeCmd.Flags().String("id", "", "Private sync (spabs-*) or public publication (sppap-*) ID (required)")
	_ = syncDescribeCmd.MarkFlagRequired("id")
}

func runSyncList(cmd *cobra.Command, args []string) error {
	defer config.CleanupArgsAndFlags(cmd, &args)

	bundleVersion, _ := cmd.Flags().GetString("bundle-version")
	status, _ := cmd.Flags().GetString("status")
	registryType, _ := cmd.Flags().GetString("registry-type")
	destinationAccountID, _ := cmd.Flags().GetString("destination-account-id")
	targetID, _ := cmd.Flags().GetString("target-id")
	updatedAfter, _ := cmd.Flags().GetString("updated-after")
	updatedBefore, _ := cmd.Flags().GetString("updated-before")
	limit, _ := cmd.Flags().GetInt("limit")
	nextPageToken, _ := cmd.Flags().GetString("next-page-token")
	output, _ := cmd.Flags().GetString(common.OutputFlag)

	if err := validateBundleVersion(bundleVersion); err != nil {
		return err
	}
	registryType, err := normalizeRegistryType(registryType)
	if err != nil {
		return err
	}
	if destinationAccountID != "" && !destinationAccountIDPattern.MatchString(destinationAccountID) {
		return fmt.Errorf("--destination-account-id must contain exactly 12 digits")
	}
	status, err = normalizeSyncStatus(status)
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
	var sm utils.SpinnerManager
	var spinner *utils.Spinner
	if output != common.OutputTypeJson {
		sm = utils.NewSpinnerManager()
		spinner = sm.AddSpinner("Listing managed artifact syncs...")
		sm.Start()
	}
	result, err := dataaccess.ListManagedArtifactSyncs(cmd.Context(), token, dataaccess.ListManagedArtifactSyncsOptions{
		RegistryType:         registryType,
		DestinationAccountID: destinationAccountID,
		BundleVersion:        bundleVersion,
		Status:               status,
		TargetID:             targetID,
		UpdatedAfter:         updatedAfter,
		UpdatedBefore:        updatedBefore,
		Limit:                limit,
		NextPageToken:        nextPageToken,
	})
	if err != nil {
		err = fmt.Errorf("failed to list managed artifact syncs: %w", err)
		utils.HandleSpinnerError(spinner, sm, err)
		return err
	}
	expectedRegistryType := registryType
	if expectedRegistryType == "" {
		expectedRegistryType = "PRIVATE_ECR"
	}
	for _, sync := range result.Syncs {
		if ((registryType != "" || destinationAccountID != "") && sync.RegistryType != expectedRegistryType) ||
			(destinationAccountID != "" && (sync.Destination == nil || sync.Destination.AccountID != destinationAccountID)) {
			err = fmt.Errorf("backend did not honor --registry-type or --destination-account-id; update the managed-artifact API before using these filters")
			utils.HandleSpinnerError(spinner, sm, err)
			return err
		}
	}
	utils.HandleSpinnerSuccess(spinner, sm, "Successfully listed managed artifact syncs")

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
		utils.PrintInfo(fmt.Sprintf("More syncs are available; pass --next-page-token %s with the same filters, including --registry-type and --destination-account-id, to fetch the next page.", result.NextPageToken))
	}
	return nil
}

func runSyncDescribe(cmd *cobra.Command, args []string) error {
	defer config.CleanupArgsAndFlags(cmd, &args)

	syncID, _ := cmd.Flags().GetString("id")
	output, _ := cmd.Flags().GetString(common.OutputFlag)
	if err := validateSyncID(syncID); err != nil {
		return err
	}

	token, err := common.GetTokenWithLogin()
	if err != nil {
		return fmt.Errorf("failed to get user token: %w", err)
	}
	var sm utils.SpinnerManager
	var spinner *utils.Spinner
	if output != common.OutputTypeJson {
		sm = utils.NewSpinnerManager()
		spinner = sm.AddSpinner(fmt.Sprintf("Describing managed artifact sync %s...", syncID))
		sm.Start()
	}
	sync, err := dataaccess.DescribeManagedArtifactSync(cmd.Context(), token, syncID)
	if err != nil {
		err = fmt.Errorf("failed to describe managed artifact sync: %w", err)
		utils.HandleSpinnerError(spinner, sm, err)
		return err
	}
	utils.HandleSpinnerSuccess(spinner, sm, fmt.Sprintf("Successfully described managed artifact sync %s", syncID))

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
			DestinationRef:  artifact.DestinationRef,
			FailureCategory: artifact.FailureCategory,
			FailureMessage:  artifact.FailureMessage,
		})
	}
	return utils.PrintTextTableJsonArrayOutput(output, artifacts)
}

func syncSummary(sync model.ManagedArtifactSync) syncTableRow {
	row := syncTableRow{
		ID:                 sync.ID,
		BundleVersion:      sync.BundleVersion,
		RegistryType:       sync.RegistryType,
		Status:             sync.Status,
		ArtifactProgress:   fmt.Sprintf("%d/%d", sync.CompletedArtifactCount, sync.ArtifactCount),
		FailedArtifacts:    sync.FailedArtifactCount,
		LastTransitionTime: sync.LastTransitionTime,
	}
	if row.RegistryType == "" {
		row.RegistryType = "PRIVATE_ECR"
	}
	if sync.Target != nil {
		row.ExecutionTargetID, row.ExecutionRegion = sync.Target.ID, sync.Target.Region
		if row.RegistryType == "PRIVATE_ECR" {
			row.DestinationAccountID = sync.Target.AccountID
		}
	}
	if sync.Destination != nil {
		row.DestinationRegistry = sync.Destination.Registry
		if sync.Destination.AccountID != "" {
			row.DestinationAccountID = sync.Destination.AccountID
		}
	}
	return row
}
