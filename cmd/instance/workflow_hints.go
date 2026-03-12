package instance

import "fmt"

func PrintWorkflowDebugGuidance(instanceID string, result WorkflowMonitorResult, workflowErr error) {
	if instanceID == "" {
		return
	}

	if workflowErr != nil {
		fmt.Println("🔎 Debug next steps:")
		fmt.Printf("  - Open interactive debug UI: omnistrate-ctl instance debug %s\n", instanceID)

		if cmd := buildWorkflowEventsDebugCommand(result); cmd != "" {
			fmt.Printf("  - Show detailed workflow events: %s\n", cmd)
		}

		if result.FailedResourceName != "" {
			fmt.Printf("  - Failed resource: %s", result.FailedResourceName)
			if result.FailedResourceKey != "" {
				fmt.Printf(" (%s)", result.FailedResourceKey)
			}
			fmt.Println()
		}

		if result.FailedStep != "" {
			fmt.Printf("  - Failed step: %s\n", result.FailedStep)
		}

		if result.FailedReason != "" {
			fmt.Printf("  - Failure reason: %s\n", result.FailedReason)
		}

		return
	}

	fmt.Printf("ℹ️  For step-by-step details, run: omnistrate-ctl instance debug %s\n", instanceID)
}

func buildWorkflowEventsDebugCommand(result WorkflowMonitorResult) string {
	if result.WorkflowID == "" || result.ServiceID == "" || result.EnvironmentID == "" {
		return ""
	}

	cmd := fmt.Sprintf(
		"omnistrate-ctl workflow events %s -s %s -e %s --detail",
		result.WorkflowID,
		result.ServiceID,
		result.EnvironmentID,
	)
	if result.FailedResourceKey != "" {
		cmd = fmt.Sprintf("%s --resource-key %s", cmd, result.FailedResourceKey)
	}

	return cmd
}
