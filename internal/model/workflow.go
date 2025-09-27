package model

import "strings"

// WorkflowStatus is an enum for workflow status values
//go:generate stringer -type=WorkflowStatus
// (You can run go generate ./... to get WorkflowStatus.String() automatically)
type WorkflowStatus int

const (
	WorkflowStatusUnknown WorkflowStatus = iota
	WorkflowStatusCompleted
	WorkflowStatusFailed
	WorkflowStatusPending
	WorkflowStatusRunning
)

func ParseWorkflowStatus(apiStatus string) WorkflowStatus {
	switch strings.ToLower(apiStatus) {
	case "success":
		return WorkflowStatusCompleted
	case "failed", "error", "cancelled":
		return WorkflowStatusFailed
	case "pending":
		return WorkflowStatusPending
	case "running":
		return WorkflowStatusRunning
	default:
		return WorkflowStatusPending
	}
}
