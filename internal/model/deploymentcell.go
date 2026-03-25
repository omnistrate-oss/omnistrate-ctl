package model

import (
	"fmt"
	"strings"
	"time"
)

// DeploymentCellTemplate represents a template structure for API responses
type DeploymentCellTemplate struct {
	ManagedAmenities []Amenity `json:"managed_amenities,omitempty" yaml:"managedAmenities,omitempty"`
	CustomAmenities  []Amenity `json:"custom_amenities,omitempty" yaml:"customAmenities,omitempty"`
}

// DeploymentCell represents a complete view of a host cluster/deployment cell
type DeploymentCell struct {
	// Basic identification
	ID          string `json:"id"`
	Key         string `json:"key,omitempty"` // Optional key for easier identification
	Status      string `json:"status"`
	Type        string `json:"type"`
	Description string `json:"description"`

	// Infrastructure details
	CloudProvider      string `json:"cloud_provider"`
	Region             string `json:"region"`
	RegionID           string `json:"region_id"`
	AccountID          string `json:"account_id"`
	AccountConfigID    string `json:"account_config_id"`
	IsCustomDeployment bool   `json:"is_custom_deployment"`

	// Deployment information
	CurrentNumberOfDeployments int64 `json:"current_number_of_deployments"`

	// Health status summary
	HealthStatus DeploymentCellHealthStatus `json:"health_status"`

	// Network configuration
	CustomNetwork *CustomNetworkInfo `json:"custom_network,omitempty"`

	// Kubernetes details
	KubernetesDashboardEndpoint *string `json:"kubernetes_dashboard_endpoint,omitempty"`

	// Helm packages
	HelmPackages []HelmPackageInfo `json:"helm_packages,omitempty"`

	Amenities []Amenity `json:"amenities,omitempty"`

	InSyncWithTemplate bool `json:"in_sync_with_template,omitempty"`

	// Additional metadata
	Role      *string `json:"role,omitempty"`
	ModelType *string `json:"model_type,omitempty"`

	// Customer metadata
	CustomerEmail            *string `json:"customer_email,omitempty"`
	CustomerOrganizationName *string `json:"customer_organization_name,omitempty"`
}

// DeploymentCellHealthStatus represents the health status of a deployment cell
type DeploymentCellHealthStatus struct {
	OverallStatus                 string             `json:"overall_status"`
	StatusMessage                 *string            `json:"status_message,omitempty"`
	TotalEntities                 int64              `json:"total_entities"`
	HealthyEntities               int64              `json:"healthy_entities"`
	FailedEntities                int64              `json:"failed_entities"`
	EntitiesByType                map[string]int64   `json:"entities_by_type,omitempty"`
	HealthyEntitiesByType         map[string]int64   `json:"healthy_entities_by_type,omitempty"`
	FailedEntitiesByType          map[string]int64   `json:"failed_entities_by_type,omitempty"`
	FailedEntityDetails           []EntityHealthInfo `json:"failed_entity_details,omitempty"`
	KubernetesControlPlaneVersion string             `json:"kubernetes_control_plane_version,omitempty"`
}

// String provides a readable representation of the health status for table display
func (h DeploymentCellHealthStatus) String() string {
	var parts []string

	// Add overall status
	parts = append(parts, fmt.Sprintf("Status: %s", h.OverallStatus))

	// Add entity counts
	parts = append(parts, fmt.Sprintf("Entities: %d/%d healthy", h.HealthyEntities, h.TotalEntities))

	// Add failed entities if any
	if h.FailedEntities > 0 {
		parts = append(parts, fmt.Sprintf("Failed: %d", h.FailedEntities))
	}

	// Add failed entity types if present
	if len(h.FailedEntitiesByType) > 0 {
		var failedTypes []string
		for entityType, count := range h.FailedEntitiesByType {
			failedTypes = append(failedTypes, fmt.Sprintf("%s:%d", entityType, count))
		}
		if len(failedTypes) > 0 {
			parts = append(parts, fmt.Sprintf("Failed Types: %s", strings.Join(failedTypes, ", ")))
		}
	}

	return strings.Join(parts, " | ")
}

// DeploymentCellTableView returns a simplified view of the deployment cell for table display
type DeploymentCellTableView struct {
	ID                         string  `json:"id"`
	CustomerEmail              *string `json:"customer_email"`
	CustomerOrganizationName   *string `json:"customer_organization_name"`
	Status                     string  `json:"status"`
	Type                       string  `json:"type"`
	CloudProvider              string  `json:"cloud_provider"`
	Region                     string  `json:"region"`
	CurrentNumberOfDeployments int64   `json:"current_number_of_deployments"`
	HealthStatus               string  `json:"health_status"`
}

// ToTableView converts a DeploymentCell to a table-friendly view
func (dc DeploymentCell) ToTableView() DeploymentCellTableView {
	// For adopted clusters, the key is the ID
	if !strings.HasPrefix(dc.Key, "dataplane-") {
		dc.ID = dc.Key
	}
	return DeploymentCellTableView{
		ID:                         dc.ID,
		CustomerEmail:              dc.CustomerEmail,
		CustomerOrganizationName:   dc.CustomerOrganizationName,
		Status:                     dc.Status,
		Type:                       dc.Type,
		CloudProvider:              dc.CloudProvider,
		Region:                     dc.Region,
		CurrentNumberOfDeployments: dc.CurrentNumberOfDeployments,
		HealthStatus:               dc.HealthStatus.String(),
	}
}

// EntityHealthInfo represents detailed health information for a specific entity
type EntityHealthInfo struct {
	Identifier string  `json:"identifier"`
	Type       string  `json:"type"`
	SyncStatus string  `json:"sync_status"`
	Error      *string `json:"error,omitempty"`
}

// CustomNetworkInfo represents custom network configuration
type CustomNetworkInfo struct {
	ID      *string `json:"id,omitempty"`
	Name    *string `json:"name,omitempty"`
	CIDR    *string `json:"cidr,omitempty"`
	OrgName *string `json:"org_name,omitempty"`
}

// HelmPackageInfo represents a Helm package installed on the cluster
type HelmPackageInfo struct {
	ChartName     string                 `json:"chart_name"`
	ChartVersion  string                 `json:"chart_version"`
	ChartRepoName string                 `json:"chart_repo_name"`
	ChartRepoURL  string                 `json:"chart_repo_url"`
	Namespace     string                 `json:"namespace"`
	ChartValues   map[string]interface{} `json:"chart_values,omitempty"`
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
	HasConfigurationDrift bool                 `json:"has_configuration_drift"`
	DriftDetails          []ConfigurationDrift `json:"drift_details,omitempty"`

	// Pending changes information
	HasPendingChanges bool                         `json:"has_pending_changes"`
	PendingChanges    []PendingConfigurationChange `json:"pending_changes,omitempty"`

	// Status and timestamps
	Status    string    `json:"status"`
	LastCheck time.Time `json:"last_check"`
}

// ConfigurationDrift represents a specific configuration drift
type ConfigurationDrift struct {
	Path         string      `json:"path"`          // JSON path to the drifted configuration
	CurrentValue interface{} `json:"current_value"` // Current value in deployment cell
	TargetValue  interface{} `json:"target_value"`  // Expected value from organization template
	DriftType    string      `json:"drift_type"`    // Type of drift: "missing", "extra", "different"
}

// PendingConfigurationChange represents a pending change to be applied
type PendingConfigurationChange struct {
	Path      string      `json:"path"`                // JSON path of the configuration
	Operation string      `json:"operation"`           // Operation type: "add", "update", "delete"
	OldValue  interface{} `json:"old_value,omitempty"` // Current value (for update/delete)
	NewValue  interface{} `json:"new_value,omitempty"` // New value (for add/update)
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
