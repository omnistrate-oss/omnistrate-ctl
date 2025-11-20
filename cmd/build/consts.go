package build

const (
	DockerComposeSpecType      = "DockerCompose"
	ServicePlanSpecType        = "ServicePlanSpec"
	OmnistrateComposeFileName  = "omnistrate-compose.yaml"
	DockerComposeFileName      = "docker-compose.yaml"
	ComposeFileName            = "compose.yaml"
	PlanSpecFileName           = "spec.yaml"
	DeploymentTypeHosted       = "hosted"
	DeploymentTypeByoa         = "byoa"
)

var (
	validSpecType = []string{DockerComposeSpecType, ServicePlanSpecType}
)
