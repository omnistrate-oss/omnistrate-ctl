package instance

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
	"github.com/omnistrate-oss/omnistrate-ctl/cmd/common"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/config"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/dataaccess"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/utils"
	openapiclientfleet "github.com/omnistrate-oss/omnistrate-sdk-go/fleet"
	"github.com/spf13/cobra"
)

var breakpointResumeCmd = &cobra.Command{
	Use:          "resume [instance-id]",
	Short:        "Resume a paused workflow for an instance breakpoint",
	Long:         "Resume the currently paused workflow for an instance and continue from the active breakpoint.",
	Args:         cobra.ExactArgs(1),
	RunE:         runBreakpointResume,
	SilenceUsage: true,
}

func init() {
	breakpointResumeCmd.Flags().BoolP("force", "f", false, "Skip confirmation prompt and resume immediately")
}

func runBreakpointResume(cmd *cobra.Command, args []string) error {
	defer config.CleanupArgsAndFlags(cmd, &args)

	instanceID := args[0]
	force, err := cmd.Flags().GetBool("force")
	if err != nil {
		return err
	}
	outputFormat, err := cmd.Flags().GetString("output")
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

	workflows, err := dataaccess.ListWorkflows(cmd.Context(), token, serviceID, environmentID, &dataaccess.ListWorkflowsOptions{
		InstanceID: instanceID,
	})
	if err != nil {
		return err
	}

	pausedWorkflow, err := selectLatestPausedWorkflow(workflows)
	if err != nil {
		return err
	}

	if !force {
		breakpointSummary := summarizeBreakpointForResume(
			instance.GetActiveBreakpoints(),
			instance.GetResourceVersionSummaries(),
		)
		confirmationMessage := fmt.Sprintf(
			"Resume workflow %s for instance %s?\nBreakpoint to resume: %s",
			pausedWorkflow.Id,
			instanceID,
			breakpointSummary,
		)
		confirmed, confirmErr := confirmBreakpointResumeAction(confirmationMessage)
		if confirmErr != nil {
			return confirmErr
		}
		if !confirmed {
			return nil
		}
	}

	result, err := dataaccess.ResumeWorkflow(cmd.Context(), token, serviceID, environmentID, pausedWorkflow.Id)
	if err != nil {
		return err
	}

	if result == nil {
		fmt.Printf("Workflow %s resumed successfully\n", pausedWorkflow.Id)
		return nil
	}

	return utils.PrintTextTableJsonArrayOutput(outputFormat, []any{result})
}

func selectLatestPausedWorkflow(workflows *openapiclientfleet.ListServiceWorkflowsResult) (*openapiclientfleet.ServiceWorkflow, error) {
	if workflows == nil || len(workflows.Workflows) == 0 {
		return nil, fmt.Errorf("no workflows found for this instance")
	}

	var latest *openapiclientfleet.ServiceWorkflow
	var latestTime time.Time
	for i := range workflows.Workflows {
		workflow := workflows.Workflows[i]
		if !isPausedWorkflowStatus(workflow.Status) {
			continue
		}

		startTime, parseErr := parseWorkflowStartTime(workflow.StartTime)
		if latest == nil {
			latest = &workflow
			latestTime = startTime
			if parseErr != nil {
				latestTime = time.Time{}
			}
			continue
		}

		if parseErr == nil && (latestTime.IsZero() || startTime.After(latestTime)) {
			latest = &workflow
			latestTime = startTime
			continue
		}

		if parseErr != nil && (latestTime.IsZero() && workflow.StartTime > latest.StartTime) {
			latest = &workflow
		}
	}

	if latest == nil {
		return nil, fmt.Errorf("no paused workflow found for instance")
	}
	return latest, nil
}

func isPausedWorkflowStatus(status string) bool {
	return strings.EqualFold(status, "pause") || strings.EqualFold(status, "paused")
}

func parseWorkflowStartTime(raw string) (time.Time, error) {
	if raw == "" {
		return time.Time{}, fmt.Errorf("empty start time")
	}

	parsed, err := time.Parse(time.RFC3339Nano, raw)
	if err == nil {
		return parsed, nil
	}

	parsed, err = time.Parse(time.RFC3339, raw)
	if err == nil {
		return parsed, nil
	}

	return time.Time{}, err
}

func summarizeBreakpointForResume(
	activeBreakpoints []openapiclientfleet.WorkflowBreakpointWithStatus,
	resourceSummaries []openapiclientfleet.ResourceVersionSummary,
) string {
	if len(activeBreakpoints) == 0 {
		return "unknown"
	}

	keyByID, idByKey := buildResourceLookupMaps(resourceSummaries)
	formatRef := func(idOrKey string) string {
		return formatBreakpointResourceRef(idOrKey, keyByID, idByKey)
	}

	hitIDs := make([]string, 0)
	pendingIDs := make([]string, 0)
	for _, breakpoint := range activeBreakpoints {
		id := formatRef(breakpoint.GetId())
		status := breakpoint.GetStatus()
		if strings.EqualFold(status, "hit") {
			hitIDs = append(hitIDs, id)
			continue
		}
		if strings.EqualFold(status, "pending") {
			pendingIDs = append(pendingIDs, id)
		}
	}

	if len(hitIDs) == 1 {
		return hitIDs[0]
	}
	if len(hitIDs) > 1 {
		return fmt.Sprintf("%s (multiple hit breakpoints)", strings.Join(hitIDs, ", "))
	}
	if len(pendingIDs) == 1 {
		return pendingIDs[0]
	}
	if len(pendingIDs) > 1 {
		return fmt.Sprintf("%s (all pending)", strings.Join(pendingIDs, ", "))
	}

	ids := make([]string, 0, len(activeBreakpoints))
	for _, breakpoint := range activeBreakpoints {
		ids = append(ids, formatRef(breakpoint.GetId()))
	}
	return strings.Join(ids, ", ")
}

func buildResourceLookupMaps(resourceSummaries []openapiclientfleet.ResourceVersionSummary) (map[string]string, map[string]string) {
	keyByID := make(map[string]string)
	idByKey := make(map[string]string)

	for _, summary := range resourceSummaries {
		key := strings.TrimSpace(summary.GetResourceName())
		id := strings.TrimSpace(summary.GetResourceId())
		if key == "" || id == "" {
			continue
		}

		keyByID[id] = key
		idByKey[key] = id
	}

	return keyByID, idByKey
}

func formatBreakpointResourceRef(idOrKey string, keyByID map[string]string, idByKey map[string]string) string {
	value := strings.TrimSpace(idOrKey)
	if value == "" {
		return "unknown [unknown]"
	}

	if key, ok := keyByID[value]; ok {
		return fmt.Sprintf("%s [%s]", key, value)
	}

	if id, ok := idByKey[value]; ok {
		return fmt.Sprintf("%s [%s]", value, id)
	}

	return fmt.Sprintf("%s [%s]", value, value)
}

func confirmBreakpointResumeAction(message string) (bool, error) {
	var confirmed bool

	theme := huh.ThemeCharm()
	green := lipgloss.AdaptiveColor{Light: "#067D43", Dark: "#02BF87"}
	theme.Focused.Title = theme.Focused.Title.Foreground(green).Bold(true)
	theme.Group.Title = theme.Focused.Title

	err := huh.NewConfirm().
		Title(message).
		Affirmative("Yes").
		Negative("No").
		Value(&confirmed).
		WithTheme(theme).
		Run()
	if err != nil {
		return false, err
	}

	return confirmed, nil
}
