package model

type Secret struct {
	EnvironmentType string `json:"environment_type"`
	Name            string `json:"name"`
	Value           string `json:"value,omitempty"`
}

type SecretList struct {
	Secrets []Secret `json:"secrets"`
}
