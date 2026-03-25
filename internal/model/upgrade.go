package model

// UpgradePathStatus represents the status of an upgrade path
type UpgradePathStatus string

func (v UpgradePathStatus) String() string {
	return string(v)
}

const (
	InProgress UpgradePathStatus = "IN_PROGRESS"
	Scheduled  UpgradePathStatus = "SCHEDULED"
	Complete   UpgradePathStatus = "COMPLETE"
	Failed     UpgradePathStatus = "FAILED"
	Cancelled  UpgradePathStatus = "CANCELLED"
	Skipped    UpgradePathStatus = "SKIPPED"
	Verifying  UpgradePathStatus = "VERIFYING"
)

type UpgradeStatus struct {
	UpgradeID      string `json:"upgrade_id"`
	Total          int64  `json:"total"`
	Pending        int64  `json:"pending"`
	InProgress     int64  `json:"in_progress"`
	Completed      int64  `json:"completed"`
	Failed         int64  `json:"failed"`
	Scheduled      *int   `json:"scheduled,omitempty"`
	Skipped        int64  `json:"skipped"`
	Status         string `json:"status"`
	NotifyCustomer bool   `json:"notify_customer"`
}

type Upgrade struct {
	UpgradeID             string  `json:"upgrade_id"`
	SourceVersion         string  `json:"source_version"`
	TargetVersion         string  `json:"target_version"`
	InstanceIDs           string  `json:"instance_ids"`
	ScheduledDate         *string `json:"scheduled_date,omitempty"`
	NotifyCustomer        bool    `json:"notify_customer"`
	MaxConcurrentUpgrades *int    `json:"max_concurrent_upgrades,omitempty"`
}

type UpgradeStatusDetail struct {
	UpgradeID            string  `json:"upgrade_id"`
	InstanceID           string  `json:"instance_id"`
	UpgradeStartTime     string  `json:"upgrade_start_time"`
	PlannedExecutionDate *string `json:"planned_execution_date"`
	UpgradeEndTime       string  `json:"upgrade_end_time"`
	UpgradeStatus        string  `json:"upgrade_status"`
}

type UpgradeMaintenanceAction string

func (a UpgradeMaintenanceAction) String() string {
	return string(a)
}

// Maintenance actions
const (
	PauseAction          UpgradeMaintenanceAction = "pause"
	ResumeAction         UpgradeMaintenanceAction = "resume"
	CancelAction         UpgradeMaintenanceAction = "cancel"
	NotifyCustomerAction UpgradeMaintenanceAction = "notify-customer"
	SkipInstancesAction  UpgradeMaintenanceAction = "skip-instances"
)

// Maintenance event types
const (
	EventTypeScheduled = "scheduled"
	EventTypeReminder  = "reminder"
	EventTypeImmediate = "immediate"
)

// Completion status types
const (
	CompletionStatusSuccess   = "success"
	CompletionStatusCancelled = "cancelled"
	CompletionStatusSkipped   = "skipped"
)
