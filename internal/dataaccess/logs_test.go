package dataaccess

import (
	"context"
	"testing"

	openapiclientfleet "github.com/omnistrate-oss/omnistrate-sdk-go/fleet"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
)

func TestLogsFeatureEnabled(t *testing.T) {
	tests := []struct {
		name    string
		raw     interface{}
		want    bool
		wantErr bool
	}{
		{name: "bool true", raw: true, want: true},
		{name: "bool false", raw: false, want: false},
		{name: "enabled map", raw: map[string]interface{}{"enabled": true}, want: true},
		{name: "disabled map", raw: map[string]interface{}{"enabled": false}, want: false},
		{name: "feature config map without enabled", raw: map[string]interface{}{"featureName": "LOGS", "scope": "INTERNAL"}, want: true},
		{name: "malformed map string enabled", raw: map[string]interface{}{"enabled": "false"}, want: false, wantErr: true},
		{name: "malformed map numeric enabled", raw: map[string]interface{}{"enabled": 0}, want: false, wantErr: true},
		{name: "struct enabled", raw: logsFeatureConfig{Enabled: boolPtr(true)}, want: true},
		{name: "struct disabled", raw: logsFeatureConfig{Enabled: boolPtr(false)}, want: false},
		{name: "struct without enabled", raw: logsFeatureConfig{}, want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := logsFeatureEnabled(tt.raw)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			require.Equal(t, tt.want, got)
		})
	}
}

func TestLogsServiceIsLogsEnabled(t *testing.T) {
	tests := []struct {
		name     string
		features map[string]interface{}
		want     bool
	}{
		{
			name: "internal logs enabled",
			features: map[string]interface{}{
				"LOGS#INTERNAL": map[string]interface{}{"enabled": true},
			},
			want: true,
		},
		{
			name: "customer logs enabled",
			features: map[string]interface{}{
				"LOGS": map[string]interface{}{"enabled": true},
			},
			want: true,
		},
		{
			name: "customer logs feature config without enabled",
			features: map[string]interface{}{
				"LOGS": map[string]interface{}{"featureName": "LOGS", "scope": "CUSTOMER"},
			},
			want: true,
		},
		{
			name: "logs disabled",
			features: map[string]interface{}{
				"LOGS#INTERNAL": map[string]interface{}{"enabled": false},
				"LOGS":          map[string]interface{}{"enabled": false},
			},
			want: false,
		},
		{
			name: "malformed enabled value is disabled",
			features: map[string]interface{}{
				"LOGS": map[string]interface{}{"enabled": "false"},
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			instance := &openapiclientfleet.ResourceInstance{
				ConsumptionResourceInstanceResult: openapiclientfleet.DescribeResourceInstanceResult{
					ProductTierFeatures: tt.features,
				},
			}

			require.Equal(t, tt.want, NewLogsService().IsLogsEnabled(instance))
		})
	}
}

func TestLogsServiceBuildLogStreamsUsesKubernetesPods(t *testing.T) {
	deploymentCellID := "hc-123"
	instanceID := "instance-abc"
	topology := map[string]openapiclientfleet.ResourceNetworkTopologyResult{
		"api": {
			ResourceKey:  "api",
			ResourceName: "api-server",
			Nodes: []openapiclientfleet.NodeNetworkTopologyResult{
				{Id: stringPtr("api-0")},
			},
		},
	}
	instance := &openapiclientfleet.ResourceInstance{
		DeploymentCellID: &deploymentCellID,
		ConsumptionResourceInstanceResult: openapiclientfleet.DescribeResourceInstanceResult{
			DetailedNetworkTopology: &topology,
			ProductTierFeatures:     logsEnabledFeatureMap(),
		},
	}

	clientset := fake.NewSimpleClientset(&corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "api-0",
			Namespace: instanceID,
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{Name: "api"},
				{Name: "sidecar"},
			},
		},
	})

	service := NewLogsService()
	service.k8sClientForCell = func(_ context.Context, _ string, cellID string, role string) (kubernetes.Interface, error) {
		require.Equal(t, deploymentCellID, cellID)
		require.Equal(t, "cluster-admin", role)
		return clientset, nil
	}

	streams, err := service.BuildLogStreams(context.Background(), "token", instance, instanceID, "api")

	require.NoError(t, err)
	require.Len(t, streams, 2)
	require.Equal(t, "api-0", streams[0].PodName)
	require.Equal(t, instanceID, streams[0].Namespace)
	require.Equal(t, "api", streams[0].ContainerName)
	require.Equal(t, "sidecar", streams[1].ContainerName)
}

func TestLogsServiceBuildLogStreamsFailsFastWhenLogsDisabled(t *testing.T) {
	deploymentCellID := "hc-123"
	instanceID := "instance-abc"
	topology := map[string]openapiclientfleet.ResourceNetworkTopologyResult{
		"api": {
			ResourceKey: "api",
			Nodes: []openapiclientfleet.NodeNetworkTopologyResult{
				{Id: stringPtr("api-0")},
			},
		},
	}
	instance := &openapiclientfleet.ResourceInstance{
		DeploymentCellID: &deploymentCellID,
		ConsumptionResourceInstanceResult: openapiclientfleet.DescribeResourceInstanceResult{
			DetailedNetworkTopology: &topology,
			ProductTierFeatures: map[string]interface{}{
				"LOGS": map[string]interface{}{"enabled": false},
			},
		},
	}

	service := NewLogsService()
	service.k8sClientForCell = func(context.Context, string, string, string) (kubernetes.Interface, error) {
		require.Fail(t, "kubernetes client should not be created when logs are disabled")
		return nil, nil
	}

	streams, err := service.BuildLogStreams(context.Background(), "token", instance, instanceID, "api")

	require.Error(t, err)
	require.Nil(t, streams)
	require.Contains(t, err.Error(), "logs are not enabled")
}

func TestLogsServiceGetAllLogStreamsSkipsDeprecatedObservabilityResource(t *testing.T) {
	deploymentCellID := "hc-123"
	instanceID := "instance-abc"
	topology := map[string]openapiclientfleet.ResourceNetworkTopologyResult{
		"omnistrateobserv": {
			ResourceKey: "omnistrateobserv",
			Nodes: []openapiclientfleet.NodeNetworkTopologyResult{
				{Id: stringPtr("observ-0")},
			},
		},
		"api": {
			ResourceKey:  "api",
			ResourceName: "api",
			Nodes: []openapiclientfleet.NodeNetworkTopologyResult{
				{Id: stringPtr("api-0")},
			},
		},
	}
	instance := &openapiclientfleet.ResourceInstance{
		DeploymentCellID: &deploymentCellID,
		ConsumptionResourceInstanceResult: openapiclientfleet.DescribeResourceInstanceResult{
			DetailedNetworkTopology: &topology,
			ProductTierFeatures:     logsEnabledFeatureMap(),
		},
	}
	clientset := fake.NewSimpleClientset(&corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "api-0", Namespace: instanceID},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{{Name: "api"}},
		},
	})

	service := NewLogsService()
	service.k8sClientForCell = func(_ context.Context, _ string, _ string, _ string) (kubernetes.Interface, error) {
		return clientset, nil
	}

	streamsByResource, err := service.GetAllLogStreamsForInstance(context.Background(), "token", instance, instanceID)

	require.NoError(t, err)
	require.Contains(t, streamsByResource, "api")
	require.NotContains(t, streamsByResource, "omnistrateobserv")
}

func TestLogsServiceGetAllLogStreamsFailsFastWhenLogsDisabled(t *testing.T) {
	deploymentCellID := "hc-123"
	instanceID := "instance-abc"
	topology := map[string]openapiclientfleet.ResourceNetworkTopologyResult{
		"api": {
			ResourceKey: "api",
			Nodes: []openapiclientfleet.NodeNetworkTopologyResult{
				{Id: stringPtr("api-0")},
			},
		},
	}
	instance := &openapiclientfleet.ResourceInstance{
		DeploymentCellID: &deploymentCellID,
		ConsumptionResourceInstanceResult: openapiclientfleet.DescribeResourceInstanceResult{
			DetailedNetworkTopology: &topology,
			ProductTierFeatures: map[string]interface{}{
				"LOGS": map[string]interface{}{"enabled": false},
			},
		},
	}

	service := NewLogsService()
	service.k8sClientForCell = func(context.Context, string, string, string) (kubernetes.Interface, error) {
		require.Fail(t, "kubernetes client should not be created when logs are disabled")
		return nil, nil
	}

	streams, err := service.GetAllLogStreamsForInstance(context.Background(), "token", instance, instanceID)

	require.Error(t, err)
	require.Nil(t, streams)
	require.Contains(t, err.Error(), "logs are not enabled")
}

func logsEnabledFeatureMap() map[string]interface{} {
	return map[string]interface{}{
		"LOGS": map[string]interface{}{"enabled": true},
	}
}

func boolPtr(value bool) *bool {
	return &value
}

func stringPtr(value string) *string {
	return &value
}
