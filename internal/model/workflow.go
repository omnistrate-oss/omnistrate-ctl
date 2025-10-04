package model

import "strings"

// WorkflowStepEventType represents the type of a workflow step event
type WorkflowStepEventType string

const (
	WorkflowStepFailed    WorkflowStepEventType = "WorkflowStepFailed"
	WorkflowStepCompleted WorkflowStepEventType = "WorkflowStepCompleted"
	WorkflowStepDebug     WorkflowStepEventType = "WorkflowStepDebug"
	WorkflowStepStarted   WorkflowStepEventType = "WorkflowStepStarted"
	WorkflowStepPending   WorkflowStepEventType = "WorkflowStepPending"
	WorkflowStepUnknown   WorkflowStepEventType = "WorkflowStepUnknown"
)



// ResourceStatus represents the status of a resource as a string
type ResourceStatus string

const (
   ResourceStatusUnknown   ResourceStatus = "UNKNOWN"
   ResourceStatusCompleted ResourceStatus = "COMPLETED"
   ResourceStatusFailed    ResourceStatus = "FAILED"
   ResourceStatusPending   ResourceStatus = "PENDING"
   ResourceStatusRunning   ResourceStatus = "RUNNING"
)

func (v ResourceStatus) String() string {
   return string(v)
}

// WorkflowStep represents a workflow step as a string
type WorkflowStep string

const (
   WorkflowStepBootstrap   WorkflowStep = "BOOTSTRAP"
   WorkflowStepStorage     WorkflowStep = "STORAGE"
   WorkflowStepNetwork     WorkflowStep = "NETWORK"
   WorkflowStepCompute     WorkflowStep = "COMPUTE"
   WorkflowStepDeployment  WorkflowStep = "DEPLOYMENT"
   WorkflowStepMonitoring  WorkflowStep = "MONITORING"
   WorkflowStepUnknownStep WorkflowStep = "UNKNOWN"
)

func (v WorkflowStep) String() string {
   return string(v)
}

// WorkflowStatus represents the status of a workflow as a string (for consistency with UpgradePathStatus)
type WorkflowStatus string

const (
   WorkflowStatusUnknown   WorkflowStatus = "UNKNOWN"
   WorkflowStatusCompleted WorkflowStatus = "COMPLETED"
   WorkflowStatusFailed    WorkflowStatus = "FAILED"
   WorkflowStatusPending   WorkflowStatus = "PENDING"
   WorkflowStatusRunning   WorkflowStatus = "RUNNING"
)

func (v WorkflowStatus) String() string {
   return string(v)
}

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
	   return WorkflowStatusUnknown
   }
}
