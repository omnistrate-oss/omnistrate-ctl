package dataaccess

import (
	"context"
)

// InspectWorkloadItem represents a workload (StatefulSet or Deployment)
type InspectWorkloadItem struct {
	Type string // StatefulSet or Deployment
	Name string
	AZs  map[string][]InspectPodItem
}

// InspectVMItem represents a node in the cluster
type InspectVMItem struct {
	Name         string
	InstanceType string
	VCPUs        int
	MemoryGB     float64
	Pods         []InspectPodItem
}

// ResourceRequirements contains resource requirements for a container
type ResourceRequirements struct {
	Limits   ResourceList
	Requests ResourceList
}

// ResourceList is a mapping of resource names to resource quantities
type ResourceList map[string]string

// InspectPodItem represents a pod
type InspectPodItem struct {
	Name      string
	Status    string
	NodeName  string
	Namespace string
	PVCs      []InspectPVCItem // PVCs attached to this pod
	Labels    map[string]string
	Resources ResourceRequirements
}

// InspectPVCItem represents a persistent volume claim
type InspectPVCItem struct {
	Name         string
	Size         string
	Status       string
	PVName       string
	StorageClass string
	AccessModes  []string
}

// InspectPVItem represents a persistent volume
type InspectPVItem struct {
	Name         string
	Size         string
	Status       string
	StorageClass string
	AccessModes  []string
	VolumeType   string
	PVCName      string
	PVCNamespace string
}

// InspectAZItem represents an availability zone
type InspectAZItem struct {
	Name string
	VMs  []InspectVMItem
}

// InspectStorageClassItem represents a storage class
type InspectStorageClassItem struct {
	Name        string
	Provisioner string
	Parameters  map[string]string
	PVs         []InspectPVItem
}

// K8sInspectClient provides methods for retrieving inspect data
type K8sInspectClient interface {
	// GetClusterData returns detailed information about workloads, AZs, and storage in a namespace
	GetClusterData(ctx context.Context, namespace string) ([]InspectWorkloadItem, []InspectAZItem, []InspectStorageClassItem, error)

	// GetSampleData returns sample data for demonstration purposes
	GetSampleData(instanceID string) ([]InspectWorkloadItem, []InspectAZItem, []InspectStorageClassItem)
}
