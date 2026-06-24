package model

type SnapshotDetail struct {
	SnapshotID       string `json:"snapshotId"`
	Status           string `json:"status"`
	Region           string `json:"region"`
	SnapshotType     string `json:"snapshotType"`
	Progress         string `json:"progress"`
	CreatedAt        string `json:"createdAt"`
	CompletedAt      string `json:"completedAt"`
	SourceInstanceID string `json:"sourceInstanceId"`
	ProductTierID    string `json:"productTierId"`
	ProductTierVer   string `json:"productTierVersion"`
	Encrypted        bool   `json:"encrypted"`
	SnapshotMetadata string `json:"snapshotMetadata,omitempty"`
}
