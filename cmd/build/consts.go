package build

const (
	DockerComposeSpecType = "DockerCompose"
	ServicePlanSpecType   = "ServicePlanSpec"
	OmnistrateComposeFileName = "omnistrate-compose.yaml"
	ComposeFileName       = "docker-compose.yaml"
	PlanSpecFileName      = "spec.yaml"
)

var (
	validSpecType = []string{DockerComposeSpecType, ServicePlanSpecType}
)
