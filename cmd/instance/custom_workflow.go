package instance

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/omnistrate-oss/omnistrate-ctl/cmd/common"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/config"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/dataaccess"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/utils"
	openapiclientfleet "github.com/omnistrate-oss/omnistrate-sdk-go/fleet"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

const (
	customWorkflowSourceCustomWorkflow = "CUSTOM_WORKFLOW"
	customWorkflowSourceSystemWorkflow = "SYSTEM_WORKFLOW"

	customWorkflowVerbBackup         = "BACKUP"
	customWorkflowVerbStart          = "START"
	customWorkflowVerbStop           = "STOP"
	customWorkflowVerbRestart        = "RESTART"
	customWorkflowVerbModify         = "MODIFY"
	customWorkflowVerbUpdate         = "UPDATE"
	customWorkflowVerbDelete         = "DELETE"
	customWorkflowVerbFailover       = "FAILOVER"
	customWorkflowVerbAddCapacity    = "ADD_CAPACITY"
	customWorkflowVerbRemoveCapacity = "REMOVE_CAPACITY"
)

const customWorkflowExample = `# List custom workflows supported by an instance
omnistrate-ctl instance custom-workflow list instance-abcd1234

# Describe a custom workflow by workflow ID or verb
omnistrate-ctl instance custom-workflow describe instance-abcd1234 cwt-12345678
omnistrate-ctl instance custom-workflow describe instance-abcd1234 BACKUP

# Trigger a custom workflow
omnistrate-ctl instance custom-workflow trigger instance-abcd1234 cwt-12345678 --param '{"primaryPodName":"postgres-1"}'

# Trigger a system workflow-backed backup
omnistrate-ctl instance custom-workflow trigger instance-abcd1234 BACKUP`

var customWorkflowCmd = &cobra.Command{
	Use:          "custom-workflow [list|describe|trigger]",
	Short:        "List, describe, and trigger instance custom workflows",
	Long:         `List, describe, and trigger custom workflows supported by an instance, including system workflow-backed actions returned by the custom workflow API.`,
	Example:      customWorkflowExample,
	SilenceUsage: true,
}

var customWorkflowListCmd = &cobra.Command{
	Use:          "list [instance-id]",
	Short:        "List custom workflows supported by an instance",
	RunE:         runCustomWorkflowList,
	SilenceUsage: true,
}

var customWorkflowDescribeCmd = &cobra.Command{
	Use:          "describe [instance-id] [workflow-id-or-verb]",
	Short:        "Describe a supported instance custom workflow",
	RunE:         runCustomWorkflowDescribe,
	SilenceUsage: true,
}

var customWorkflowTriggerCmd = &cobra.Command{
	Use:          "trigger [instance-id] [workflow-id-or-verb]",
	Short:        "Trigger a supported instance custom workflow",
	RunE:         runCustomWorkflowTrigger,
	SilenceUsage: true,
}

type customWorkflowTriggerResult struct {
	CustomWorkflow      openapiclientfleet.ResourceInstanceSupportedOperation            `json:"customWorkflow"`
	WorkflowExecutionID string                                                           `json:"workflowExecutionId,omitempty"`
	WorkflowID          string                                                           `json:"workflowId,omitempty"`
	Status              *string                                                          `json:"status,omitempty"`
	Backup              *openapiclientfleet.FleetAutomaticInstanceSnapshotCreationResult `json:"backup,omitempty"`
	Message             string                                                           `json:"message,omitempty"`
}

func init() {
	customWorkflowListCmd.Args = cobra.ExactArgs(1)
	customWorkflowDescribeCmd.Args = cobra.ExactArgs(2)
	customWorkflowTriggerCmd.Args = cobra.ExactArgs(2)

	customWorkflowTriggerCmd.Flags().String("param", "", "Parameters for the custom workflow")
	customWorkflowTriggerCmd.Flags().String("param-file", "", "JSON file containing object parameters for the custom workflow")
	customWorkflowTriggerCmd.Flags().String("resource-id", "", "Resource ID to pass to the custom workflow. Defaults to the instance root resource ID.")
	customWorkflowTriggerCmd.Flags().Int64("capacity", 0, "Capacity to add or remove for ADD_CAPACITY and REMOVE_CAPACITY custom workflows")
	customWorkflowTriggerCmd.Flags().String("failed-replica-id", "", "Failed replica ID for FAILOVER custom workflows")
	customWorkflowTriggerCmd.Flags().String("failed-replica-action", "", "Failed replica action for FAILOVER custom workflows")
	customWorkflowTriggerCmd.Flags().BoolP("yes", "y", false, "Pre-approve destructive system workflow-backed custom workflows without prompting for confirmation")

	if err := customWorkflowTriggerCmd.MarkFlagFilename("param-file"); err != nil {
		return
	}

	customWorkflowCmd.AddCommand(customWorkflowListCmd)
	customWorkflowCmd.AddCommand(customWorkflowDescribeCmd)
	customWorkflowCmd.AddCommand(customWorkflowTriggerCmd)
}

func runCustomWorkflowList(cmd *cobra.Command, args []string) error {
	defer config.CleanupArgsAndFlags(cmd, &args)

	output, err := cmd.Flags().GetString("output")
	if err != nil {
		utils.PrintError(err)
		return err
	}

	customWorkflows, err := loadSupportedCustomWorkflows(cmd, args[0])
	if err != nil {
		utils.PrintError(err)
		return err
	}

	return utils.PrintTextTableJsonArrayOutput(output, customWorkflows)
}

func runCustomWorkflowDescribe(cmd *cobra.Command, args []string) error {
	defer config.CleanupArgsAndFlags(cmd, &args)

	output, err := cmd.Flags().GetString("output")
	if err != nil {
		utils.PrintError(err)
		return err
	}

	customWorkflows, err := loadSupportedCustomWorkflows(cmd, args[0])
	if err != nil {
		utils.PrintError(err)
		return err
	}
	customWorkflow, err := findSupportedCustomWorkflow(customWorkflows, args[1])
	if err != nil {
		utils.PrintError(err)
		return err
	}

	return utils.PrintTextTableJsonOutput(output, customWorkflow)
}

func runCustomWorkflowTrigger(cmd *cobra.Command, args []string) error {
	defer config.CleanupArgsAndFlags(cmd, &args)

	output, err := cmd.Flags().GetString("output")
	if err != nil {
		utils.PrintError(err)
		return err
	}
	param, err := cmd.Flags().GetString("param")
	if err != nil {
		utils.PrintError(err)
		return err
	}
	paramFile, err := cmd.Flags().GetString("param-file")
	if err != nil {
		utils.PrintError(err)
		return err
	}
	resourceIDOverride, err := cmd.Flags().GetString("resource-id")
	if err != nil {
		utils.PrintError(err)
		return err
	}

	requestParams, err := common.FormatParams(param, paramFile)
	if err != nil {
		utils.PrintError(err)
		return err
	}

	token, err := common.GetTokenWithLogin()
	if err != nil {
		utils.PrintError(err)
		return err
	}

	instanceID := args[0]
	selector := args[1]
	serviceID, environmentID, _, resourceID, err := getInstance(cmd.Context(), token, instanceID)
	if err != nil {
		utils.PrintError(err)
		return err
	}
	if resourceIDOverride != "" {
		resourceID = resourceIDOverride
	}

	instance, err := dataaccess.DescribeResourceInstance(cmd.Context(), token, serviceID, environmentID, instanceID)
	if err != nil {
		utils.PrintError(err)
		return err
	}
	customWorkflow, err := findSupportedCustomWorkflow(instance.ConsumptionResourceInstanceResult.GetSupportedOperations(), selector)
	if err != nil {
		utils.PrintError(err)
		return err
	}

	var sm utils.SpinnerManager
	var spinner *utils.Spinner
	if output != common.OutputTypeJson {
		sm = utils.NewSpinnerManager()
		spinner = sm.AddSpinner("Triggering custom workflow...")
		sm.Start()
	}

	result, err := triggerSupportedCustomWorkflow(cmd, token, serviceID, environmentID, resourceID, instanceID, customWorkflow, requestParams)
	if err != nil {
		utils.HandleSpinnerError(spinner, sm, err)
		return err
	}
	utils.HandleSpinnerSuccess(spinner, sm, "Successfully triggered custom workflow")

	return utils.PrintTextTableJsonOutput(output, result)
}

func loadSupportedCustomWorkflows(cmd *cobra.Command, instanceID string) ([]openapiclientfleet.ResourceInstanceSupportedOperation, error) {
	token, err := common.GetTokenWithLogin()
	if err != nil {
		return nil, err
	}
	serviceID, environmentID, _, _, err := getInstance(cmd.Context(), token, instanceID)
	if err != nil {
		return nil, err
	}
	instance, err := dataaccess.DescribeResourceInstance(cmd.Context(), token, serviceID, environmentID, instanceID)
	if err != nil {
		return nil, err
	}
	return customWorkflowOperations(instance.ConsumptionResourceInstanceResult.GetSupportedOperations()), nil
}

func findSupportedCustomWorkflow(customWorkflows []openapiclientfleet.ResourceInstanceSupportedOperation, selector string) (openapiclientfleet.ResourceInstanceSupportedOperation, error) {
	selector = strings.TrimSpace(selector)
	for _, customWorkflow := range customWorkflows {
		if customWorkflow.Id != nil && *customWorkflow.Id == selector {
			return customWorkflow, nil
		}
		if strings.EqualFold(customWorkflow.Verb, selector) {
			return customWorkflow, nil
		}
		if strings.EqualFold(customWorkflow.Name, selector) {
			return customWorkflow, nil
		}
	}
	return openapiclientfleet.ResourceInstanceSupportedOperation{}, fmt.Errorf("custom workflow %q not found", selector)
}

func triggerSupportedCustomWorkflow(
	cmd *cobra.Command,
	token, serviceID, environmentID, resourceID, instanceID string,
	customWorkflow openapiclientfleet.ResourceInstanceSupportedOperation,
	requestParams map[string]any,
) (*customWorkflowTriggerResult, error) {
	switch strings.ToUpper(customWorkflow.Source) {
	case customWorkflowSourceCustomWorkflow:
		if customWorkflow.Id == nil || *customWorkflow.Id == "" {
			return nil, errors.New("custom workflow does not include a workflow ID")
		}
		customResult, err := dataaccess.ExecuteResourceInstanceCustomWorkflow(cmd.Context(), token, serviceID, environmentID, instanceID, resourceID, *customWorkflow.Id, requestParams)
		if err != nil {
			return nil, err
		}
		return &customWorkflowTriggerResult{
			CustomWorkflow:      customWorkflow,
			WorkflowExecutionID: customResult.GetWorkflowExecutionId(),
			WorkflowID:          customResult.GetWorkflowId(),
			Status:              customResult.Status,
		}, nil
	case customWorkflowSourceSystemWorkflow:
		return triggerSystemCustomWorkflow(cmd, token, serviceID, environmentID, resourceID, instanceID, customWorkflow, requestParams)
	default:
		return nil, fmt.Errorf("custom workflow source %q is not supported", customWorkflow.Source)
	}
}

func customWorkflowOperations(supportedOperations []openapiclientfleet.ResourceInstanceSupportedOperation) []openapiclientfleet.ResourceInstanceSupportedOperation {
	customWorkflows := make([]openapiclientfleet.ResourceInstanceSupportedOperation, 0, len(supportedOperations))
	for _, operation := range supportedOperations {
		switch strings.ToUpper(operation.Source) {
		case customWorkflowSourceCustomWorkflow, customWorkflowSourceSystemWorkflow:
			customWorkflows = append(customWorkflows, operation)
		}
	}
	return customWorkflows
}

func triggerSystemCustomWorkflow(
	cmd *cobra.Command,
	token, serviceID, environmentID, resourceID, instanceID string,
	customWorkflow openapiclientfleet.ResourceInstanceSupportedOperation,
	requestParams map[string]any,
) (*customWorkflowTriggerResult, error) {
	result := &customWorkflowTriggerResult{CustomWorkflow: customWorkflow}
	switch strings.ToUpper(customWorkflow.Verb) {
	case customWorkflowVerbBackup:
		backup, err := dataaccess.TriggerResourceInstanceAutoBackup(cmd.Context(), token, serviceID, environmentID, instanceID)
		if err != nil {
			return nil, err
		}
		result.Backup = backup
		result.Message = "backup triggered"
	case customWorkflowVerbStart:
		err := dataaccess.StartResourceInstance(cmd.Context(), token, serviceID, environmentID, resourceID, instanceID)
		if err != nil {
			return nil, err
		}
		result.Message = "start triggered"
	case customWorkflowVerbStop:
		err := dataaccess.StopResourceInstance(cmd.Context(), token, serviceID, environmentID, resourceID, instanceID)
		if err != nil {
			return nil, err
		}
		result.Message = "stop triggered"
	case customWorkflowVerbRestart:
		err := dataaccess.RestartResourceInstance(cmd.Context(), token, serviceID, environmentID, resourceID, instanceID)
		if err != nil {
			return nil, err
		}
		result.Message = "restart triggered"
	case customWorkflowVerbModify, customWorkflowVerbUpdate:
		if len(requestParams) == 0 {
			return nil, fmt.Errorf("%s requires --param or --param-file", strings.ToUpper(customWorkflow.Verb))
		}
		err := dataaccess.UpdateResourceInstance(cmd.Context(), token, serviceID, environmentID, instanceID, resourceID, nil, requestParams, nil)
		if err != nil {
			return nil, err
		}
		result.Message = fmt.Sprintf("%s triggered", strings.ToLower(customWorkflow.Verb))
	case customWorkflowVerbDelete:
		yes, _ := cmd.Flags().GetBool("yes")
		if !yes {
			confirmed, err := utils.ConfirmAction("Are you sure you want to delete this instance?")
			if err != nil {
				return nil, err
			}
			if !confirmed {
				return nil, errors.New("delete cancelled")
			}
		}
		err := dataaccess.DeleteResourceInstance(cmd.Context(), token, serviceID, environmentID, resourceID, instanceID)
		if err != nil {
			return nil, err
		}
		result.Message = "delete triggered"
	case customWorkflowVerbFailover:
		failedReplicaID, failedReplicaAction, err := failoverParams(cmd, requestParams)
		if err != nil {
			return nil, err
		}
		err = dataaccess.FailoverResourceInstance(cmd.Context(), token, serviceID, environmentID, instanceID, failedReplicaID, failedReplicaAction)
		if err != nil {
			return nil, err
		}
		result.Message = "failover triggered"
	case customWorkflowVerbAddCapacity:
		capacity, err := capacityParam(cmd, requestParams, "capacityToBeAdded")
		if err != nil {
			return nil, err
		}
		err = dataaccess.AddCapacityToResourceInstance(cmd.Context(), token, serviceID, environmentID, resourceID, instanceID, capacity)
		if err != nil {
			return nil, err
		}
		result.Message = "add capacity triggered"
	case customWorkflowVerbRemoveCapacity:
		capacity, err := capacityParam(cmd, requestParams, "capacityToBeRemoved")
		if err != nil {
			return nil, err
		}
		err = dataaccess.RemoveCapacityFromResourceInstance(cmd.Context(), token, serviceID, environmentID, resourceID, instanceID, capacity)
		if err != nil {
			return nil, err
		}
		result.Message = "remove capacity triggered"
	default:
		return nil, fmt.Errorf("system workflow-backed custom workflow verb %q is not supported by this command", customWorkflow.Verb)
	}
	return result, nil
}

func failoverParams(cmd *cobra.Command, requestParams map[string]any) (failedReplicaID, failedReplicaAction string, err error) {
	failedReplicaID, _ = cmd.Flags().GetString("failed-replica-id")
	failedReplicaAction, _ = cmd.Flags().GetString("failed-replica-action")
	if failedReplicaID == "" {
		failedReplicaID = stringParam(requestParams, "failedReplicaID")
	}
	if failedReplicaAction == "" {
		failedReplicaAction = stringParam(requestParams, "failedReplicaAction")
	}
	if failedReplicaID == "" {
		return "", "", errors.New("FAILOVER requires --failed-replica-id or request param failedReplicaID")
	}
	return failedReplicaID, failedReplicaAction, nil
}

func capacityParam(cmd *cobra.Command, requestParams map[string]any, paramKey string) (int64, error) {
	capacity, _ := cmd.Flags().GetInt64("capacity")
	if capacity != 0 {
		return capacity, nil
	}
	value, ok := requestParams[paramKey]
	if !ok {
		value, ok = requestParams["capacity"]
	}
	if !ok {
		return 0, fmt.Errorf("capacity requires --capacity or request param %s", paramKey)
	}
	return int64Param(value)
}

func stringParam(params map[string]any, key string) string {
	if value, ok := params[key]; ok {
		if str, ok := value.(string); ok {
			return str
		}
	}
	return ""
}

func int64Param(value any) (int64, error) {
	switch typed := value.(type) {
	case float64:
		return int64(typed), nil
	case int:
		return int64(typed), nil
	case int64:
		return typed, nil
	case json.Number:
		return typed.Int64()
	case string:
		return strconv.ParseInt(typed, 10, 64)
	default:
		return 0, fmt.Errorf("capacity value %v is not a number", value)
	}
}
