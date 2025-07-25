package model

import "time"

type Service struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Environments string `json:"environments"`
}

// AmenitiesConfiguration represents the organization-level amenities configuration
type AmenitiesConfiguration struct {
	// Organization identifier (comes from credentials)
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
