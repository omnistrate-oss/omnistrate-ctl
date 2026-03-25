package model

type Instance struct {
	InstanceID     string `json:"instance_id"`
	Service        string `json:"service"`
	Environment    string `json:"environment"`
	Plan           string `json:"plan"`
	Version        string `json:"version"`
	Resource       string `json:"resource"`
	CloudProvider  string `json:"cloud_provider"`
	Region         string `json:"region"`
	Status         string `json:"status"`
	SubscriptionID string `json:"subscription_id"`
	Tags           string `json:"tags"`
}
