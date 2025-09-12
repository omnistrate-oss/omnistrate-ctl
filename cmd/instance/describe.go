package instance

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/omnistrate-oss/omnistrate-ctl/cmd/common"
	openapiclientfleet "github.com/omnistrate-oss/omnistrate-sdk-go/fleet"

	"github.com/chelnak/ysmrr"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/config"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/dataaccess"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/utils"
	"github.com/spf13/cobra"
)

const (
	describeExample = `# Describe an instance deployment
omctl instance describe instance-abcd1234`
)

type InstanceStatusType string

var InstanceStatus InstanceStatusType

const (
	InstanceStatusRunning   InstanceStatusType = "RUNNING"
	InstanceStatusStopped   InstanceStatusType = "STOPPED"
	InstanceStatusFailed    InstanceStatusType = "FAILED"
	InstanceStatusCancelled InstanceStatusType = "CANCELLED"
	InstanceStatusUnknown   InstanceStatusType = "UNKNOWN"
)

var describeCmd = &cobra.Command{
	Use:          "describe [instance-id]",
	Short:        "Describe an instance deployment for your service",
	Long:         `This command helps you describe the instance for your service.`,
	Example:      describeExample,
	RunE:         runDescribe,
	SilenceUsage: true,
}

func init() {
	describeCmd.Args = cobra.ExactArgs(1) // Require exactly one argument
	describeCmd.Flags().StringP("output", "o", "json", "Output format. Only json is supported")
	describeCmd.Flags().String("resource-id", "", "Filter results by resource ID")
	describeCmd.Flags().String("resource-key", "", "Filter results by resource key")
}

func runDescribe(cmd *cobra.Command, args []string) error {
	defer config.CleanupArgsAndFlags(cmd, &args)

	// Retrieve args
	instanceID := args[0]

	// Retrieve flags
	output, err := cmd.Flags().GetString("output")
	if err != nil {
		utils.PrintError(err)
		return err
	}

	resourceID, err := cmd.Flags().GetString("resource-id")
	if err != nil {
		utils.PrintError(err)
		return err
	}

	resourceKey, err := cmd.Flags().GetString("resource-key")
	if err != nil {
		utils.PrintError(err)
		return err
	}

	// Validate output flag
	if output != "json" {
		err = errors.New("only json output is supported")
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
	if output != "json" {
		sm = ysmrr.NewSpinnerManager()
		msg := "Describing instance..."
		spinner = sm.AddSpinner(msg)
		sm.Start()
	}

	// Check if instance exists
	serviceID, environmentID, _, _, err := getInstance(cmd.Context(), token, instanceID)
	if err != nil {
		utils.HandleSpinnerError(spinner, sm, err)
		return err
	}

	// Describe instance
	var instance *openapiclientfleet.ResourceInstance
	instance, err = dataaccess.DescribeResourceInstance(cmd.Context(), token, serviceID, environmentID, instanceID)
	if err != nil {
		utils.HandleSpinnerError(spinner, sm, err)
		return err
	}

	utils.HandleSpinnerSuccess(spinner, sm, "Successfully described instance")
	if instance.ConsumptionResourceInstanceResult.Status != nil {
		InstanceStatus = InstanceStatusType(*instance.ConsumptionResourceInstanceResult.Status)
	} else {
		InstanceStatus = InstanceStatusUnknown
	}

	// Apply resource filtering if specified
	if resourceID != "" || resourceKey != "" {
		filteredInstance, err := filterInstanceByResource(cmd.Context(), token, instance, serviceID, environmentID, instanceID, resourceID, resourceKey)
		if err != nil {
			utils.PrintError(fmt.Errorf("failed to apply resource filter: %w", err))
			return err
		}
		instance = filteredInstance
	}

	// Print output
	err = utils.PrintTextTableJsonOutput(output, instance)
	if err != nil {
		utils.PrintError(err)
		return err
	}

	return nil
}

// Helper functions

func getInstance(ctx context.Context, token, instanceID string) (serviceID, environmentID, productTierID, resourceID string, err error) {
	searchRes, err := dataaccess.SearchInventory(ctx, token, fmt.Sprintf("resourceinstance:%s", instanceID))
	if err != nil {
		return
	}

	var found bool
	for _, instance := range searchRes.ResourceInstanceResults {
		if instance.Id == instanceID {
			serviceID = instance.ServiceId
			environmentID = instance.ServiceEnvironmentId
			productTierID = instance.ProductTierId
			if instance.ResourceId != nil {
				resourceID = *instance.ResourceId
			}
			found = true
			break
		}
	}

	if !found {
		err = fmt.Errorf("%s not found. Please check the instance ID and try again", instanceID)
		return
	}

	return
}

func getResourceFromInstance(ctx context.Context, token string, instanceID string, resourceName string) (resourceID, resourceType string, err error) {
	// Check if instance exists
	serviceID, environmentID, _, _, err := getInstance(ctx, token, instanceID)
	if err != nil {
		return
	}

	// Retrieve resource ID
	instanceDes, err := dataaccess.DescribeResourceInstance(ctx, token, serviceID, environmentID, instanceID)
	if err != nil {
		return
	}

	versionSetDes, err := dataaccess.DescribeVersionSet(ctx, token, serviceID, instanceDes.ProductTierId, instanceDes.TierVersion)
	if err != nil {
		return
	}

	for _, resource := range versionSetDes.Resources {
		if resource.Name == resourceName {
			resourceID = resource.Id
			if resource.ManagedResourceType != nil {
				resourceType = strings.ToLower(*resource.ManagedResourceType)
			}
		}
	}

	return
}

func filterInstanceByResource(ctx context.Context, token string, instance *openapiclientfleet.ResourceInstance, serviceID, environmentID, instanceID, resourceID, resourceKey string) (*openapiclientfleet.ResourceInstance, error) {
	// The ResourceInstance has complex nested structure with resource-specific data
	// For filtering, we'll focus on the most relevant parts that contain resource information

	// Make a copy of the instance to avoid modifying the original
	filteredInstance := *instance

	// Get version set to map resource names to IDs if needed
	var keyToID, idToKey map[string]string
	if resourceID != "" || resourceKey != "" {
		versionSetDes, err := dataaccess.DescribeVersionSet(ctx, token, serviceID, instance.ProductTierId, instance.TierVersion)
		if err != nil {
			return instance, err // Return original if we can't get version set
		}

		// Create mapping from resource key to resource ID and vice versa
		keyToID = make(map[string]string)
		idToKey = make(map[string]string)
		for _, resource := range versionSetDes.Resources {
			keyToID[resource.Name] = resource.Id
			idToKey[resource.Id] = resource.Name
		}
	}

	// Filter ResourceVersionSummaries array
	if len(instance.ResourceVersionSummaries) > 0 {
		var filteredSummaries []openapiclientfleet.ResourceVersionSummary

		for _, summary := range instance.ResourceVersionSummaries {
			includeResource := false

			if resourceID != "" && summary.ResourceId != nil {
				// If filtering by resource ID, check if it matches
				if *summary.ResourceId == resourceID {
					includeResource = true
				}
			}

			if resourceKey != "" && summary.ResourceName != nil {
				// If filtering by resource key, check if it matches resource name
				if *summary.ResourceName == resourceKey {
					includeResource = true
				}
			}

			// Also check cross-mapping: if filtering by resource ID but summary has resource name, check mapping
			if resourceID != "" && summary.ResourceName != nil && !includeResource {
				if keyToID[*summary.ResourceName] == resourceID {
					includeResource = true
				}
			}

			// Also check cross-mapping: if filtering by resource key but summary has resource ID, check mapping
			if resourceKey != "" && summary.ResourceId != nil && !includeResource {
				if idToKey[*summary.ResourceId] == resourceKey {
					includeResource = true
				}
			}

			if includeResource {
				filteredSummaries = append(filteredSummaries, summary)
			}
		}

		filteredInstance.ResourceVersionSummaries = filteredSummaries
	}

	// Filter DetailedNetworkTopology if present - this contains resource-specific network information
	if instance.ConsumptionResourceInstanceResult.DetailedNetworkTopology != nil {
		filteredTopology := make(map[string]interface{})

		// Filter the topology based on resourceID or resourceKey
		for key, value := range instance.ConsumptionResourceInstanceResult.DetailedNetworkTopology {
			includeResource := false

			if resourceID != "" {
				// If filtering by resource ID, check if key matches resource ID or if key is resource name that maps to resource ID
				if key == resourceID || keyToID[key] == resourceID {
					includeResource = true
				}
			}

			if resourceKey != "" {
				// If filtering by resource key, check if key matches resource key or if key is resource ID that maps to resource key
				if key == resourceKey || idToKey[key] == resourceKey {
					includeResource = true
				}
			}

			if includeResource {
				filteredTopology[key] = value
			}
		}

		filteredInstance.ConsumptionResourceInstanceResult.DetailedNetworkTopology = filteredTopology
	}

	// Add filtered resource information as additional metadata
	if filteredInstance.AdditionalProperties == nil {
		filteredInstance.AdditionalProperties = make(map[string]interface{})
	}

	// Add filter information to the response
	filterInfo := map[string]interface{}{}
	if resourceID != "" {
		filterInfo["resourceId"] = resourceID
	}
	if resourceKey != "" {
		filterInfo["resourceKey"] = resourceKey
	}
	filteredInstance.AdditionalProperties["appliedFilters"] = filterInfo

	// Add count information for filtered resources
	countInfo := map[string]interface{}{
		"totalResourceVersionSummaries":    len(instance.ResourceVersionSummaries),
		"filteredResourceVersionSummaries": len(filteredInstance.ResourceVersionSummaries),
	}
	if instance.ConsumptionResourceInstanceResult.DetailedNetworkTopology != nil {
		countInfo["totalNetworkTopologyEntries"] = len(instance.ConsumptionResourceInstanceResult.DetailedNetworkTopology)
		countInfo["filteredNetworkTopologyEntries"] = len(filteredInstance.ConsumptionResourceInstanceResult.DetailedNetworkTopology)
	}
	filteredInstance.AdditionalProperties["filteringStats"] = countInfo

	return &filteredInstance, nil
}
