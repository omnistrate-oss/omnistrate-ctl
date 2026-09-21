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

var policyCmd = &cobra.Command{
	Use:          "policy [operation] [flags]",
	Short:        "Manage Base Amenities release policy",
	Run:          run,
	SilenceUsage: true,
}

var policyDescribeCmd = &cobra.Command{
	Use:          "describe [flags]",
	Short:        "Describe a managed artifact release policy",
	Example:      "omnistrate-ctl managed-artifact policy describe --environment-type prod --cloud-provider aws",
	Args:         cobra.NoArgs,
	RunE:         runPolicyDescribe,
	SilenceUsage: true,
}

var policyUpdateCmd = &cobra.Command{
	Use:   "update [flags]",
	Short: "Update a managed artifact release policy",
	Long: `Update automatic managed artifact release adoption for one Base Amenities environment and cloud provider.

Enabling auto-upgrade clears a stored release pin. Disabling auto-upgrade without
--preferred-bundle-version freezes the current effective release.`,
	Example: `omnistrate-ctl managed-artifact policy update --environment-type prod --cloud-provider aws --auto-upgrade=true
omnistrate-ctl managed-artifact policy update --environment-type prod --cloud-provider aws --auto-upgrade=false --preferred-bundle-version r0000020`,
	Args:         cobra.NoArgs,
	RunE:         runPolicyUpdate,
	SilenceUsage: true,
}

func init() {
	policyCmd.AddCommand(policyDescribeCmd)
	policyCmd.AddCommand(policyUpdateCmd)

	addPolicyScopeFlags(policyDescribeCmd)
	addPolicyScopeFlags(policyUpdateCmd)
	policyUpdateCmd.Flags().Bool("auto-upgrade", false, "Automatically adopt the newest published managed artifact release (required)")
	policyUpdateCmd.Flags().String("preferred-bundle-version", "", "Release to pin when auto-upgrade is disabled")
	_ = policyUpdateCmd.MarkFlagRequired("auto-upgrade")
}

func addPolicyScopeFlags(cmd *cobra.Command) {
	cmd.Flags().String("environment-type", "", "Environment type (dev, qa, staging, canary, prod, private, or global) (required)")
	cmd.Flags().String("cloud-provider", "", "Cloud provider (aws, azure, gcp, nebius, oci, byoc-onprem, or all) (required)")
	_ = cmd.MarkFlagRequired("environment-type")
	_ = cmd.MarkFlagRequired("cloud-provider")
}

func policyScope(cmd *cobra.Command) (string, string, error) {
	environmentType, _ := cmd.Flags().GetString("environment-type")
	cloudProvider, _ := cmd.Flags().GetString("cloud-provider")

	normalizedEnvironment, err := normalizeEnvironmentType(environmentType)
	if err != nil {
		return "", "", err
	}
	normalizedCloud, err := normalizeCloudProvider(cloudProvider)
	if err != nil {
		return "", "", err
	}
	return normalizedEnvironment, normalizedCloud, nil
}

func runPolicyDescribe(cmd *cobra.Command, args []string) error {
	defer config.CleanupArgsAndFlags(cmd, &args)

	environmentType, cloudProvider, err := policyScope(cmd)
	if err != nil {
		return err
	}
	output, _ := cmd.Flags().GetString("output")
	token, err := common.GetTokenWithLogin()
	if err != nil {
		return fmt.Errorf("failed to get user token: %w", err)
	}
	policy, err := dataaccess.DescribeManagedArtifactReleasePolicy(cmd.Context(), token, environmentType, cloudProvider)
	if err != nil {
		return fmt.Errorf("failed to describe managed artifact release policy: %w", err)
	}
	return utils.PrintTextTableJsonOutput(output, policy)
}

func runPolicyUpdate(cmd *cobra.Command, args []string) error {
	defer config.CleanupArgsAndFlags(cmd, &args)

	environmentType, cloudProvider, err := policyScope(cmd)
	if err != nil {
		return err
	}
	autoUpgrade, _ := cmd.Flags().GetBool("auto-upgrade")
	preferredBundleVersion, _ := cmd.Flags().GetString("preferred-bundle-version")
	output, _ := cmd.Flags().GetString("output")
	if err := validateBundleVersion(preferredBundleVersion); err != nil {
		return err
	}
	if autoUpgrade && preferredBundleVersion != "" {
		return fmt.Errorf("--preferred-bundle-version cannot be used when --auto-upgrade=true")
	}

	token, err := common.GetTokenWithLogin()
	if err != nil {
		return fmt.Errorf("failed to get user token: %w", err)
	}
	policy, err := dataaccess.UpdateManagedArtifactReleasePolicy(cmd.Context(), token, environmentType, cloudProvider, model.UpdateManagedArtifactReleasePolicyRequest{
		AutoUpgrade:            autoUpgrade,
		PreferredBundleVersion: preferredBundleVersion,
	})
	if err != nil {
		return fmt.Errorf("failed to update managed artifact release policy: %w", err)
	}
	return utils.PrintTextTableJsonOutput(output, policy)
}
