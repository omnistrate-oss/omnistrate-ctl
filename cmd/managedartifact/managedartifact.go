package managedartifact

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

var bundleVersionPattern = regexp.MustCompile(`^r[0-9]{7,}$`)
var syncIDPattern = regexp.MustCompile(`^spabs-[a-zA-Z0-9-]+$`)

// Cmd is the parent command for managed artifact operations.
var Cmd = &cobra.Command{
	Use:          "managed-artifact [operation] [flags]",
	Short:        "Manage Base Amenities artifact releases",
	Long:         "Inspect managed artifact releases and provisioner synchronization, and manage Base Amenities release policy.",
	Run:          run,
	SilenceUsage: true,
}

func init() {
	Cmd.AddCommand(releaseCmd)
	Cmd.AddCommand(policyCmd)
	Cmd.AddCommand(syncCmd)
}

func run(cmd *cobra.Command, _ []string) {
	_ = cmd.Help()
}

func normalizeEnvironmentType(value string) (string, error) {
	normalized := strings.ToUpper(strings.TrimSpace(value))
	switch normalized {
	case "DEV", "QA", "STAGING", "CANARY", "PROD", "PRIVATE", "GLOBAL":
		return normalized, nil
	default:
		return "", fmt.Errorf("invalid environment type %q: expected dev, qa, staging, canary, prod, private, or global", value)
	}
}

func normalizeCloudProvider(value string) (string, error) {
	normalized := strings.ToLower(strings.TrimSpace(value))
	switch normalized {
	case "aws", "azure", "gcp", "nebius", "oci", "byoc-onprem", "all":
		return normalized, nil
	default:
		return "", fmt.Errorf("invalid cloud provider %q: expected aws, azure, gcp, nebius, oci, byoc-onprem, or all", value)
	}
}

func normalizeSyncStatus(value string) (string, error) {
	if strings.TrimSpace(value) == "" {
		return "", nil
	}
	normalized := strings.ToUpper(strings.TrimSpace(value))
	switch normalized {
	case "PENDING", "IN_PROGRESS", "READY", "FAILED", "SKIPPED":
		return normalized, nil
	default:
		return "", fmt.Errorf("invalid sync status %q: expected pending, in_progress, ready, failed, or skipped", value)
	}
}

func validateBundleVersion(value string) error {
	if value != "" && !bundleVersionPattern.MatchString(value) {
		return fmt.Errorf("invalid bundle version %q: expected r followed by at least seven digits", value)
	}
	return nil
}

func validateSyncID(value string) error {
	if !syncIDPattern.MatchString(value) {
		return fmt.Errorf("invalid sync ID %q: expected a value beginning with spabs-", value)
	}
	return nil
}

func validateRFC3339Flag(name, value string) error {
	if value == "" {
		return nil
	}
	if _, err := time.Parse(time.RFC3339, value); err != nil {
		return fmt.Errorf("invalid %s %q: expected RFC3339 format: %w", name, value, err)
	}
	return nil
}

func validateLimit(limit int) error {
	if limit < 1 || limit > 100 {
		return fmt.Errorf("limit must be between 1 and 100")
	}
	return nil
}
