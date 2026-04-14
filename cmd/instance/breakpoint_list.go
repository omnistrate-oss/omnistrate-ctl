package instance

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/omnistrate-oss/omnistrate-ctl/cmd/common"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/config"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/dataaccess"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/utils"
	"github.com/spf13/cobra"
)

type BreakpointListItem struct {
	Key        string `json:"key" table:"Key"`
	ID         string `json:"id" table:"ID"`
	Status     string `json:"status" table:"Status"`
	Conditions string `json:"conditions,omitempty" table:"Conditions"`
}

var breakpointListCmd = &cobra.Command{
	Use:          "list [instance-id]",
	Short:        "List active workflow breakpoints for an instance",
	Long:         "List active workflow breakpoints with status for a specific instance.",
	Args:         cobra.ExactArgs(1),
	RunE:         runBreakpointList,
	SilenceUsage: true,
}

func runBreakpointList(cmd *cobra.Command, args []string) error {
	defer config.CleanupArgsAndFlags(cmd, &args)

	instanceID := args[0]
	output, err := cmd.Flags().GetString("output")
	if err != nil {
		return err
	}

	token, err := common.GetTokenWithLogin()
	if err != nil {
		return err
	}

	serviceID, environmentID, _, _, err := getInstance(cmd.Context(), token, instanceID)
	if err != nil {
		return err
	}

	instance, err := dataaccess.DescribeResourceInstance(cmd.Context(), token, serviceID, environmentID, instanceID)
	if err != nil {
		return err
	}

	activeBreakpoints := instance.GetActiveBreakpoints()
	keyByID, idByKey := buildResourceLookupMaps(instance.GetResourceVersionSummaries())
	formatted := make([]BreakpointListItem, 0, len(activeBreakpoints))
	for _, breakpoint := range activeBreakpoints {
		resourceID, resourceKey := resolveBreakpointResourceIDAndKey(breakpoint.GetId(), keyByID, idByKey)
		item := BreakpointListItem{
			Key:    resourceKey,
			ID:     resourceID,
			Status: breakpoint.GetStatus(),
		}
		if conditions, ok := breakpoint.GetConditionsOk(); ok && len(conditions) > 0 {
			conditionsJSON, marshalErr := json.Marshal(conditions)
			if marshalErr != nil {
				item.Conditions = fmt.Sprintf("%v", conditions)
			} else {
				item.Conditions = string(conditionsJSON)
			}
		}
		formatted = append(formatted, item)
	}

	return utils.PrintTextTableJsonArrayOutput(output, formatted)
}

func resolveBreakpointResourceIDAndKey(idOrKey string, keyByID map[string]string, idByKey map[string]string) (string, string) {
	value := strings.TrimSpace(idOrKey)
	if value == "" {
		return "unknown", "unknown"
	}

	if key, ok := keyByID[value]; ok {
		return value, key
	}

	if id, ok := idByKey[value]; ok {
		return id, value
	}

	return value, value
}
