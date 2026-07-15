package model

type Account struct {
	ID              string `json:"id"`
	Name            string `json:"name"`
	Status          string `json:"status"`
	CloudProvider   string `json:"cloud_provider"`
	TargetAccountID string `json:"target_account_id"`
	CustomTags      string `json:"custom_tags,omitempty"`
}
