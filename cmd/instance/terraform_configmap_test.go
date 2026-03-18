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

func TestMerge(t *testing.T) {
	dataplaneConfigMaps := []corev1.ConfigMap{
		{
			ObjectMeta: metav1.ObjectMeta{Name: "tf-state-tf-r-dataplane-instance-abc"},
			Data:       map[string]string{"main.tf": "dataplane resource"},
		},
		{
			ObjectMeta: metav1.ObjectMeta{Name: "terraform-progress-dp-1"},
			Data:       map[string]string{"log": "dataplane log"},
		},
	}

	controlPlaneConfigMaps := []corev1.ConfigMap{
		{
			ObjectMeta: metav1.ObjectMeta{Name: "tf-state-tf-r-cpresource-instance-abc"},
			Data:       map[string]string{"main.tf": "control plane resource"},
		},
		{
			ObjectMeta: metav1.ObjectMeta{Name: "terraform-progress-cp-1"},
			Data:       map[string]string{"log": "control plane log"},
		},
	}

	dpIndex := newTerraformConfigMapIndex("instance-abc", dataplaneConfigMaps)
	cpIndex := newTerraformConfigMapIndex("instance-abc", controlPlaneConfigMaps)

	dpIndex.merge(cpIndex)

	// Both resources should be present
	if _, ok := dpIndex.stateByResource["tf-r-dataplane"]; !ok {
		t.Fatalf("expected dataplane resource in merged index")
	}
	if _, ok := dpIndex.stateByResource["tf-r-cpresource"]; !ok {
		t.Fatalf("expected control plane resource in merged index")
	}

	// Progress configmaps from both should be present
	if len(dpIndex.progress) != 2 {
		t.Fatalf("expected 2 progress configmaps after merge, got %d", len(dpIndex.progress))
	}
}

func TestMerge_DoesNotOverwrite(t *testing.T) {
	// Same resource ID exists in both clusters — dataplane should win
	dataplaneConfigMaps := []corev1.ConfigMap{
		{
			ObjectMeta: metav1.ObjectMeta{Name: "tf-state-tf-r-shared-instance-abc"},
			Data:       map[string]string{"main.tf": "dataplane version"},
		},
	}
	controlPlaneConfigMaps := []corev1.ConfigMap{
		{
			ObjectMeta: metav1.ObjectMeta{Name: "tf-state-tf-r-shared-instance-abc"},
			Data:       map[string]string{"main.tf": "control plane version"},
		},
	}

	dpIndex := newTerraformConfigMapIndex("instance-abc", dataplaneConfigMaps)
	cpIndex := newTerraformConfigMapIndex("instance-abc", controlPlaneConfigMaps)

	dpIndex.merge(cpIndex)

	cm := dpIndex.stateByResource["tf-r-shared"]
	if cm == nil {
		t.Fatalf("expected tf-r-shared in merged index")
	}
	if cm.Data["main.tf"] != "dataplane version" {
		t.Fatalf("expected dataplane version to be preserved, got %q", cm.Data["main.tf"])
	}
}

func TestMerge_NilOther(t *testing.T) {
	configMaps := []corev1.ConfigMap{
		{ObjectMeta: metav1.ObjectMeta{Name: "tf-state-tf-r-1-instance-abc"}},
	}

	index := newTerraformConfigMapIndex("instance-abc", configMaps)
	index.merge(nil)

	if _, ok := index.stateByResource["tf-r-1"]; !ok {
		t.Fatalf("expected tf-r-1 to still be present after nil merge")
	}
}
