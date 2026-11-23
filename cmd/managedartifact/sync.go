package managedartifact

import (
	"fmt"
	"strings"

	"github.com/omnistrate-oss/omnistrate-ctl/cmd/common"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/artifactsync"
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
	Long:         "Describe a private synchronization or public publication. Artifact tables include source and destination references. Use --output json for digests, completion timestamps, and full execution-target metadata.",
	Example:      "omnistrate-ctl managed-artifact sync describe --id spabs-123\nomnistrate-ctl managed-artifact sync describe --id sppap-123 -o json",
	Args:         cobra.NoArgs,
	RunE:         runSyncDescribe,
	SilenceUsage: true,
}

var syncCopyCmd = &cobra.Command{
	Use:   "copy [flags]",
	Short: "Copy and optionally sign a MANUAL managed artifact sync",
	Long: `Fetch the server-authored copy plan and copy every artifact by digest to the persisted destination prefix in the destination ECR registry. Every artifact is copied to a staging tag first; the optional customer vetter then runs for every artifact, followed by the optional signer. Canonical and immutable version tags are published only after both hooks succeed for every artifact, and staging tags are then removed. After a successful copy, the command asks the control plane to refresh the sync status and prints the result; use 'sync refresh-status' to request destination verification again.

Hooks run directly without a shell, receive one JSON object on stdin, and inherit AWS_PROFILE, AWS_REGION, and AWS_DEFAULT_REGION for the selected destination. With --aws-profile, hooks also receive the profile's resolved credentials as AWS_ACCESS_KEY_ID, AWS_SECRET_ACCESS_KEY, and AWS_SESSION_TOKEN in place of any inherited AWS credential variables, so they authenticate as the same identity as the copy. The JSON includes artifactKey, sourceRef, destinationRef, destinationDigest, destinationRegion, and awsProfile. Hook output is written to stderr.

Re-running the command is safe: artifacts whose version tag already references the planned digest are skipped, including their hooks. Source registries are read with local Docker credentials. Repository policies that grant other accounts pull access are not managed by this command.`,
	Example:      "omnistrate-ctl managed-artifact sync copy --id spabs-123 --aws-profile customer-prod --vetter ./vet-artifact --signer ./sign-artifact",
	Args:         cobra.NoArgs,
	RunE:         runSyncCopy,
	SilenceUsage: true,
}

var syncRefreshStatusCmd = &cobra.Command{
	Use:          "refresh-status [flags]",
	Short:        "Verify a customer-populated MANUAL sync destination",
	Long:         `Refresh the status of one private MANUAL sync after the customer copies its artifacts. The control plane only verifies the existing destination and never starts a platform copy. A complete destination transitions to READY; an incomplete destination remains MANUAL with per-artifact results. Requests for any other status are rejected.`,
	Example:      "omnistrate-ctl managed-artifact sync refresh-status --id spabs-123",
	Args:         cobra.NoArgs,
	RunE:         runSyncRefreshStatus,
	SilenceUsage: true,
}

var syncReconcileCmd = &cobra.Command{
	Use:          "reconcile [flags]",
	Short:        "Run one platform push for a private managed artifact sync",
	Long:         `Run one platform-managed copy and verification for a private MANUAL, FAILED, SKIPPED, or READY sync and its exact bundle version. Automatic private publication must not be explicitly disabled. MANUAL, FAILED, and SKIPPED enter IN_PROGRESS after dispatch, become READY on success, and restore their previous status on failure. READY remains READY throughout and is unchanged if the republish fails. IN_PROGRESS requests and READY requests with an active republish are rejected. This command never retries other syncs or versions in the background.`,
	Example:      "omnistrate-ctl managed-artifact sync reconcile --id spabs-123",
	Args:         cobra.NoArgs,
	RunE:         runSyncReconcile,
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
	SourceRef       string `json:"source_ref"`
	DestinationRef  string `json:"destination_ref"`
	FailureCategory string `json:"failure_category"`
	FailureMessage  string `json:"failure_message"`
}

func init() {
	syncCmd.AddCommand(syncListCmd)
	syncCmd.AddCommand(syncDescribeCmd)
	syncCmd.AddCommand(syncCopyCmd)
	syncCmd.AddCommand(syncRefreshStatusCmd)
	syncCmd.AddCommand(syncReconcileCmd)

	syncListCmd.Flags().String("bundle-version", "", "Filter by bundle version, for example r0000020")
	syncListCmd.Flags().String("status", "", "Filter by status: pending, in_progress, ready, failed, skipped, or manual")
	syncListCmd.Flags().String("registry-type", "", "Registry type: private_ecr or public_ecr (omitted defaults to private ECR)")
	syncListCmd.Flags().String("destination-account-id", "", "Filter either ECR registry type by 12-digit destination AWS account ID")
	syncListCmd.Flags().String("target-id", "", "Filter by execution provisioner ID; must also match any destination account filter")
	syncListCmd.Flags().String("updated-after", "", "Filter syncs updated at or after this RFC3339 timestamp")
	syncListCmd.Flags().String("updated-before", "", "Filter syncs updated before this RFC3339 timestamp")
	syncListCmd.Flags().Int("limit", 20, "Maximum number of syncs to return (1-100)")
	syncListCmd.Flags().String("next-page-token", "", "Opaque token returned by the previous page")

	syncDescribeCmd.Flags().String("id", "", "Private sync (spabs-*) or public publication (sppap-*) ID (required)")
	_ = syncDescribeCmd.MarkFlagRequired("id")

	syncCopyCmd.Flags().String("id", "", "Private MANUAL sync ID (spabs-*) (required)")
	syncCopyCmd.Flags().String("aws-profile", "", "AWS shared-config profile for the destination account")
	syncCopyCmd.Flags().String("vetter", "", "Customer vetting executable; receives one JSON object on stdin per staged artifact")
	syncCopyCmd.Flags().String("signer", "", "Customer signing executable; receives one JSON object on stdin per artifact")
	_ = syncCopyCmd.MarkFlagRequired("id")

	syncRefreshStatusCmd.Flags().String("id", "", "Private MANUAL sync ID (spabs-*) (required)")
	_ = syncRefreshStatusCmd.MarkFlagRequired("id")
	syncReconcileCmd.Flags().String("id", "", "Private MANUAL, FAILED, SKIPPED, or READY sync ID (spabs-*) (required)")
	_ = syncReconcileCmd.MarkFlagRequired("id")
}

func runSyncCopy(cmd *cobra.Command, args []string) error {
	defer config.CleanupArgsAndFlags(cmd, &args)
	syncID, _ := cmd.Flags().GetString("id")
	output, _ := cmd.Flags().GetString(common.OutputFlag)
	awsProfile, _ := cmd.Flags().GetString("aws-profile")
	vetter, _ := cmd.Flags().GetString("vetter")
	signer, _ := cmd.Flags().GetString("signer")
	if err := validateManualSyncID(syncID); err != nil {
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
	if sync.Status != "MANUAL" || sync.ManualCopyPlan == nil {
		return fmt.Errorf("managed artifact sync %s is not ready for manual copy", syncID)
	}
	if err = artifactsync.Run(cmd.Context(), *sync.ManualCopyPlan, artifactsync.Options{AWSProfile: awsProfile, Vetter: vetter, Signer: signer}); err != nil {
		return fmt.Errorf("failed to copy managed artifacts: %w", err)
	}
	if output != common.OutputTypeJson {
		utils.PrintSuccess(fmt.Sprintf("Copied managed artifact sync %s; refreshing control-plane status", syncID))
	}
	verified, err := dataaccess.RefreshManagedArtifactSyncStatus(cmd.Context(), token, syncID)
	if err != nil {
		return fmt.Errorf("artifacts were copied, but status refresh failed; retry with 'omnistrate-ctl managed-artifact sync refresh-status --id %s': %w", syncID, err)
	}
	return printSync(output, *verified)
}

func runSyncRefreshStatus(cmd *cobra.Command, args []string) error {
	defer config.CleanupArgsAndFlags(cmd, &args)
	syncID, _ := cmd.Flags().GetString("id")
	output, _ := cmd.Flags().GetString(common.OutputFlag)
	if err := validateManualSyncID(syncID); err != nil {
		return err
	}
	token, err := common.GetTokenWithLogin()
	if err != nil {
		return fmt.Errorf("failed to get user token: %w", err)
	}
	sync, err := dataaccess.RefreshManagedArtifactSyncStatus(cmd.Context(), token, syncID)
	if err != nil {
		return fmt.Errorf("failed to refresh managed artifact sync status: %w", err)
	}
	return printSync(output, *sync)
}

func runSyncReconcile(cmd *cobra.Command, args []string) error {
	defer config.CleanupArgsAndFlags(cmd, &args)
	syncID, _ := cmd.Flags().GetString("id")
	output, _ := cmd.Flags().GetString(common.OutputFlag)
	if err := validateManualSyncID(syncID); err != nil {
		return err
	}
	token, err := common.GetTokenWithLogin()
	if err != nil {
		return fmt.Errorf("failed to get user token: %w", err)
	}
	sync, err := dataaccess.ReconcileManagedArtifactSync(cmd.Context(), token, syncID)
	if err != nil {
		return fmt.Errorf("failed to reconcile managed artifact sync: %w", err)
	}
	return printSync(output, *sync)
}

func validateManualSyncID(syncID string) error {
	if err := validateSyncID(syncID); err != nil {
		return err
	}
	if !strings.HasPrefix(syncID, "spabs-") {
		return fmt.Errorf("manual copy requires a private spabs-* sync ID")
	}
	return nil
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
	return printSync(output, *sync)
}

func printSync(output string, sync model.ManagedArtifactSync) error {
	if output != "table" {
		return utils.PrintTextTableJsonOutput(output, sync)
	}
	if err := utils.PrintTextTableJsonOutput(output, syncSummary(sync)); err != nil {
		return err
	}
	artifacts := make([]syncArtifactTableRow, 0, len(sync.Artifacts))
	for _, artifact := range sync.Artifacts {
		artifacts = append(artifacts, syncArtifactTableRow{
			AmenityName:     artifact.AmenityName,
			ArtifactKey:     artifact.ArtifactKey,
			Version:         artifact.Version,
			Status:          artifact.Status,
			SourceRef:       artifact.SourceRef,
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
