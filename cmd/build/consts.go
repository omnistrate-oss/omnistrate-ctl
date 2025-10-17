package build

const (
	DockerComposeSpecType = "DockerCompose"
	ServicePlanSpecType   = "ServicePlanSpec"
	ComposeFileName       = "compose.yaml"
	PlanSpecFileName      = "spec.yaml"
	githubPAT             = "${{ secrets.GitHubPAT }}"
)

var (
	validSpecType = []string{DockerComposeSpecType, ServicePlanSpecType}
)
