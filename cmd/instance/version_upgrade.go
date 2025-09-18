package instance

import (
	"errors"
	"fmt"
	"os"

	"github.com/chelnak/ysmrr"
	"github.com/omnistrate-oss/omnistrate-ctl/cmd/common"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/config"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/dataaccess"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/utils"
	openapiclientfleet "github.com/omnistrate-oss/omnistrate-sdk-go/fleet"
	errors2 "github.com/pkg/errors"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

const (
	versionUpgradeExample = `# Issue a version upgrade for an instance with upgrade configuration override to the latest tier version
omctl instance version-upgrade instance-abcd1234 --upgrade-configuration-override /path/to/config.yaml

# Issue a version upgrade to a specific target tier version
omctl instance version-upgrade instance-abcd1234 --upgrade-configuration-override /path/to/config.yaml --target-tier-version 3.0

# Issue a version upgrade and wait for completion with progress tracking
omctl instance version-upgrade instance-abcd1234 --upgrade-configuration-override /path/to/config.yaml --target-tier-version 3.0 --wait

# [HELM ONLY] Use generate-configuration with a target tier version to generate a default deployment instance configuration file based on the current helm values as well as the proposed helm values for the target tier version
omctl instance version-upgrade instance-abcd1234 --existing-configuration existing-config.yaml --proposed-configuration proposed-config.yaml --generate-configuration --target-tier-version 3.0

# Example upgrade configuration override YAML file:
# resource-key-1:
#   helmChartValues:
#     key1: value1
#     key2: value2
# resource-key-2:
#   helmChartValues:
#     database:
#       host: new-host
#       port: 5432`
)

var versionUpgradeCmd = &cobra.Command{
	Use:          "version-upgrade [instance-id]",
	Short:        "Issue a version upgrade for a deployment instance",
	Long:         `This command helps you issue a version upgrade for a deployment instance with the specified upgrade configuration override.`,
	Example:      versionUpgradeExample,
	RunE:         runVersionUpgrade,
	SilenceUsage: true,
}

func init() {
	versionUpgradeCmd.Flags().String("upgrade-configuration-override", "", "YAML file containing upgrade configuration override")
	versionUpgradeCmd.Flags().String("existing-configuration", "", "Path to write the existing configuration to (optional, used with --generate-configuration)")
	versionUpgradeCmd.Flags().String("proposed-configuration", "", "Path to write the proposed configuration to (optional, used with --generate-configuration)")
	versionUpgradeCmd.Flags().String("target-tier-version", "", "Target tier version for the version upgrade")
	versionUpgradeCmd.Flags().Bool("generate-configuration", false, "Generate a default configuration file based on current helm values and proposed helm values for the target tier version."+
		"This will not perform an upgrade, but will generate a configuration file that can be used for the upgrade.")
	versionUpgradeCmd.Flags().Bool("wait", false, "Wait for upgrade to complete and show progress")

	versionUpgradeCmd.Args = cobra.ExactArgs(1) // Require exactly one argument (i.e. instance ID)

	var err error
	if err = versionUpgradeCmd.MarkFlagFilename("upgrade-configuration-override"); err != nil {
		return
	}
	if err = versionUpgradeCmd.MarkFlagFilename("existing-configuration"); err != nil {
		return
	}
	if err = versionUpgradeCmd.MarkFlagFilename("proposed-configuration"); err != nil {
		return
	}
}

func runVersionUpgrade(cmd *cobra.Command, args []string) error {
	defer config.CleanupArgsAndFlags(cmd, &args)

	if len(args) == 0 {
		err := errors.New("instance id is required")
		utils.PrintError(err)
		return err
	}

	// Retrieve args
	instanceID := args[0]

	// Retrieve flags
	// Generate configuration override if requested
	generateConfig, err := cmd.Flags().GetBool("generate-configuration")
	if err != nil {
		utils.PrintError(err)
		return err
	}

	targetTierVersion, err := cmd.Flags().GetString("target-tier-version")
	if err != nil {
		utils.PrintError(err)
		return err
	}

	if targetTierVersion == "" {
		utils.PrintError(errors.New("target tier version is required for version upgrade"))
		return errors.New("target tier version is required for version upgrade")
	}

	waitFlag, err := cmd.Flags().GetBool("wait")
	if err != nil {
		utils.PrintError(err)
		return err
	}

	output, err := cmd.Flags().GetString("output")
	if err != nil {
		utils.PrintError(err)
		return err
	}

	// Validate user login
	token, err := common.GetTokenWithLogin()
	if err != nil {
		utils.PrintError(err)
		return err
	}

	// Initialize spinner if output is not JSON
	var sm ysmrr.SpinnerManager
	var spinner *ysmrr.Spinner

	defer func() {
		if spinner != nil {
			spinner.Complete()
		}
		if sm != nil {
			sm.Stop()
		}
	}()

	if output != "json" {
		sm = ysmrr.NewSpinnerManager()
		var msg string
		if !generateConfig {
			msg = fmt.Sprintf("Upgrading deployment instance to target tier version %s", targetTierVersion)
		}
		spinner = sm.AddSpinner(msg)
		sm.Start()
	}

	// Check if instance exists
	if spinner != nil {
		spinner.UpdateMessage("Looking up deployment instance")
	}
	serviceID, environmentID, _, _, err := getInstance(cmd.Context(), token, instanceID)
	if err != nil {
		utils.HandleSpinnerError(spinner, sm, err)
		return err
	}

	// If we have to generate the configuration file, describe the resource instance
	if generateConfig {
		if spinner != nil {
			spinner.UpdateMessage("Generating existing configuration for deployment instance")
		}

		// Retrieve existing and proposed configuration file paths
		existingConfigFile, err := cmd.Flags().GetString("existing-configuration")
		if err != nil {
			utils.PrintError(err)
			return err
		}

		proposedConfigFile, err := cmd.Flags().GetString("proposed-configuration")
		if err != nil {
			utils.PrintError(err)
			return err
		}

		// Validate that both existing and proposed configuration files are provided
		if existingConfigFile == "" || proposedConfigFile == "" {
			err = errors.New("both --existing-configuration and --proposed-configuration must be provided when generating configuration")
			utils.HandleSpinnerError(spinner, sm, err)
			return err
		}

		// Update spinner message
		if spinner != nil {
			spinner.UpdateMessage("Processing instance configuration to generate overrides")
		}
		// Describe instance to get current configuration
		instance, err := dataaccess.DescribeResourceInstance(cmd.Context(), token, serviceID, environmentID, instanceID)
		if err != nil {
			utils.HandleSpinnerError(spinner, sm, err)
			return err
		}

		resourceOverrideConfig := make(map[string]openapiclientfleet.ResourceOneOffPatchConfigurationOverride)
		if instance.ConsumptionResourceInstanceResult.DetailedNetworkTopology == nil ||
			len(*instance.ConsumptionResourceInstanceResult.DetailedNetworkTopology) == 0 {
			utils.HandleSpinnerError(spinner, sm, errors.New("no eligible component topology found for the instance"))
			return errors.New("no eligible component topology found for the instance")
		}

		if spinner != nil {
			spinner.UpdateMessage("Looking up Helm releases for deployment instance to generate existing configuration")
		}

		for _, resourceVersionSummary := range instance.ResourceVersionSummaries {
			// We only support helm overrides for now
			if resourceVersionSummary.HelmDeploymentConfiguration == nil {
				// Skip resources that are not helm deployments
				continue
			}

			resource, ok := (*instance.ConsumptionResourceInstanceResult.DetailedNetworkTopology)[*resourceVersionSummary.ResourceId]

			if !ok {
				// Skip
				continue
			}

			resourceOverrideConfig[resource.ResourceKey] = openapiclientfleet.ResourceOneOffPatchConfigurationOverride{
				HelmChartValues: resourceVersionSummary.HelmDeploymentConfiguration.Values,
			}
		}

		// Write the existing configuration to the specified file
		var marshalledData []byte
		if marshalledData, err = yaml.Marshal(resourceOverrideConfig); err != nil {
			utils.HandleSpinnerError(spinner, sm, errors2.Wrap(err, "failed to marshal existing configuration"))
			return err
		}

		if err = os.WriteFile(existingConfigFile, marshalledData, 0600); err != nil {
			utils.HandleSpinnerError(spinner, sm, errors2.Wrap(err, "failed to write existing configuration to file"))
			return err
		}

		// Write the proposed configuration to the specified file
		if spinner != nil {
			spinner.UpdateMessage("Writing proposed configuration to file")
		}

		proposedConfig := make(map[string]openapiclientfleet.ResourceOneOffPatchConfigurationOverride)

		// Get list of resources in the target tier version
		resources, err := dataaccess.ListResources(cmd.Context(), token, serviceID, instance.ProductTierId, &targetTierVersion)
		if err != nil {
			utils.HandleSpinnerError(spinner, sm, err)
			return err
		}

		for _, resource := range resources.Resources {
			// We only support helm overrides for now
			if resource.HelmChartConfiguration == nil {
				// Skip resources that are not helm deployments
				continue
			}

			resourceKey := resource.Key
			proposedConfig[resourceKey] = openapiclientfleet.ResourceOneOffPatchConfigurationOverride{
				HelmChartValues: resource.HelmChartConfiguration.ChartValues,
			}
		}

		// Write the proposed configuration to the specified file
		if marshalledData, err = yaml.Marshal(proposedConfig); err != nil {
			utils.HandleSpinnerError(spinner, sm, errors2.Wrap(err, "failed to marshal proposed configuration"))
			return err
		}

		if err = os.WriteFile(proposedConfigFile, marshalledData, 0600); err != nil {
			utils.HandleSpinnerError(spinner, sm, errors2.Wrap(err, "failed to write proposed configuration to file"))
			return err
		}

		utils.HandleSpinnerSuccess(spinner, sm, fmt.Sprintf("Saved existing configuration to file: %s; Proposed configuration to file: %s", existingConfigFile, proposedConfigFile))
		if spinner != nil {
			spinner.UpdateMessage("Configuration files generated successfully. You can now use them to perform the upgrade.")
		}

		return nil
	}

	// Read and parse configuration override YAML file
	configOverrideFile, err := cmd.Flags().GetString("upgrade-configuration-override")
	if err != nil {
		utils.PrintError(err)
		return err
	}

	var resourceOverrideConfig map[string]openapiclientfleet.ResourceOneOffPatchConfigurationOverride
	if configOverrideFile != "" {
		configData, err := os.ReadFile(configOverrideFile)
		if err != nil {
			err = errors2.Wrap(err, "failed to read configuration override file")
			utils.HandleSpinnerError(spinner, sm, err)
			return err
		}

		err = yaml.Unmarshal(configData, &resourceOverrideConfig)
		if err != nil {
			err = errors2.Wrap(err, "failed to parse configuration override YAML")
			utils.HandleSpinnerError(spinner, sm, err)
			return err
		}
	}

	// Issue one-off patch
	err = dataaccess.OneOffPatchResourceInstance(cmd.Context(), token,
		serviceID,
		environmentID,
		instanceID,
		resourceOverrideConfig,
		targetTierVersion,
	)
	if err != nil {
		utils.HandleSpinnerError(spinner, sm, err)
		return err
	}

	utils.HandleSpinnerSuccess(spinner, sm, "Successfully initiated version upgrade for deployment instance")

	// Search for the instance to get updated details
	searchRes, err := dataaccess.SearchInventory(cmd.Context(), token, fmt.Sprintf("resourceinstance:%s", instanceID))
	if err != nil {
		utils.PrintError(err)
		return err
	}

	if len(searchRes.ResourceInstanceResults) == 0 {
		err = errors.New("failed to find the upgraded instance")
		utils.PrintError(err)
		return err
	}

	// Format instance
	formattedInstance := formatInstance(&searchRes.ResourceInstanceResults[0], false)

	// Print output
	if err = utils.PrintTextTableJsonOutput(output, formattedInstance); err != nil {
		return err
	}

	// Display workflow resource-wise data if output is not JSON and wait flag is enabled
	if output != "json" && waitFlag {
		fmt.Println("🔄 Deployment progress...")
		err = displayWorkflowResourceDataWithSpinners(cmd.Context(), token, formattedInstance.InstanceID, "upgrade")
		if err != nil {
			// Handle spinner error if deployment monitoring fails
			fmt.Printf("❌ Deployment failed %s\n", err)
		} else {
			fmt.Println("✅ Deployment successful")
		}
	}

	return nil
}
