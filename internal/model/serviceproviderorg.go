package model

import "time"

// ServiceProviderOrg represents a service provider organization
type ServiceProviderOrg struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// AmenitiesConfiguration represents the organization-level amenities configuration
type AmenitiesConfiguration struct {
	// Organization identifier (comes from credentials)
	OrganizationID string `json:"organization_id"`

	// Environment this configuration applies to - using environment type instead of string
	EnvironmentType string `json:"environment_type"`

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
	OrganizationID  string    `json:"organization_id"`
	EnvironmentType string    `json:"environment_type"`
	Version         string    `json:"version"`
	UpdatedAt       time.Time `json:"updated_at"`
	ConfigCount     int       `json:"config_count"` // Number of configuration items
}

// ToTableView converts AmenitiesConfiguration to table-friendly view
func (ac AmenitiesConfiguration) ToTableView() AmenitiesConfigurationTableView {
	configCount := len(ac.ConfigurationTemplate)
	return AmenitiesConfigurationTableView{
		OrganizationID:  ac.OrganizationID,
		EnvironmentType: ac.EnvironmentType,
		Version:         ac.Version,
		UpdatedAt:       ac.UpdatedAt,
		ConfigCount:     configCount,
	}
}

// Amenity represents an amenity in the deployment cell
type Amenity struct {
	Name        string                 `json:"name" yaml:"name"`
	Description *string                `json:"description,omitempty" yaml:"description,omitempty"`
	Type        *string                `json:"type,omitempty" yaml:"type,omitempty"`
	Properties  map[string]interface{} `json:"properties,omitempty" yaml:"properties,omitempty"`
}

type InternalAmenity struct {
	Name        string                 `json:"name" yaml:"name"`
	Description *string                `json:"description,omitempty" yaml:"description,omitempty"`
	Type        *string                `json:"type,omitempty" yaml:"type,omitempty"`
	IsManaged   *bool                  `json:"isManaged,omitempty" yaml:"isManaged,omitempty"`
	Properties  map[string]interface{} `json:"properties,omitempty" yaml:"properties,omitempty"`
}

// AmenityConfig represents an amenity configuration in a more structured format
type AmenityConfig struct {
	Name        string                 `json:"name" yaml:"name"`
	Modifiable  *bool                  `json:"modifiable,omitempty" yaml:"modifiable,omitempty"`
	Description *string                `json:"description,omitempty" yaml:"description,omitempty"`
	IsManaged   *bool                  `json:"isManaged,omitempty" yaml:"isManaged,omitempty"`
	Type        *string                `json:"type,omitempty" yaml:"type,omitempty"`
	Properties  map[string]interface{} `json:"properties,omitempty" yaml:"properties,omitempty"`
}
