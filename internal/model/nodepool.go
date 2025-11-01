package model

// NodepoolTableView represents a table view of a nodepool
type NodepoolTableView struct {
	Name          string            `json:"name"`
	Type          string            `json:"type"`
	MachineType   string            `json:"machineType,omitempty"`
	DiskSizeGB    int64             `json:"diskSizeGb"`
	DiskType      string            `json:"diskType,omitempty"`
	ImageType     string            `json:"imageType,omitempty"`
	MinNodes      int64             `json:"minNodes"`
	MaxNodes      int64             `json:"maxNodes"`
	DesiredNodes  int64             `json:"desiredNodes"`
	CurrentNodes  int64             `json:"currentNodes"`
	Location      string            `json:"location,omitempty"`
	AutoRepair    bool              `json:"autoRepair,omitempty"`
	AutoUpgrade   bool              `json:"autoUpgrade,omitempty"`
	CapacityType  string            `json:"capacityType,omitempty"`
	PrivateSubnet bool              `json:"privateSubnet,omitempty"`
	Labels        map[string]string `json:"labels,omitempty"`
}
