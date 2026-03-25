package dataaccess

import (
	"context"
	"fmt"
	"github.com/rs/zerolog/log"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

// K8sClientConfig contains configuration for the K8s client
type K8sClientConfig struct {
	Kubeconfig  string
	KubeContext string
}

// nodeInfo holds information about a node
type nodeInfo struct {
	Name         string
	InstanceType string
	VCPUs        int
	MemoryGB     float64
	AZ           string
	Pods         []InspectPodItem
}

// K8sInspectClientImpl is the Kubernetes implementation of K8sInspectClient
type K8sInspectClientImpl struct {
	config K8sClientConfig
}

// NewK8sInspectClient creates a new K8s inspection client
func NewK8sInspectClient(config K8sClientConfig) K8sInspectClient {
	return &K8sInspectClientImpl{
		config: config,
	}
}

// GetClusterData fetches real data from a Kubernetes cluster
func (k *K8sInspectClientImpl) GetClusterData(ctx context.Context, namespace string) ([]InspectWorkloadItem, []InspectAZItem, []InspectStorageClassItem, error) {
	// Load Kubernetes configuration
	config, err := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		&clientcmd.ClientConfigLoadingRules{ExplicitPath: k.config.Kubeconfig},
		&clientcmd.ConfigOverrides{CurrentContext: k.config.KubeContext},
	).ClientConfig()
	if err != nil {
		return nil, nil, nil, fmt.Errorf("error building kubeconfig: %v", err)
	}

	// Create a Kubernetes client
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("error creating Kubernetes client: %v", err)
	}

	// Initialize data structures
	var workloadItems []InspectWorkloadItem
	nodesInfo := make(map[string]*nodeInfo)
	podMap := make(map[string][]InspectPodItem)

	// Get nodes information
	nodes, err := clientset.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, nil, nil, fmt.Errorf("error listing nodes: %v", err)
	}

	// Process nodes
	for _, node := range nodes.Items {
		// Extract availability zone from node labels
		az := node.Labels["topology.kubernetes.io/zone"]
		if az == "" {
			// Try alternative labels
			az = node.Labels["failure-domain.beta.kubernetes.io/zone"]
			if az == "" {
				az = "unknown"
			}
		}

		// Extract instance type
		instanceType := node.Labels["node.kubernetes.io/instance-type"]
		if instanceType == "" {
			instanceType = "unknown"
		}

		// Extract CPU and memory info
		cpuStr := node.Status.Capacity.Cpu().String()
		memStr := node.Status.Capacity.Memory().String()

		// Parse CPU count
		cpuCount := 0
		_, err = fmt.Sscanf(cpuStr, "%d", &cpuCount)
		if err != nil {
			return nil, nil, nil, err
		}

		// Create node info
		nodeInf := &nodeInfo{
			Name:         node.Name,
			InstanceType: instanceType,
			VCPUs:        cpuCount,
			MemoryGB:     k.parseMemoryToGB(memStr),
			AZ:           az,
			Pods:         []InspectPodItem{},
		}

		nodesInfo[node.Name] = nodeInf

		// Initialize pod map for this AZ
		if _, exists := podMap[az]; !exists {
			podMap[az] = []InspectPodItem{}
		}
	}

	// Get pods in the namespace
	pods, err := clientset.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, nil, nil, fmt.Errorf("error listing pods in namespace %s: %v", namespace, err)
	}

	// Process pods
	for _, pod := range pods.Items {
		// Extract resource requirements
		resources := ResourceRequirements{
			Limits:   make(ResourceList),
			Requests: make(ResourceList),
		}

		// Get resource limits and requests from all containers
		for _, container := range pod.Spec.Containers {
			// Add CPU limits
			if cpu := container.Resources.Limits.Cpu(); cpu != nil {
				cpuVal := container.Resources.Limits.Cpu().String()
				if val, exists := resources.Limits["cpu"]; exists {
					// Add to existing value
					resources.Limits["cpu"] = k.addResourceValues(val, cpuVal)
				} else {
					resources.Limits["cpu"] = cpuVal
				}
			}

			// Add memory limits
			if mem := container.Resources.Limits.Memory(); mem != nil {
				memVal := container.Resources.Limits.Memory().String()
				if val, exists := resources.Limits["memory"]; exists {
					// Add to existing value
					resources.Limits["memory"] = k.addResourceValues(val, memVal)
				} else {
					resources.Limits["memory"] = memVal
				}
			}

			// Add CPU requests
			if cpu := container.Resources.Requests.Cpu(); cpu != nil {
				cpuVal := container.Resources.Requests.Cpu().String()
				if val, exists := resources.Requests["cpu"]; exists {
					// Add to existing value
					resources.Requests["cpu"] = k.addResourceValues(val, cpuVal)
				} else {
					resources.Requests["cpu"] = cpuVal
				}
			}

			// Add memory requests
			if mem := container.Resources.Requests.Memory(); mem != nil {
				memVal := container.Resources.Requests.Memory().String()
				if val, exists := resources.Requests["memory"]; exists {
					// Add to existing value
					resources.Requests["memory"] = k.addResourceValues(val, memVal)
				} else {
					resources.Requests["memory"] = memVal
				}
			}
		}

		podItem := InspectPodItem{
			Name:      pod.Name,
			Status:    string(pod.Status.Phase),
			NodeName:  pod.Spec.NodeName,
			Namespace: pod.Namespace,
			Labels:    pod.Labels,
			Resources: resources,
		}

		// Add pod to node
		if nodeInfo, exists := nodesInfo[pod.Spec.NodeName]; exists {
			nodeInfo.Pods = append(nodeInfo.Pods, podItem)

			// Add pod to AZ pod map
			az := nodeInfo.AZ
			podMap[az] = append(podMap[az], podItem)
		}
	}

	// Get PVCs in the namespace
	pvcs, err := clientset.CoreV1().PersistentVolumeClaims(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, nil, nil, fmt.Errorf("error listing PVCs in namespace %s: %v", namespace, err)
	}

	// Process PVCs and create a map
	pvcMap := make(map[string]InspectPVCItem)
	for _, pvc := range pvcs.Items {
		pvcItem := InspectPVCItem{
			Name:         pvc.Name,
			Size:         pvc.Spec.Resources.Requests.Storage().String(),
			Status:       string(pvc.Status.Phase),
			PVName:       pvc.Spec.VolumeName,
			StorageClass: *pvc.Spec.StorageClassName,
			AccessModes:  k.accessModesToStrings(pvc.Spec.AccessModes),
		}

		pvcMap[pvc.Name] = pvcItem
	}

	// Create a map to track pod name -> PVCs
	podToPVCsMap := make(map[string][]InspectPVCItem)

	// First pass: gather all PVCs for each pod
	for _, pod := range pods.Items {
		var podPVCs []InspectPVCItem

		// Check each volume in the pod
		for _, vol := range pod.Spec.Volumes {
			if vol.PersistentVolumeClaim != nil {
				if pvcItem, ok := pvcMap[vol.PersistentVolumeClaim.ClaimName]; ok {
					podPVCs = append(podPVCs, pvcItem)
				}
			}
		}

		if len(podPVCs) > 0 {
			// Store the pod's PVCs in the map
			podToPVCsMap[pod.Name] = podPVCs
		}
	}

	// Get StatefulSets
	statefulSets, err := clientset.AppsV1().StatefulSets(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, nil, nil, fmt.Errorf("error listing statefulsets in namespace %s: %v", namespace, err)
	}

	// Process StatefulSets
	for _, sts := range statefulSets.Items {
		// Get pods for this StatefulSet
		azPods := make(map[string][]InspectPodItem)
		podSelector := metav1.FormatLabelSelector(sts.Spec.Selector)
		stsPods, err := clientset.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{
			LabelSelector: podSelector,
		})

		if err == nil {
			for _, pod := range stsPods.Items {
				if nodeInfo, exists := nodesInfo[pod.Spec.NodeName]; exists {
					az := nodeInfo.AZ

					// Create pod item with PVCs if available
					podItem := InspectPodItem{
						Name:      pod.Name,
						Status:    string(pod.Status.Phase),
						NodeName:  pod.Spec.NodeName,
						Namespace: pod.Namespace,
						PVCs:      podToPVCsMap[pod.Name], // Get PVCs from the map
					}

					// Initialize AZ if not exists
					if _, exists := azPods[az]; !exists {
						azPods[az] = []InspectPodItem{}
					}

					// Add pod to AZ map
					azPods[az] = append(azPods[az], podItem)
				}
			}
		}

		// Create workload item
		workloadItem := InspectWorkloadItem{
			Type: "StatefulSet",
			Name: sts.Name,
			AZs:  azPods,
		}

		workloadItems = append(workloadItems, workloadItem)
	}

	// Get Deployments
	deployments, err := clientset.AppsV1().Deployments(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, nil, nil, fmt.Errorf("error listing deployments in namespace %s: %v", namespace, err)
	}

	// Process Deployments
	for _, deploy := range deployments.Items {
		// Get pods for this Deployment
		azPods := make(map[string][]InspectPodItem)
		podSelector := metav1.FormatLabelSelector(deploy.Spec.Selector)
		deployPods, err := clientset.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{
			LabelSelector: podSelector,
		})

		if err == nil {
			for _, pod := range deployPods.Items {
				if nodeInfo, exists := nodesInfo[pod.Spec.NodeName]; exists {
					az := nodeInfo.AZ

					// Create pod item with PVCs if available
					podItem := InspectPodItem{
						Name:      pod.Name,
						Status:    string(pod.Status.Phase),
						NodeName:  pod.Spec.NodeName,
						Namespace: pod.Namespace,
						PVCs:      podToPVCsMap[pod.Name], // Get PVCs from the map
					}

					// Initialize AZ if not exists
					if _, exists := azPods[az]; !exists {
						azPods[az] = []InspectPodItem{}
					}

					// Add pod to AZ map
					azPods[az] = append(azPods[az], podItem)
				}
			}
		}

		// Create workload item
		workloadItem := InspectWorkloadItem{
			Type: "Deployment",
			Name: deploy.Name,
			AZs:  azPods,
		}

		workloadItems = append(workloadItems, workloadItem)
	}

	// Create AZ items
	azItems := []InspectAZItem{}
	for az := range podMap {
		azItem := InspectAZItem{
			Name: az,
			VMs:  []InspectVMItem{},
		}

		// Add VMs (nodes) in this AZ
		for _, nodeInfo := range nodesInfo {
			if nodeInfo.AZ == az {
				vmItem := InspectVMItem{
					Name:         nodeInfo.Name,
					InstanceType: nodeInfo.InstanceType,
					VCPUs:        nodeInfo.VCPUs,
					MemoryGB:     nodeInfo.MemoryGB,
					Pods:         []InspectPodItem{},
				}

				// Add pods to the VM with their PVCs from the map
				for _, pod := range nodeInfo.Pods {
					podWithPVCs := pod
					if pvcs, hasPVCs := podToPVCsMap[pod.Name]; hasPVCs {
						podWithPVCs.PVCs = pvcs
					}
					vmItem.Pods = append(vmItem.Pods, podWithPVCs)
				}

				azItem.VMs = append(azItem.VMs, vmItem)
			}
		}

		azItems = append(azItems, azItem)
	}

	// Get PVs in the cluster
	pvs, err := clientset.CoreV1().PersistentVolumes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return workloadItems, azItems, nil, fmt.Errorf("error listing PVs: %v", err)
	}

	// Get StorageClasses
	scs, err := clientset.StorageV1().StorageClasses().List(ctx, metav1.ListOptions{})
	if err != nil {
		return workloadItems, azItems, nil, fmt.Errorf("error listing StorageClasses: %v", err)
	}

	// Process storage classes
	storageClassMap := make(map[string]*InspectStorageClassItem)
	for _, sc := range scs.Items {
		storageClassMap[sc.Name] = &InspectStorageClassItem{
			Name:        sc.Name,
			Provisioner: sc.Provisioner,
			Parameters:  sc.Parameters,
			PVs:         []InspectPVItem{},
		}
	}

	// Process PVs and link them to StorageClasses
	pvItemMap := make(map[string]InspectPVItem)
	for _, pv := range pvs.Items {
		var storageClass string
		if pv.Spec.StorageClassName != "" {
			storageClass = pv.Spec.StorageClassName
		}

		var pvcName, pvcNamespace string
		if pv.Spec.ClaimRef != nil {
			pvcName = pv.Spec.ClaimRef.Name
			pvcNamespace = pv.Spec.ClaimRef.Namespace
		}

		// Determine volume type based on volume source
		volumeType := "unknown"
		if pv.Spec.HostPath != nil {
			volumeType = "HostPath"
		} else if pv.Spec.GCEPersistentDisk != nil {
			volumeType = "GCEPersistentDisk"
		} else if pv.Spec.AWSElasticBlockStore != nil {
			volumeType = "AWSElasticBlockStore"
		} else if pv.Spec.NFS != nil {
			volumeType = "NFS"
		} else if pv.Spec.ISCSI != nil {
			volumeType = "ISCSI"
		} else if pv.Spec.CSI != nil {
			volumeType = "CSI"
			if pv.Spec.CSI.Driver != "" {
				volumeType = "CSI:" + pv.Spec.CSI.Driver
			}
		}

		pvItem := InspectPVItem{
			Name:         pv.Name,
			Size:         pv.Spec.Capacity.Storage().String(),
			Status:       string(pv.Status.Phase),
			StorageClass: storageClass,
			AccessModes:  k.accessModesToStrings(pv.Spec.AccessModes),
			VolumeType:   volumeType,
			PVCName:      pvcName,
			PVCNamespace: pvcNamespace,
		}

		pvItemMap[pv.Name] = pvItem

		// Add PV to corresponding StorageClass
		if sc, ok := storageClassMap[storageClass]; ok {
			sc.PVs = append(sc.PVs, pvItem)
		}
	}

	// Convert storage class map to slice
	var storageClassItems []InspectStorageClassItem
	for _, sc := range storageClassMap {
		storageClassItems = append(storageClassItems, *sc)
	}

	return workloadItems, azItems, storageClassItems, nil
}

// GetSampleData returns sample data for demonstration
func (k *K8sInspectClientImpl) GetSampleData(instanceID string) ([]InspectWorkloadItem, []InspectAZItem, []InspectStorageClassItem) {
	// Sample PVCs for pods
	pvcPostgresData0 := InspectPVCItem{
		Name:         "postgres-data-0",
		Size:         "10Gi",
		Status:       "Bound",
		PVName:       "pvc-abcd1234",
		StorageClass: "gp2",
		AccessModes:  []string{"ReadWriteOnce"},
	}

	pvcPostgresWal0 := InspectPVCItem{
		Name:         "postgres-wal-0",
		Size:         "5Gi",
		Status:       "Bound",
		PVName:       "pvc-efgh5678",
		StorageClass: "gp2",
		AccessModes:  []string{"ReadWriteOnce"},
	}

	pvcPostgresData1 := InspectPVCItem{
		Name:         "postgres-data-1",
		Size:         "10Gi",
		Status:       "Bound",
		PVName:       "pvc-ijkl9012",
		StorageClass: "gp2",
		AccessModes:  []string{"ReadWriteOnce"},
	}

	pvcPostgresData2 := InspectPVCItem{
		Name:         "postgres-data-2",
		Size:         "10Gi",
		Status:       "Bound",
		PVName:       "pvc-mnop3456",
		StorageClass: "gp2",
		AccessModes:  []string{"ReadWriteOnce"},
	}

	// PVCs for Redis deployment
	pvcRedisData := InspectPVCItem{
		Name:         "redis-data",
		Size:         "8Gi",
		Status:       "Bound",
		PVName:       "pvc-redisdata",
		StorageClass: "gp2",
		AccessModes:  []string{"ReadWriteOnce"},
	}

	pvcRedisConfig := InspectPVCItem{
		Name:         "redis-config",
		Size:         "1Gi",
		Status:       "Bound",
		PVName:       "pvc-redisconfig",
		StorageClass: "gp2",
		AccessModes:  []string{"ReadWriteOnce"},
	}

	// PVC for standalone pod
	pvcStandalone := InspectPVCItem{
		Name:         "standalone-data",
		Size:         "5Gi",
		Status:       "Bound",
		PVName:       "pvc-standalone",
		StorageClass: "gp2",
		AccessModes:  []string{"ReadWriteOnce"},
	}

	// Sample workload items
	workloadItems := []InspectWorkloadItem{
		{
			Type: "StatefulSet",
			Name: "postgres-cluster",
			AZs: map[string][]InspectPodItem{
				"us-west-2a": {
					{
						Name:      "postgres-cluster-0",
						Status:    "Running",
						NodeName:  "node-1a",
						Namespace: instanceID,
						PVCs:      []InspectPVCItem{pvcPostgresData0, pvcPostgresWal0},
						Labels: map[string]string{
							"app":                                "postgres",
							"statefulset.kubernetes.io/pod-name": "postgres-cluster-0",
							"tier":                               "database",
						},
						Resources: ResourceRequirements{
							Limits: ResourceList{
								"cpu":    "1000m",
								"memory": "2Gi",
							},
							Requests: ResourceList{
								"cpu":    "500m",
								"memory": "1Gi",
							},
						},
					},
					{
						Name:      "postgres-cluster-1",
						Status:    "Running",
						NodeName:  "node-1a",
						Namespace: instanceID,
						PVCs:      []InspectPVCItem{pvcPostgresData1},
						Labels: map[string]string{
							"app":                                "postgres",
							"statefulset.kubernetes.io/pod-name": "postgres-cluster-1",
							"tier":                               "database",
						},
						Resources: ResourceRequirements{
							Limits: ResourceList{
								"cpu":    "1000m",
								"memory": "2Gi",
							},
							Requests: ResourceList{
								"cpu":    "500m",
								"memory": "1Gi",
							},
						},
					},
				},
				"us-west-2b": {
					{
						Name:      "postgres-cluster-2",
						Status:    "Running",
						NodeName:  "node-1b",
						Namespace: instanceID,
						PVCs:      []InspectPVCItem{pvcPostgresData2},
						Labels: map[string]string{
							"app":                                "postgres",
							"statefulset.kubernetes.io/pod-name": "postgres-cluster-2",
							"tier":                               "database",
						},
						Resources: ResourceRequirements{
							Limits: ResourceList{
								"cpu":    "1000m",
								"memory": "2Gi",
							},
							Requests: ResourceList{
								"cpu":    "500m",
								"memory": "1Gi",
							},
						},
					},
				},
			},
		},
		{
			Type: "Deployment",
			Name: "api-server",
			AZs: map[string][]InspectPodItem{
				"us-west-2a": {
					{Name: "api-server-abc123", Status: "Running", NodeName: "node-2a", Namespace: instanceID},
				},
				"us-west-2b": {
					{Name: "api-server-def456", Status: "Running", NodeName: "node-2b", Namespace: instanceID},
				},
			},
		},
		{
			Type: "Deployment",
			Name: "redis-cache",
			AZs: map[string][]InspectPodItem{
				"us-west-2a": {
					{
						Name:      "redis-cache-abc123",
						Status:    "Running",
						NodeName:  "node-1a",
						Namespace: instanceID,
						PVCs:      []InspectPVCItem{pvcRedisData},
					},
				},
				"us-west-2c": {
					{
						Name:      "redis-cache-def456",
						Status:    "Running",
						NodeName:  "node-1c",
						Namespace: instanceID,
						PVCs:      []InspectPVCItem{pvcRedisData, pvcRedisConfig},
					},
				},
			},
		},
	}

	// Sample infrastructure items
	azItems := []InspectAZItem{
		{
			Name: "us-west-2a",
			VMs: []InspectVMItem{
				{
					Name:         "node-1a",
					InstanceType: "m5.xlarge",
					VCPUs:        4,
					MemoryGB:     16.0,
					Pods: []InspectPodItem{
						{
							Name:      "postgres-cluster-0",
							Status:    "Running",
							NodeName:  "node-1a",
							Namespace: instanceID,
							PVCs:      []InspectPVCItem{pvcPostgresData0, pvcPostgresWal0},
						},
						{
							Name:      "postgres-cluster-1",
							Status:    "Running",
							NodeName:  "node-1a",
							Namespace: instanceID,
							PVCs:      []InspectPVCItem{pvcPostgresData1},
						},
						{
							Name:      "redis-cache-abc123",
							Status:    "Running",
							NodeName:  "node-1a",
							Namespace: instanceID,
							PVCs:      []InspectPVCItem{pvcRedisData},
						},
						{
							Name:      "standalone-pod-1",
							Status:    "Running",
							NodeName:  "node-1a",
							Namespace: instanceID,
							PVCs:      []InspectPVCItem{pvcStandalone},
						},
					},
				},
				{
					Name:         "node-2a",
					InstanceType: "c5.2xlarge",
					VCPUs:        8,
					MemoryGB:     16.0,
					Pods: []InspectPodItem{
						{Name: "api-server-abc123", Status: "Running", NodeName: "node-2a", Namespace: instanceID},
						{
							Name:      "standalone-pod-2",
							Status:    "Running",
							NodeName:  "node-2a",
							Namespace: instanceID,
						},
					},
				},
			},
		},
		{
			Name: "us-west-2b",
			VMs: []InspectVMItem{
				{
					Name:         "node-1b",
					InstanceType: "m5.xlarge",
					VCPUs:        4,
					MemoryGB:     16.0,
					Pods: []InspectPodItem{
						{
							Name:      "postgres-cluster-2",
							Status:    "Running",
							NodeName:  "node-1b",
							Namespace: instanceID,
							PVCs:      []InspectPVCItem{pvcPostgresData2},
						},
					},
				},
				{
					Name:         "node-2b",
					InstanceType: "c5.2xlarge",
					VCPUs:        8,
					MemoryGB:     16.0,
					Pods: []InspectPodItem{
						{Name: "api-server-def456", Status: "Running", NodeName: "node-2b", Namespace: instanceID},
					},
				},
			},
		},
		{
			Name: "us-west-2c",
			VMs: []InspectVMItem{
				{
					Name:         "node-1c",
					InstanceType: "m5.xlarge",
					VCPUs:        4,
					MemoryGB:     16.0,
					Pods: []InspectPodItem{
						{
							Name:      "redis-cache-def456",
							Status:    "Running",
							NodeName:  "node-1c",
							Namespace: instanceID,
							PVCs:      []InspectPVCItem{pvcRedisData, pvcRedisConfig},
						},
					},
				},
			},
		},
	}

	// Sample PV items
	pvItems := []InspectPVItem{
		{
			Name:         "pvc-abcd1234",
			Size:         "10Gi",
			Status:       "Bound",
			StorageClass: "gp2",
			AccessModes:  []string{"ReadWriteOnce"},
			VolumeType:   "CSI:ebs.csi.aws",
			PVCName:      "postgres-data-0",
			PVCNamespace: instanceID,
		},
		{
			Name:         "pvc-efgh5678",
			Size:         "5Gi",
			Status:       "Bound",
			StorageClass: "gp2",
			AccessModes:  []string{"ReadWriteOnce"},
			VolumeType:   "CSI:ebs.csi.aws",
			PVCName:      "postgres-wal-0",
			PVCNamespace: instanceID,
		},
		{
			Name:         "pvc-ijkl9012",
			Size:         "10Gi",
			Status:       "Bound",
			StorageClass: "gp2",
			AccessModes:  []string{"ReadWriteOnce"},
			VolumeType:   "CSI:ebs.csi.aws",
			PVCName:      "postgres-data-1",
			PVCNamespace: instanceID,
		},
		{
			Name:         "pvc-mnop3456",
			Size:         "10Gi",
			Status:       "Bound",
			StorageClass: "gp2",
			AccessModes:  []string{"ReadWriteOnce"},
			VolumeType:   "CSI:ebs.csi.aws",
			PVCName:      "postgres-data-2",
			PVCNamespace: instanceID,
		},
		{
			Name:         "pvc-redisdata",
			Size:         "8Gi",
			Status:       "Bound",
			StorageClass: "gp2",
			AccessModes:  []string{"ReadWriteOnce"},
			VolumeType:   "CSI:ebs.csi.aws",
			PVCName:      "redis-data",
			PVCNamespace: instanceID,
		},
		{
			Name:         "pvc-redisconfig",
			Size:         "1Gi",
			Status:       "Bound",
			StorageClass: "gp2",
			AccessModes:  []string{"ReadWriteOnce"},
			VolumeType:   "CSI:ebs.csi.aws",
			PVCName:      "redis-config",
			PVCNamespace: instanceID,
		},
		{
			Name:         "pvc-standalone",
			Size:         "5Gi",
			Status:       "Bound",
			StorageClass: "gp2",
			AccessModes:  []string{"ReadWriteOnce"},
			VolumeType:   "CSI:ebs.csi.aws",
			PVCName:      "standalone-data",
			PVCNamespace: instanceID,
		},
	}

	// Sample storage classes
	storageClasses := []InspectStorageClassItem{
		{
			Name:        "gp2",
			Provisioner: "ebs.csi.aws.com",
			Parameters: map[string]string{
				"type":   "gp2",
				"fsType": "ext4",
			},
			PVs: pvItems,
		},
		{
			Name:        "standard",
			Provisioner: "k8s.io/minikube-hostpath",
			Parameters:  map[string]string{},
			PVs:         []InspectPVItem{},
		},
	}

	return workloadItems, azItems, storageClasses
}

// parseMemoryToGB converts memory string (like "16129108Ki") to GB float
func (k *K8sInspectClientImpl) parseMemoryToGB(memStr string) float64 {
	var value float64
	var unit string

	_, err := fmt.Sscanf(memStr, "%f%s", &value, &unit)
	if err != nil {
		log.Fatal().Err(err).Msg("Error parsing memory string")
		return 0
	}

	switch unit {
	case "Ki":
		return value / (1024 * 1024)
	case "Mi":
		return value / 1024
	case "Gi":
		return value
	case "Ti":
		return value * 1024
	default:
		return value / (1024 * 1024 * 1024)
	}
}

// accessModesToStrings converts Kubernetes AccessModes to string representation
func (k *K8sInspectClientImpl) accessModesToStrings(modes []corev1.PersistentVolumeAccessMode) []string {
	var result []string

	for _, mode := range modes {
		switch mode {
		case corev1.ReadWriteOnce:
			result = append(result, "RWO")
		case corev1.ReadOnlyMany:
			result = append(result, "ROX")
		case corev1.ReadWriteMany:
			result = append(result, "RWX")
		case corev1.ReadWriteOncePod:
			result = append(result, "RWOP")
		default:
			result = append(result, string(mode))
		}
	}

	return result
}

// addResourceValues adds two resource values (cpu, memory) as strings
// For simplicity, this is a very naive implementation that just appends values
func (k *K8sInspectClientImpl) addResourceValues(val1, val2 string) string {
	// For now, just concatenate the values as a list
	return val1 + " + " + val2

	// In a real implementation, you would parse the values, add them, and format properly
	// This would handle conversions between units (m, Ki, Mi, etc.)
}
