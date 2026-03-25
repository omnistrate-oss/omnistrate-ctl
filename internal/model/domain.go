package model

type Domain struct {
	EnvironmentType string `json:"environment_type"`
	Name            string `json:"name"`
	Domain          string `json:"domain"`
	Status          string `json:"status"`
	ClusterEndpoint string `json:"cluster_endpoint"`
}
