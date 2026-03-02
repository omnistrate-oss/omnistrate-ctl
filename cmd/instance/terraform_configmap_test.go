package instance

import (
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestNormalizeInstanceIDForConfigMap(t *testing.T) {
	tests := []struct {
		name       string
		instanceID string
		expected   string
	}{
		{"with prefix", "instance-abc123", "abc123"},
		{"without prefix", "abc123", "abc123"},
		{"empty", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := normalizeInstanceIDForConfigMap(tt.instanceID); got != tt.expected {
				t.Fatalf("expected %q, got %q", tt.expected, got)
			}
		})
	}
}

func TestNewTerraformConfigMapIndex(t *testing.T) {
	configMaps := []corev1.ConfigMap{
		{ObjectMeta: metav1.ObjectMeta{Name: "tf-state-tf-r-1-instance-abc"}},
		{ObjectMeta: metav1.ObjectMeta{Name: "tf-state-tf-r-2-instance-xyz"}},
		{ObjectMeta: metav1.ObjectMeta{Name: "terraform-progress-111"}},
		{ObjectMeta: metav1.ObjectMeta{Name: "random-config"}},
	}

	index := newTerraformConfigMapIndex("instance-abc", configMaps)

	if _, ok := index.stateByResource["tf-r-1"]; !ok {
		t.Fatalf("expected tf-r-1 state configmap to be indexed")
	}
	if _, ok := index.stateByResource["tf-r-2"]; ok {
		t.Fatalf("did not expect tf-r-2 state configmap to be indexed for instance-abc")
	}
	if len(index.progress) != 1 {
		t.Fatalf("expected 1 progress configmap, got %d", len(index.progress))
	}
}

func TestFindBestProgressConfigMap(t *testing.T) {
	now := time.Now()
	configMaps := []corev1.ConfigMap{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "terraform-progress-1",
				Labels:            map[string]string{"resourceId": "tf-r-1", "instanceId": "instance-abc"},
				CreationTimestamp: metav1.NewTime(now.Add(-time.Minute)),
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "terraform-progress-2",
				Labels:            map[string]string{"resourceId": "tf-r-1", "instanceId": "instance-abc"},
				CreationTimestamp: metav1.NewTime(now),
			},
		},
	}

	index := newTerraformConfigMapIndex("instance-abc", configMaps)
	best := index.findBestProgressConfigMap("tf-r-1")
	if best == nil {
		t.Fatalf("expected to find a progress configmap")
	}
	if best.Name != "terraform-progress-2" {
		t.Fatalf("expected terraform-progress-2, got %s", best.Name)
	}
}
