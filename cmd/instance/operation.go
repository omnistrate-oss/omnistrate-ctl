package instance

import (
	"encoding/json"
	"fmt"
	"math"
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
	operationSourceCustomWorkflow = "CUSTOM_WORKFLOW"
	operationSourceSystemWorkflow = "SYSTEM_WORKFLOW"

	operationVerbBackup         = "BACKUP"
	operationVerbStart          = "START"
	operationVerbStop           = "STOP"
	operationVerbRestart        = "RESTART"
	operationVerbModify         = "MODIFY"
	operationVerbUpdate         = "UPDATE"
	operationVerbDelete         = "DELETE"
	operationVerbFailover       = "FAILOVER"
	operationVerbAddCapacity    = "ADD_CAPACITY"
	operationVerbRemoveCapacity = "REMOVE_CAPACITY"
)

const operationExample = `# List custom operations supported by an instance
omnistrate-ctl instance operation list instance-abcd1234

# Describe a custom operation by operation ID or verb
omnistrate-ctl instance operation describe instance-abcd1234 cwt-12345678
omnistrate-ctl instance operation describe instance-abcd1234 BACKUP

# Trigger a custom operation
omnistrate-ctl instance operation trigger instance-abcd1234 cwt-12345678 --param '{"primaryPodName":"postgres-1"}'

# Trigger a system workflow-backed backup
omnistrate-ctl instance operation trigger instance-abcd1234 BACKUP`

var operationCmd = &cobra.Command{
	Use:          "operation [list|describe|trigger]",
	Short:        "List, describe, and trigger instance custom operations",
	Long:         `List, describe, and trigger custom operations supported by an instance, including system workflow-backed actions returned by the custom operations API.`,
	Example:      operationExample,
	SilenceUsage: true,
}

var operationListCmd = &cobra.Command{
	Use:          "list [instance-id]",
	Short:        "List custom operations supported by an instance",
	RunE:         runOperationList,
	SilenceUsage: true,
}

var operationDescribeCmd = &cobra.Command{
	Use:          "describe [instance-id] [operation-id-or-verb]",
	Short:        "Describe a supported instance custom operation",
	RunE:         runOperationDescribe,
	SilenceUsage: true,
}

var operationTriggerCmd = &cobra.Command{
	Use:          "trigger [instance-id] [operation-id-or-verb]",
	Short:        "Trigger a supported instance custom operation",
	RunE:         runOperationTrigger,
	SilenceUsage: true,
}

type operationTriggerResult struct {
	CustomOperation     openapiclientfleet.ResourceInstanceSupportedOperation            `json:"customOperation"`
	WorkflowExecutionID string                                                           `json:"workflowExecutionId,omitempty"`
	WorkflowID          string                                                           `json:"workflowId,omitempty"`
	Status              *string                                                          `json:"status,omitempty"`
	Backup              *openapiclientfleet.FleetAutomaticInstanceSnapshotCreationResult `json:"backup,omitempty"`
	Message             string                                                           `json:"message,omitempty"`
}

func init() {
	operationListCmd.Args = cobra.ExactArgs(1)
	operationDescribeCmd.Args = cobra.ExactArgs(2)
	operationTriggerCmd.Args = cobra.ExactArgs(2)

	operationTriggerCmd.Flags().String("param", "", "Parameters for the custom operation")
	operationTriggerCmd.Flags().String("param-file", "", "Json file containing parameters for the custom operation")
	operationTriggerCmd.Flags().String("resource-id", "", "Resource ID to pass to the custom operation. Defaults to the instance root resource ID.")
	operationTriggerCmd.Flags().Int64("capacity", 0, "Capacity to add or remove for ADD_CAPACITY and REMOVE_CAPACITY custom operations")
	operationTriggerCmd.Flags().String("failed-replica-id", "", "Failed replica ID for FAILOVER custom operations")
	operationTriggerCmd.Flags().String("failed-replica-action", "", "Failed replica action for FAILOVER custom operations")
	operationTriggerCmd.Flags().BoolP("yes", "y", false, "Pre-approve destructive system workflow-backed custom operations without prompting for confirmation")

	if err := operationTriggerCmd.MarkFlagFilename("param-file"); err != nil {
		return
	}

	operationCmd.AddCommand(operationListCmd)
	operationCmd.AddCommand(operationDescribeCmd)
	operationCmd.AddCommand(operationTriggerCmd)
}

func runOperationList(cmd *cobra.Command, args []string) error {
	defer config.CleanupArgsAndFlags(cmd, &args)

	output, err := cmd.Flags().GetString("output")
	if err != nil {
		utils.PrintError(err)
		return err
	}

	operations, err := loadSupportedOperations(cmd, args[0])
	if err != nil {
		utils.PrintError(err)
		return err
	}

	return utils.PrintTextTableJsonArrayOutput(output, operations)
}

func runOperationDescribe(cmd *cobra.Command, args []string) error {
	defer config.CleanupArgsAndFlags(cmd, &args)

	output, err := cmd.Flags().GetString("output")
	if err != nil {
		utils.PrintError(err)
		return err
	}

	operations, err := loadSupportedOperations(cmd, args[0])
	if err != nil {
		utils.PrintError(err)
		return err
	}
	operation, err := findSupportedOperation(operations, args[1])
	if err != nil {
		utils.PrintError(err)
		return err
	}

	return utils.PrintTextTableJsonOutput(output, operation)
}

func runOperationTrigger(cmd *cobra.Command, args []string) error {
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
	operation, err := findSupportedOperation(instance.ConsumptionResourceInstanceResult.GetSupportedOperations(), selector)
	if err != nil {
		utils.PrintError(err)
		return err
	}

	var sm utils.SpinnerManager
	var spinner *utils.Spinner
	if output != common.OutputTypeJson {
		sm = utils.NewSpinnerManager()
		spinner = sm.AddSpinner("Triggering custom operation...")
		sm.Start()
	}

	result, err := triggerSupportedOperation(cmd, token, serviceID, environmentID, resourceID, instanceID, operation, requestParams)
	if err != nil {
		utils.HandleSpinnerError(spinner, sm, err)
		return err
	}
	if result == nil {
		if sm != nil {
			sm.Stop()
			utils.EnsureCursorRestoration()
		}
		return nil
	}
	utils.HandleSpinnerSuccess(spinner, sm, "Successfully triggered custom operation")

	return utils.PrintTextTableJsonOutput(output, result)
}

func loadSupportedOperations(cmd *cobra.Command, instanceID string) ([]openapiclientfleet.ResourceInstanceSupportedOperation, error) {
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
	return operationOperations(instance.ConsumptionResourceInstanceResult.GetSupportedOperations()), nil
}

func findSupportedOperation(operations []openapiclientfleet.ResourceInstanceSupportedOperation, selector string) (openapiclientfleet.ResourceInstanceSupportedOperation, error) {
	selector = strings.TrimSpace(selector)
	for _, operation := range operations {
		if operation.Id != nil && *operation.Id == selector {
			return operation, nil
		}
		if strings.EqualFold(operation.Verb, selector) {
			return operation, nil
		}
		if strings.EqualFold(operation.Name, selector) {
			return operation, nil
		}
	}
	return openapiclientfleet.ResourceInstanceSupportedOperation{}, fmt.Errorf("custom operation %q not found", selector)
}

func triggerSupportedOperation(
	cmd *cobra.Command,
	token, serviceID, environmentID, resourceID, instanceID string,
	operation openapiclientfleet.ResourceInstanceSupportedOperation,
	requestParams map[string]any,
) (*operationTriggerResult, error) {
	switch strings.ToUpper(operation.Source) {
	case operationSourceCustomWorkflow:
		if operation.Id == nil || *operation.Id == "" {
			return nil, errors.New("custom operation does not include an operation ID")
		}
		customResult, err := dataaccess.ExecuteResourceInstanceCustomWorkflow(cmd.Context(), token, serviceID, environmentID, instanceID, resourceID, *operation.Id, requestParams)
		if err != nil {
			return nil, err
		}
		return &operationTriggerResult{
			CustomOperation:     operation,
			WorkflowExecutionID: customResult.GetWorkflowExecutionId(),
			WorkflowID:          customResult.GetWorkflowId(),
			Status:              customResult.Status,
		}, nil
	case operationSourceSystemWorkflow:
		return triggerSystemOperation(cmd, token, serviceID, environmentID, resourceID, instanceID, operation, requestParams)
	default:
		return nil, fmt.Errorf("custom operation source %q is not supported", operation.Source)
	}
}

func operationOperations(supportedOperations []openapiclientfleet.ResourceInstanceSupportedOperation) []openapiclientfleet.ResourceInstanceSupportedOperation {
	operations := make([]openapiclientfleet.ResourceInstanceSupportedOperation, 0, len(supportedOperations))
	for _, operation := range supportedOperations {
		switch strings.ToUpper(operation.Source) {
		case operationSourceCustomWorkflow, operationSourceSystemWorkflow:
			operations = append(operations, operation)
		}
	}
	return operations
}

func triggerSystemOperation(
	cmd *cobra.Command,
	token, serviceID, environmentID, resourceID, instanceID string,
	operation openapiclientfleet.ResourceInstanceSupportedOperation,
	requestParams map[string]any,
) (*operationTriggerResult, error) {
	result := &operationTriggerResult{CustomOperation: operation}
	switch strings.ToUpper(operation.Verb) {
	case operationVerbBackup:
		backup, err := dataaccess.TriggerResourceInstanceAutoBackup(cmd.Context(), token, serviceID, environmentID, instanceID)
		if err != nil {
			return nil, err
		}
		result.Backup = backup
		result.Message = "backup triggered"
	case operationVerbStart:
		err := dataaccess.StartResourceInstance(cmd.Context(), token, serviceID, environmentID, resourceID, instanceID)
		if err != nil {
			return nil, err
		}
		result.Message = "start triggered"
	case operationVerbStop:
		err := dataaccess.StopResourceInstance(cmd.Context(), token, serviceID, environmentID, resourceID, instanceID)
		if err != nil {
			return nil, err
		}
		result.Message = "stop triggered"
	case operationVerbRestart:
		err := dataaccess.RestartResourceInstance(cmd.Context(), token, serviceID, environmentID, resourceID, instanceID)
		if err != nil {
			return nil, err
		}
		result.Message = "restart triggered"
	case operationVerbModify, operationVerbUpdate:
		if len(requestParams) == 0 {
			return nil, fmt.Errorf("%s requires --param or --param-file", strings.ToUpper(operation.Verb))
		}
		err := dataaccess.UpdateResourceInstance(cmd.Context(), token, serviceID, environmentID, instanceID, resourceID, nil, requestParams, nil)
		if err != nil {
			return nil, err
		}
		result.Message = fmt.Sprintf("%s triggered", strings.ToLower(operation.Verb))
	case operationVerbDelete:
		yes, _ := cmd.Flags().GetBool("yes")
		if !yes {
			confirmed, err := utils.ConfirmAction("Are you sure you want to delete this instance?")
			if err != nil {
				return nil, err
			}
			if !confirmed {
				return nil, nil
			}
		}
		err := dataaccess.DeleteResourceInstance(cmd.Context(), token, serviceID, environmentID, resourceID, instanceID, false)
		if err != nil {
			return nil, err
		}
		result.Message = "delete triggered"
	case operationVerbFailover:
		failedReplicaID, failedReplicaAction, err := failoverParams(cmd, requestParams)
		if err != nil {
			return nil, err
		}
		err = dataaccess.FailoverResourceInstance(cmd.Context(), token, serviceID, environmentID, instanceID, failedReplicaID, failedReplicaAction)
		if err != nil {
			return nil, err
		}
		result.Message = "failover triggered"
	case operationVerbAddCapacity:
		capacity, err := capacityParam(cmd, requestParams, "capacityToBeAdded", operationVerbAddCapacity)
		if err != nil {
			return nil, err
		}
		err = dataaccess.AddCapacityToResourceInstance(cmd.Context(), token, serviceID, environmentID, resourceID, instanceID, capacity)
		if err != nil {
			return nil, err
		}
		result.Message = "add capacity triggered"
	case operationVerbRemoveCapacity:
		capacity, err := capacityParam(cmd, requestParams, "capacityToBeRemoved", operationVerbRemoveCapacity)
		if err != nil {
			return nil, err
		}
		err = dataaccess.RemoveCapacityFromResourceInstance(cmd.Context(), token, serviceID, environmentID, resourceID, instanceID, capacity)
		if err != nil {
			return nil, err
		}
		result.Message = "remove capacity triggered"
	default:
		return nil, fmt.Errorf("system workflow-backed custom operation verb %q is not supported by this command", operation.Verb)
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

func capacityParam(cmd *cobra.Command, requestParams map[string]any, paramKey, operationVerb string) (int64, error) {
	capacity, _ := cmd.Flags().GetInt64("capacity")
	if capacity != 0 {
		return capacity, nil
	}
	value, ok := requestParams[paramKey]
	if !ok {
		value, ok = requestParams["capacity"]
	}
	if !ok {
		return 0, fmt.Errorf("%s requires --capacity or request param %s", operationVerb, paramKey)
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
		if math.Trunc(typed) != typed {
			return 0, fmt.Errorf("capacity value %v is not a whole number", value)
		}
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
