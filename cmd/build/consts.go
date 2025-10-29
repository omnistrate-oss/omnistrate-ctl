package build

const (
	DockerComposeSpecType = "DockerCompose"
	ServicePlanSpecType   = "ServicePlanSpec"
	ComposeFileName       = "compose.yaml"
	PlanSpecFileName      = "spec.yaml"
	DeploymentTypeHosted  = "hosted"
	DeploymentTypeByoa    = "byoa"
)

var (
	validSpecType = []string{DockerComposeSpecType, ServicePlanSpecType}
)
