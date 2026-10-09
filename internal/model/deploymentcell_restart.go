package model

// DeploymentCellRestartResult acknowledges a restart request, not its completion.
type DeploymentCellRestartResult struct {
	ID      string `json:"id"`
	Message string `json:"message"`
}
