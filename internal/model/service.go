package model

type Service struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Environments string `json:"environments"`
}
