package model

type Environment struct {
	EnvironmentID   string `json:"environment_id"`
	EnvironmentName string `json:"environment_name"`
	EnvironmentType string `json:"environment_type"`
	ServiceID       string `json:"service_id"`
	ServiceName     string `json:"service_name"`
	SourceEnvName   string `json:"source_env_name"`
}

type DetailedEnvironment struct {
	EnvironmentID    string `json:"environment_id,omitempty"`
	EnvironmentName  string `json:"environment_name,omitempty"`
	EnvironmentType  string `json:"environment_type,omitempty"`
	ServiceID        string `json:"service_id,omitempty"`
	ServiceName      string `json:"service_name,omitempty"`
	SourceEnvName    string `json:"source_env_name,omitempty"`
	PromoteStatus    string `json:"promote_status,omitempty"`
	SaaSPortalStatus string `json:"saas_portal_status,omitempty"`
	SaaSPortalURL    string `json:"saas_portal_url,omitempty"`
}

type Promotion struct {
	ServiceID             string `json:"service_id"`
	ServiceName           string `json:"service_name"`
	SourceEnvironmentID   string `json:"source_environment_id"`
	SourceEnvironmentName string `json:"source_environment_name"`
	TargetEnvID           string `json:"target_env_id"`
	TargetEnvName         string `json:"target_env_name"`
	PromoteStatus         string `json:"promote_status"`
}
