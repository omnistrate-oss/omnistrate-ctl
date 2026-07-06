package dataaccess

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	openapiclientfleet "github.com/omnistrate-oss/omnistrate-sdk-go/fleet"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// LogsStream represents a log stream configuration
type LogsStream struct {
	PodName       string `json:"podName"`
	Namespace     string `json:"namespace,omitempty"`
	ContainerName string `json:"containerName,omitempty"`
	ClusterID     string `json:"clusterId,omitempty"`
	ClusterRole   string `json:"clusterRole,omitempty"`
	LogsURL       string `json:"-"` // Deprecated: logs are now read directly through Kubernetes pod logs.

	clientset kubernetes.Interface
}

// LogsService provides methods for log-related operations
type LogsService struct {
	k8sClientForCell k8sClientForCellFunc
}

type k8sClientForCellFunc func(ctx context.Context, token, deploymentCellID, role string) (kubernetes.Interface, error)

type logsFeatureConfig struct {
	Enabled *bool `json:"enabled"`
}

// NewLogsService creates a new LogsService instance
func NewLogsService() *LogsService {
	return &LogsService{
		k8sClientForCell: defaultK8sClientForCell,
	}
}

func defaultK8sClientForCell(ctx context.Context, token, deploymentCellID, role string) (kubernetes.Interface, error) {
	return NewK8sClientForDeploymentCell(ctx, token, deploymentCellID, role)
}

// IsLogsEnabled checks if logs are enabled for the given resource instance
func (ls *LogsService) IsLogsEnabled(instance *openapiclientfleet.ResourceInstance) bool {
	if instance == nil {
		return false
	}

	features := instance.ConsumptionResourceInstanceResult.ProductTierFeatures
	if features == nil {
		return false
	}

	for _, featureKey := range []string{"LOGS#INTERNAL", "LOGS"} {
		enabled, _ := logsFeatureEnabled(features[featureKey])
		if enabled {
			return true
		}
	}
	return false
}

func logsFeatureEnabled(raw interface{}) (bool, error) {
	if raw == nil {
		return false, nil
	}

	switch feature := raw.(type) {
	case bool:
		return feature, nil
	case map[string]interface{}:
		enabled, ok := feature["enabled"].(bool)
		if ok {
			return enabled, nil
		}
		return true, nil
	case logsFeatureConfig:
		if feature.Enabled != nil {
			return *feature.Enabled, nil
		}
		return true, nil
	case *logsFeatureConfig:
		if feature == nil {
			return false, nil
		}
		if feature.Enabled != nil {
			return *feature.Enabled, nil
		}
		return true, nil
	}

	data, err := json.Marshal(raw)
	if err != nil {
		return false, err
	}
	var feature logsFeatureConfig
	if err := json.Unmarshal(data, &feature); err != nil {
		return false, err
	}
	if feature.Enabled != nil {
		return *feature.Enabled, nil
	}
	return true, nil
}

// BuildLogStreams creates log stream configurations for a specific resource
func (ls *LogsService) BuildLogStreams(ctx context.Context, token string, instance *openapiclientfleet.ResourceInstance, instanceID string, resourceKey string) ([]LogsStream, error) {
	if instance == nil {
		return nil, fmt.Errorf("instance is nil")
	}

	topology := instance.ConsumptionResourceInstanceResult.DetailedNetworkTopology
	if topology == nil {
		return nil, fmt.Errorf("no network topology available")
	}

	clusters, err := ls.logClusterClients(ctx, token, instance)
	if err != nil {
		return nil, err
	}

	for _, entry := range *topology {
		if entry.ResourceKey != resourceKey {
			continue
		}

		logStreams, err := ls.buildLogStreamsForTopologyEntry(ctx, clusters, instanceID, entry)
		if err != nil {
			return nil, err
		}
		if len(logStreams) == 0 {
			return nil, fmt.Errorf("no log streams found for resource %s", resourceKey)
		}
		return logStreams, nil
	}

	return nil, fmt.Errorf("resource %s not found in network topology", resourceKey)
}

// GetAllLogStreamsForInstance gets all available log streams for an instance
func (ls *LogsService) GetAllLogStreamsForInstance(ctx context.Context, token string, instance *openapiclientfleet.ResourceInstance, instanceID string) (map[string][]LogsStream, error) {
	if instance == nil {
		return nil, fmt.Errorf("instance is nil")
	}

	topology := instance.ConsumptionResourceInstanceResult.DetailedNetworkTopology
	if topology == nil {
		return nil, fmt.Errorf("no network topology available")
	}

	clusters, err := ls.logClusterClients(ctx, token, instance)
	if err != nil {
		return nil, err
	}

	result := make(map[string][]LogsStream)
	resourceAliases := logResourceAliases(instance)
	var discoveryErrors []string

	for _, entry := range *topology {
		if entry.ResourceKey == "omnistrateobserv" {
			continue
		}

		logStreams, err := ls.buildLogStreamsForTopologyEntry(ctx, clusters, instanceID, entry)
		if err != nil {
			discoveryErrors = append(discoveryErrors, fmt.Sprintf("%s: %v", entry.ResourceKey, err))
			continue
		}
		if len(logStreams) > 0 {
			for _, alias := range logStreamAliasesForTopologyEntry(entry, resourceAliases) {
				result[alias] = logStreams
			}
		}
	}

	if len(result) == 0 && len(discoveryErrors) > 0 {
		return result, fmt.Errorf("no Kubernetes pod log streams found: %s", strings.Join(discoveryErrors, "; "))
	}

	return result, nil
}

// OpenLogStream opens a Kubernetes pod log stream for a configured LogsStream.
func (ls *LogsService) OpenLogStream(ctx context.Context, stream LogsStream) (io.ReadCloser, error) {
	if stream.clientset == nil {
		return nil, fmt.Errorf("kubernetes client is not configured for pod %s", stream.PodName)
	}
	if strings.TrimSpace(stream.Namespace) == "" {
		return nil, fmt.Errorf("namespace is required for pod %s", stream.PodName)
	}
	if strings.TrimSpace(stream.PodName) == "" {
		return nil, fmt.Errorf("pod name is required")
	}

	tailLines := int64(200)
	options := &corev1.PodLogOptions{
		Container: stream.ContainerName,
		Follow:    true,
		TailLines: &tailLines,
	}
	return stream.clientset.CoreV1().Pods(stream.Namespace).GetLogs(stream.PodName, options).Stream(ctx)
}

type logClusterClient struct {
	id        string
	role      string
	clientset kubernetes.Interface
}

func (ls *LogsService) logClusterClients(ctx context.Context, token string, instance *openapiclientfleet.ResourceInstance) ([]logClusterClient, error) {
	if instance == nil {
		return nil, fmt.Errorf("instance is nil")
	}

	loader := ls.k8sClientForCell
	if loader == nil {
		loader = defaultK8sClientForCell
	}

	deploymentCellID := instance.GetDeploymentCellID()
	if deploymentCellID == "" {
		return nil, fmt.Errorf("deployment cell ID not found")
	}

	dataplaneClient, err := loader(ctx, token, deploymentCellID, "cluster-admin")
	if err != nil {
		return nil, fmt.Errorf("failed to create Kubernetes client for deployment cell %s: %w", deploymentCellID, err)
	}

	clusters := []logClusterClient{{
		id:        deploymentCellID,
		role:      "dataplane",
		clientset: dataplaneClient,
	}}

	controlPlaneDeploymentCellID := instance.GetControlPlaneDeploymentCellID()
	if controlPlaneDeploymentCellID != "" && controlPlaneDeploymentCellID != deploymentCellID {
		controlPlaneClient, err := loader(ctx, token, controlPlaneDeploymentCellID, "cluster-admin")
		if err != nil {
			return nil, fmt.Errorf("failed to create Kubernetes client for control plane deployment cell %s: %w", controlPlaneDeploymentCellID, err)
		}
		clusters = append(clusters, logClusterClient{
			id:        controlPlaneDeploymentCellID,
			role:      "control-plane",
			clientset: controlPlaneClient,
		})
	}

	return clusters, nil
}

func (ls *LogsService) buildLogStreamsForTopologyEntry(ctx context.Context, clusters []logClusterClient, instanceID string, entry openapiclientfleet.ResourceNetworkTopologyResult) ([]LogsStream, error) {
	namespace := logNamespaceForInstance(instanceID)
	var streams []LogsStream
	var missingPods []string

	for _, node := range entry.Nodes {
		podName := strings.TrimSpace(node.GetId())
		if podName == "" {
			continue
		}

		found := false
		for _, cluster := range clusters {
			pod, err := cluster.clientset.CoreV1().Pods(namespace).Get(ctx, podName, metav1.GetOptions{})
			if err != nil {
				if apierrors.IsNotFound(err) {
					continue
				}
				return nil, fmt.Errorf("failed to get pod %s/%s from %s deployment cell %s: %w", namespace, podName, cluster.role, cluster.id, err)
			}
			streams = append(streams, logStreamsForPod(cluster, namespace, pod)...)
			found = true
			break
		}
		if !found {
			missingPods = append(missingPods, podName)
		}
	}

	if len(streams) == 0 {
		fallbackStreams, err := ls.findResourcePodsByName(ctx, clusters, namespace, entry)
		if err != nil {
			return nil, err
		}
		streams = fallbackStreams
	}

	if len(streams) == 0 && len(missingPods) > 0 {
		return nil, fmt.Errorf("pods not found in namespace %s: %s", namespace, strings.Join(missingPods, ", "))
	}

	return streams, nil
}

func logStreamsForPod(cluster logClusterClient, namespace string, pod *corev1.Pod) []LogsStream {
	if pod == nil {
		return nil
	}

	containers := pod.Spec.Containers
	if len(containers) == 0 {
		return []LogsStream{{
			PodName:     pod.Name,
			Namespace:   namespace,
			ClusterID:   cluster.id,
			ClusterRole: cluster.role,
			clientset:   cluster.clientset,
		}}
	}

	streams := make([]LogsStream, 0, len(containers))
	for _, container := range containers {
		streams = append(streams, LogsStream{
			PodName:       pod.Name,
			Namespace:     namespace,
			ContainerName: container.Name,
			ClusterID:     cluster.id,
			ClusterRole:   cluster.role,
			clientset:     cluster.clientset,
		})
	}
	return streams
}

func (ls *LogsService) findResourcePodsByName(ctx context.Context, clusters []logClusterClient, namespace string, entry openapiclientfleet.ResourceNetworkTopologyResult) ([]LogsStream, error) {
	var streams []LogsStream
	for _, cluster := range clusters {
		pods, err := cluster.clientset.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{})
		if err != nil {
			if apierrors.IsNotFound(err) {
				continue
			}
			return nil, fmt.Errorf("failed to list pods in namespace %s from %s deployment cell %s: %w", namespace, cluster.role, cluster.id, err)
		}
		for i := range pods.Items {
			pod := &pods.Items[i]
			if !podMatchesResource(entry, pod) {
				continue
			}
			streams = append(streams, logStreamsForPod(cluster, namespace, pod)...)
		}
	}
	return streams, nil
}

func podMatchesResource(entry openapiclientfleet.ResourceNetworkTopologyResult, pod *corev1.Pod) bool {
	if pod == nil {
		return false
	}

	candidates := []string{
		strings.TrimSpace(entry.ResourceKey),
		strings.TrimSpace(entry.ResourceName),
	}
	for _, candidate := range candidates {
		if candidate == "" {
			continue
		}
		if pod.Name == candidate || strings.HasPrefix(pod.Name, candidate+"-") {
			return true
		}
		for _, labelKey := range []string{
			"app",
			"app.kubernetes.io/name",
			"app.kubernetes.io/instance",
			"omnistrate.com/resource-key",
			"omnistrate.io/resource-key",
			"resource-key",
		} {
			if pod.Labels[labelKey] == candidate {
				return true
			}
		}
	}

	return false
}

func logNamespaceForInstance(instanceID string) string {
	return strings.ToLower(strings.TrimSpace(instanceID))
}

func logResourceAliases(instance *openapiclientfleet.ResourceInstance) map[string][]string {
	aliases := make(map[string][]string)
	if instance == nil {
		return aliases
	}

	for _, summary := range instance.ResourceVersionSummaries {
		resourceName := strings.TrimSpace(summary.GetResourceName())
		resourceID := strings.TrimSpace(summary.GetResourceId())
		if resourceName == "" || resourceID == "" {
			continue
		}
		aliases[resourceName] = append(aliases[resourceName], resourceID)
	}
	return aliases
}

func logStreamAliasesForTopologyEntry(entry openapiclientfleet.ResourceNetworkTopologyResult, aliases map[string][]string) []string {
	seen := make(map[string]struct{})
	var result []string
	add := func(value string) {
		value = strings.TrimSpace(value)
		if value == "" {
			return
		}
		if _, ok := seen[value]; ok {
			return
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}

	add(entry.ResourceKey)
	add(entry.ResourceName)
	for _, alias := range aliases[entry.ResourceKey] {
		add(alias)
	}
	for _, alias := range aliases[entry.ResourceName] {
		add(alias)
	}
	return result
}
