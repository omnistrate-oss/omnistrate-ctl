package model

type CustomerUser struct {
	UserID            string `json:"user_id"`
	UserName          string `json:"user_name"`
	Email             string `json:"email"`
	Status            string `json:"status"`
	Enabled           string `json:"enabled"`
	OrgID             string `json:"org_id"`
	OrgName           string `json:"org_name"`
	SubscriptionCount string `json:"subscription_count,omitempty"`
	InstanceCount     string `json:"instance_count,omitempty"`
	CreatedAt         string `json:"created_at,omitempty"`
}
