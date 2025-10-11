package model

type ServicePlan struct {
	PlanID         string `json:"plan_id,omitempty"`
	PlanName       string `json:"plan_name,omitempty"`
	ServiceID      string `json:"service_id,omitempty"`
	ServiceName    string `json:"service_name,omitempty"`
	Environment    string `json:"environment,omitempty"`
	DeploymentType string `json:"deployment_type,omitempty"`
	TenancyType    string `json:"tenancy_type,omitempty"`
}

type ServicePlanDetails struct {
	PlanID          string                       `json:"plan_id,omitempty"`
	PlanName        string                       `json:"plan_name,omitempty"`
	ServiceID       string                       `json:"service_id,omitempty"`
	ServiceName     string                       `json:"service_name,omitempty"`
	Environment     string                       `json:"environment,omitempty"`
	DeploymentType  string                       `json:"deployment_type,omitempty"`
	TenancyType     string                       `json:"tenancy_type,omitempty"`
	EnabledFeatures any                          `json:"enabled_features,omitempty"`
	Resources       []Resource                   `json:"resources,omitempty"`
	PendingChanges  map[string]ResourceChangeSet `json:"pending_changes,omitempty"`
}

type ResourceChangeSet struct {
	ResourceName              string `json:"resource_name,omitempty"`
	ResourceChanges           any    `json:"resource_changes,omitempty"`
	ProductTierFeatureChanges any    `json:"product_tier_feature_changes,omitempty"`
	ImageConfigChanges        any    `json:"image_config_changes,omitempty"`
	InfraConfigChanges        any    `json:"infra_config_changes,omitempty"`
}

type ServicePlanVersion struct {
	PlanID                         string `json:"plan_id,omitempty"`
	PlanName                       string `json:"plan_name,omitempty"`
	ServiceID                      string `json:"service_id,omitempty"`
	ServiceName                    string `json:"service_name,omitempty"`
	Environment                    string `json:"environment,omitempty"`
	Version                        string `json:"version,omitempty"`
	ReleaseDescription             string `json:"release_description,omitempty"`
	VersionSetStatus               string `json:"version_set_status,omitempty"`
	IsNewServicePlanVersionCreated bool   `json:"is_new_service_plan_version_created,omitempty"`
}

type ServicePlanVersionDetails struct {
	PlanID             string     `json:"plan_id,omitempty"`
	PlanName           string     `json:"plan_name,omitempty"`
	ServiceID          string     `json:"service_id,omitempty"`
	ServiceName        string     `json:"service_name,omitempty"`
	Environment        string     `json:"environment,omitempty"`
	Version            string     `json:"version,omitempty"`
	ReleaseDescription string     `json:"release_description,omitempty"`
	VersionSetStatus   string     `json:"version_set_status,omitempty"`
	EnabledFeatures    any        `json:"enabled_features,omitempty"`
	Resources          []Resource `json:"resources,omitempty"`
}

type Resource struct {
	ResourceID                  string `json:"resource_id,omitempty"`
	ResourceName                string `json:"resource_name,omitempty"`
	ResourceDescription         string `json:"resource_description,omitempty"`
	ResourceType                string `json:"resource_type,omitempty"`
	ActionHooks                 any    `json:"action_hooks,omitempty"`
	AdditionalSecurityContext   any    `json:"additional_security_context,omitempty"`
	BackupConfiguration         any    `json:"backup_configuration,omitempty"`
	Capabilities                any    `json:"capabilities,omitempty"`
	CustomLabels                any    `json:"custom_labels,omitempty"`
	CustomSysCTLs               any    `json:"custom_sysctls,omitempty"`
	CustomULimits               any    `json:"custom_ulimits,omitempty"`
	Dependencies                any    `json:"dependencies,omitempty"`
	EnvironmentVariables        any    `json:"environment_variables,omitempty"`
	FileSystemConfiguration     any    `json:"file_system_configuration,omitempty"`
	HelmChartConfiguration      any    `json:"helm_chart_configuration,omitempty"`
	KustomizeConfiguration      any    `json:"kustomize_configuration,omitempty"`
	L4LoadBalancerConfiguration any    `json:"l4_load_balancer_configuration,omitempty"`
	L7LoadBalancerConfiguration any    `json:"l7_load_balancer_configuration,omitempty"`
	OperatorCRDConfiguration    any    `json:"operator_crd_configuration,omitempty"`
}
