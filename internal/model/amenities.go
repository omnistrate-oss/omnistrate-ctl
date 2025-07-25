package model

import "time"

// AmenitiesConfiguration represents the organization-level amenities configuration
type AmenitiesConfiguration struct {
	// Organization identifier
	OrganizationID string `json:"organization_id"`
	
	// Environment this configuration applies to
	Environment string `json:"environment"`
	
	// Configuration template data
	ConfigurationTemplate map[string]interface{} `json:"configuration_template"`
	
	// Metadata
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Version   string    `json:"version"`
}

// DeploymentCellAmenitiesStatus represents the status of amenities configuration for a deployment cell
type DeploymentCellAmenitiesStatus struct {
	// Deployment cell identifier
	DeploymentCellID string `json:"deployment_cell_id"`
	
	// Current configuration
	CurrentConfiguration map[string]interface{} `json:"current_configuration,omitempty"`
	
	// Target configuration from organization template
	TargetConfiguration map[string]interface{} `json:"target_configuration,omitempty"`
	
	// Drift detection results
	HasConfigurationDrift bool                   `json:"has_configuration_drift"`
	DriftDetails          []ConfigurationDrift   `json:"drift_details,omitempty"`
	
	// Pending changes information
	HasPendingChanges bool                      `json:"has_pending_changes"`
	PendingChanges    []PendingConfigurationChange `json:"pending_changes,omitempty"`
	
	// Status and timestamps
	Status    string    `json:"status"`
	LastCheck time.Time `json:"last_check"`
}

// ConfigurationDrift represents a specific configuration drift
type ConfigurationDrift struct {
	Path         string      `json:"path"`         // JSON path to the drifted configuration
	CurrentValue interface{} `json:"current_value"` // Current value in deployment cell
	TargetValue  interface{} `json:"target_value"`  // Expected value from organization template
	DriftType    string      `json:"drift_type"`    // Type of drift: "missing", "extra", "different"
}

// PendingConfigurationChange represents a pending change to be applied
type PendingConfigurationChange struct {
	Path      string      `json:"path"`       // JSON path of the configuration
	Operation string      `json:"operation"`  // Operation type: "add", "update", "delete"
	OldValue  interface{} `json:"old_value,omitempty"`  // Current value (for update/delete)
	NewValue  interface{} `json:"new_value,omitempty"`  // New value (for add/update)
}

// AmenitiesEnvironment represents an environment configuration context
type AmenitiesEnvironment struct {
	Name        string `json:"name"`
	DisplayName string `json:"display_name"`
	Description string `json:"description"`
}

// AmenitiesConfigurationTableView provides a simplified view for table display
type AmenitiesConfigurationTableView struct {
	OrganizationID string    `json:"organization_id"`
	Environment    string    `json:"environment"`
	Version        string    `json:"version"`
	UpdatedAt      time.Time `json:"updated_at"`
	ConfigCount    int       `json:"config_count"` // Number of configuration items
}

// ToTableView converts AmenitiesConfiguration to table-friendly view
func (ac AmenitiesConfiguration) ToTableView() AmenitiesConfigurationTableView {
	configCount := len(ac.ConfigurationTemplate)
	return AmenitiesConfigurationTableView{
		OrganizationID: ac.OrganizationID,
		Environment:    ac.Environment,
		Version:        ac.Version,
		UpdatedAt:      ac.UpdatedAt,
		ConfigCount:    configCount,
	}
}

// DeploymentCellAmenitiesTableView provides a simplified view for table display
type DeploymentCellAmenitiesTableView struct {
	DeploymentCellID      string    `json:"deployment_cell_id"`
	Status                string    `json:"status"`
	HasConfigurationDrift bool      `json:"has_configuration_drift"`
	HasPendingChanges     bool      `json:"has_pending_changes"`
	DriftCount            int       `json:"drift_count"`
	PendingChangesCount   int       `json:"pending_changes_count"`
	LastCheck             time.Time `json:"last_check"`
}

// ToTableView converts DeploymentCellAmenitiesStatus to table-friendly view
func (dcas DeploymentCellAmenitiesStatus) ToTableView() DeploymentCellAmenitiesTableView {
	return DeploymentCellAmenitiesTableView{
		DeploymentCellID:      dcas.DeploymentCellID,
		Status:                dcas.Status,
		HasConfigurationDrift: dcas.HasConfigurationDrift,
		HasPendingChanges:     dcas.HasPendingChanges,
		DriftCount:            len(dcas.DriftDetails),
		PendingChangesCount:   len(dcas.PendingChanges),
		LastCheck:             dcas.LastCheck,
	}
}