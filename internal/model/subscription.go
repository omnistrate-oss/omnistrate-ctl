package model

type Subscription struct {
	SubscriptionID         string `json:"subscription_id"`
	ServiceID              string `json:"service_id"`
	ServiceName            string `json:"service_name"`
	PlanID                 string `json:"plan_id"`
	PlanName               string `json:"plan_name"`
	Environment            string `json:"environment"`
	SubscriptionOwnerName  string `json:"subscription_owner_name"`
	SubscriptionOwnerEmail string `json:"subscription_owner_email"`
	Status                 string `json:"status"`
}
