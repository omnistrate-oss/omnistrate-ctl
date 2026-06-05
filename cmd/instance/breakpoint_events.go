package instance

import (
	"fmt"
	"reflect"
	"strings"

	openapiclientfleet "github.com/omnistrate-oss/omnistrate-sdk-go/fleet"
)

const workflowBreakpointEventsProperty = "events"
const workflowBreakpointEventProperty = "event"

var supportedWorkflowBreakpointEvents = map[string]struct{}{
	"StartHelmInstall":       {},
	"CompleteHelmInstall":    {},
	"FailHelmInstall":        {},
	"StartTerraformPlan":     {},
	"CompleteTerraformPlan":  {},
	"FailTerraformPlan":      {},
	"StartTerraformApply":    {},
	"CompleteTerraformApply": {},
	"FailTerraformApply":     {},
}

func normalizeWorkflowBreakpointEvents(rawEvents []string) ([]string, error) {
	if len(rawEvents) == 0 {
		return nil, nil
	}

	events := make([]string, 0, len(rawEvents))
	seen := make(map[string]struct{}, len(rawEvents))
	for _, rawEvent := range rawEvents {
		event := strings.TrimSpace(rawEvent)
		if event == "" {
			return nil, fmt.Errorf("breakpoint event cannot be empty")
		}
		if _, ok := supportedWorkflowBreakpointEvents[event]; !ok {
			return nil, fmt.Errorf("unsupported breakpoint event %q", event)
		}
		if _, ok := seen[event]; ok {
			continue
		}

		seen[event] = struct{}{}
		events = append(events, event)
	}

	return events, nil
}

func setWorkflowBreakpointEvents(breakpoint *openapiclientfleet.WorkflowBreakpoint, events []string) {
	if len(events) == 0 {
		return
	}
	if breakpoint.AdditionalProperties == nil {
		breakpoint.AdditionalProperties = make(map[string]interface{})
	}
	breakpoint.AdditionalProperties[workflowBreakpointEventsProperty] = events
}

func workflowBreakpointStatusEvent(breakpoint openapiclientfleet.WorkflowBreakpointWithStatus) string {
	value := reflect.ValueOf(breakpoint)
	if value.Kind() == reflect.Struct {
		eventField := value.FieldByName("Event")
		if eventField.IsValid() && eventField.Kind() == reflect.Ptr && !eventField.IsNil() && eventField.Elem().Kind() == reflect.String {
			return strings.TrimSpace(eventField.Elem().String())
		}
	}

	if breakpoint.AdditionalProperties == nil {
		return ""
	}
	rawEvent, ok := breakpoint.AdditionalProperties[workflowBreakpointEventProperty]
	if !ok || rawEvent == nil {
		return ""
	}
	if event, ok := rawEvent.(string); ok {
		return strings.TrimSpace(event)
	}
	return fmt.Sprintf("%v", rawEvent)
}
