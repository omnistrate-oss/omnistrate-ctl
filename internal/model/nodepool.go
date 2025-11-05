package model

// NodepoolTableView represents a table view of a nodepool for list operations
type NodepoolTableView struct {
	Name          string `json:"name"`
	Type          string `json:"type"`
	MachineType   string `json:"machineType,omitempty"`
	ImageType     string `json:"imageType,omitempty"`
	MinNodes      int64  `json:"minNodes"`
	MaxNodes      int64  `json:"maxNodes"`
	Location      string `json:"location,omitempty"`
	AutoRepair    bool   `json:"autoRepair,omitempty"`
	AutoUpgrade   bool   `json:"autoUpgrade,omitempty"`
	AutoScaling   bool   `json:"autoScaling,omitempty"`
	CapacityType  string `json:"capacityType,omitempty"`
	PrivateSubnet bool   `json:"privateSubnet,omitempty"`
}

// NodepoolDescribeView represents a table view of a nodepool for describe operations
type NodepoolDescribeView struct {
	Name          string `json:"name"`
	Type          string `json:"type"`
	MachineType   string `json:"machineType,omitempty"`
	ImageType     string `json:"imageType,omitempty"`
	MinNodes      int64  `json:"minNodes"`
	MaxNodes      int64  `json:"maxNodes"`
	CurrentNodes  int64  `json:"currentNodes"`
	Location      string `json:"location,omitempty"`
	AutoRepair    bool   `json:"autoRepair,omitempty"`
	AutoUpgrade   bool   `json:"autoUpgrade,omitempty"`
	AutoScaling   bool   `json:"autoScaling,omitempty"`
	CapacityType  string `json:"capacityType,omitempty"`
	PrivateSubnet bool   `json:"privateSubnet,omitempty"`
}
