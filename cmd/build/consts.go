package build

const (
	DockerComposeSpecType = "DockerCompose"
	ServicePlanSpecType   = "ServicePlanSpec"
	ComposeFileName       = "compose.yaml"
	PlanSpecFileName      = "spec.yaml"
)

var (
	validSpecType = []string{DockerComposeSpecType, ServicePlanSpecType}
)
