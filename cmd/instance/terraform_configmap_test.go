package instance

import (
	"context"
	"fmt"
	"testing"
	"time"

	openapiclientfleet "github.com/omnistrate-oss/omnistrate-sdk-go/fleet"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
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
			require.Equal(t, tt.expected, normalizeInstanceIDForConfigMap(tt.instanceID))
		})
	}
}

func TestNewTerraformConfigMapIndex(t *testing.T) {
	require := require.New(t)

	configMaps := []corev1.ConfigMap{
		{ObjectMeta: metav1.ObjectMeta{Name: "tf-state-tf-r-1-instance-abc"}},
		{ObjectMeta: metav1.ObjectMeta{Name: "tf-state-tf-r-2-instance-xyz"}},
		{ObjectMeta: metav1.ObjectMeta{Name: "terraform-progress-111"}},
		{ObjectMeta: metav1.ObjectMeta{Name: "random-config"}},
	}

	index := newTerraformConfigMapIndex("instance-abc", configMaps)

	_, ok := index.stateByResource["tf-r-1"]
	require.True(ok, "expected tf-r-1 state configmap to be indexed")

	_, ok = index.stateByResource["tf-r-2"]
	require.False(ok, "did not expect tf-r-2 state configmap to be indexed for instance-abc")

	require.Len(index.progress, 1)
}

func TestFindBestProgressConfigMap(t *testing.T) {
	require := require.New(t)

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
	require.NotNil(best)
	require.Equal("terraform-progress-2", best.Name)
}

func TestMerge(t *testing.T) {
	require := require.New(t)

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

	_, ok := dpIndex.stateByResource["tf-r-dataplane"]
	require.True(ok, "expected dataplane resource in merged index")

	_, ok = dpIndex.stateByResource["tf-r-cpresource"]
	require.True(ok, "expected control plane resource in merged index")

	require.Len(dpIndex.progress, 2, "expected 2 progress configmaps after merge")
}

func TestMerge_DoesNotOverwrite(t *testing.T) {
	require := require.New(t)

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
	require.NotNil(cm, "expected tf-r-shared in merged index")
	require.Equal("dataplane version", cm.Data["main.tf"], "expected dataplane version to be preserved")
}

func TestMerge_NilOther(t *testing.T) {
	require := require.New(t)

	configMaps := []corev1.ConfigMap{
		{ObjectMeta: metav1.ObjectMeta{Name: "tf-state-tf-r-1-instance-abc"}},
	}

	index := newTerraformConfigMapIndex("instance-abc", configMaps)
	index.merge(nil)

	_, ok := index.stateByResource["tf-r-1"]
	require.True(ok, "expected tf-r-1 to still be present after nil merge")
}

// makeStubLoader returns a k8sConnectionLoader that serves configmaps from the provided map,
// keyed by cell ID. The fake clientset returns all provided configmaps when listing the
// terraformConfigMapNamespace namespace.
func makeStubLoader(cellConfigMaps map[string][]corev1.ConfigMap) k8sConnectionLoader {
	return func(_ context.Context, _ string, cellID string) (*k8sConnection, error) {
		cms, ok := cellConfigMaps[cellID]
		if !ok {
			return nil, fmt.Errorf("unknown cell %s", cellID)
		}
		objects := make([]runtime.Object, len(cms))
		for i := range cms {
			objects[i] = &cms[i]
		}
		clientset := fake.NewClientset(objects...)
		return &k8sConnection{clientset: clientset}, nil
	}
}

func makeInstanceData(deploymentCellID, controlPlaneDeploymentCellID string) *openapiclientfleet.ResourceInstance {
	inst := &openapiclientfleet.ResourceInstance{}
	if deploymentCellID != "" {
		inst.DeploymentCellID = &deploymentCellID
	}
	if controlPlaneDeploymentCellID != "" {
		inst.ControlPlaneDeploymentCellID = &controlPlaneDeploymentCellID
	}
	return inst
}

func TestLoadTerraformConfigMapIndex_DataplaneOnly(t *testing.T) {
	require := require.New(t)
	ctx := context.Background()

	dpCMs := []corev1.ConfigMap{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "tf-state-tf-r-dp-instance-abc",
				Namespace: terraformConfigMapNamespace,
			},
			Data: map[string]string{"main.tf": "dp resource"},
		},
	}

	loader := makeStubLoader(map[string][]corev1.ConfigMap{
		"cell-dp": dpCMs,
	})

	inst := makeInstanceData("cell-dp", "")
	index, conns, err := loadTerraformConfigMapIndexForInstanceWithLoader(ctx, "token", inst, "instance-abc", loader)

	require.NoError(err)
	require.NotNil(index)
	require.NotNil(conns)
	require.NotNil(conns.dataplane)
	require.Nil(conns.controlPlane, "no CP cell set, controlPlane connection should be nil")

	_, ok := index.stateByResource["tf-r-dp"]
	require.True(ok, "expected dp resource in index")
}

func TestLoadTerraformConfigMapIndex_WithControlPlane(t *testing.T) {
	require := require.New(t)
	ctx := context.Background()

	dpCMs := []corev1.ConfigMap{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "tf-state-tf-r-dp-instance-abc",
				Namespace: terraformConfigMapNamespace,
			},
			Data: map[string]string{"main.tf": "dp resource"},
		},
	}
	cpCMs := []corev1.ConfigMap{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "tf-state-tf-r-cp-instance-abc",
				Namespace: terraformConfigMapNamespace,
			},
			Data: map[string]string{"main.tf": "cp resource"},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "terraform-progress-cp-1",
				Namespace: terraformConfigMapNamespace,
			},
			Data: map[string]string{"log": "cp log"},
		},
	}

	loader := makeStubLoader(map[string][]corev1.ConfigMap{
		"cell-dp": dpCMs,
		"cell-cp": cpCMs,
	})

	inst := makeInstanceData("cell-dp", "cell-cp")
	index, conns, err := loadTerraformConfigMapIndexForInstanceWithLoader(ctx, "token", inst, "instance-abc", loader)

	require.NoError(err)
	require.NotNil(index)
	require.NotNil(conns)
	require.NotNil(conns.dataplane)
	require.NotNil(conns.controlPlane, "CP cell set, controlPlane connection should be populated")

	_, ok := index.stateByResource["tf-r-dp"]
	require.True(ok, "expected dp resource in merged index")

	_, ok = index.stateByResource["tf-r-cp"]
	require.True(ok, "expected cp resource in merged index")

	require.Len(index.progress, 1, "expected 1 progress configmap from CP cluster")
}

func TestLoadTerraformConfigMapIndex_SameCellNotDuplicated(t *testing.T) {
	require := require.New(t)
	ctx := context.Background()

	cms := []corev1.ConfigMap{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "tf-state-tf-r-shared-instance-abc",
				Namespace: terraformConfigMapNamespace,
			},
			Data: map[string]string{"main.tf": "shared"},
		},
	}

	loader := makeStubLoader(map[string][]corev1.ConfigMap{
		"cell-same": cms,
	})

	// When both IDs are the same, CP branch should be skipped
	inst := makeInstanceData("cell-same", "cell-same")
	index, conns, err := loadTerraformConfigMapIndexForInstanceWithLoader(ctx, "token", inst, "instance-abc", loader)

	require.NoError(err)
	require.NotNil(index)
	require.NotNil(conns)
	require.Nil(conns.controlPlane, "same cell ID — CP connection should not be created")
	require.Len(index.stateByResource, 1)
}

func TestLoadTerraformConfigMapIndex_MissingDeploymentCell(t *testing.T) {
	require := require.New(t)
	ctx := context.Background()

	loader := makeStubLoader(map[string][]corev1.ConfigMap{})
	inst := &openapiclientfleet.ResourceInstance{}

	_, _, err := loadTerraformConfigMapIndexForInstanceWithLoader(ctx, "token", inst, "instance-abc", loader)
	require.Error(err)
}

func TestExtractPlanPreviewOpSuffix(t *testing.T) {
	tests := []struct {
		name           string
		instanceAndOp  string
		instanceSuffix string
		instanceID     string
		expected       string
	}{
		{
			name:           "suffix match",
			instanceAndOp:  "y24o87zd1-7629a67a7ad45ef55fc4",
			instanceSuffix: "y24o87zd1",
			instanceID:     "instance-y24o87zd1",
			expected:       "7629a67a7ad45ef55fc4",
		},
		{
			name:           "full instance ID match",
			instanceAndOp:  "instance-abc-op123",
			instanceSuffix: "abc",
			instanceID:     "instance-abc",
			expected:       "op123",
		},
		{
			name:           "suffix match takes priority",
			instanceAndOp:  "abc-op456",
			instanceSuffix: "abc",
			instanceID:     "instance-abc",
			expected:       "op456",
		},
		{
			name:           "no match",
			instanceAndOp:  "xyz-op789",
			instanceSuffix: "abc",
			instanceID:     "instance-abc",
			expected:       "",
		},
		{
			name:           "instance suffix only without op suffix",
			instanceAndOp:  "abc",
			instanceSuffix: "abc",
			instanceID:     "instance-abc",
			expected:       "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractPlanPreviewOpSuffix(tt.instanceAndOp, tt.instanceSuffix, tt.instanceID)
			require.Equal(t, tt.expected, result)
		})
	}
}

func TestNewTerraformConfigMapIndex_PlanPreviewCMs(t *testing.T) {
	require := require.New(t)

	configMaps := []corev1.ConfigMap{
		{ObjectMeta: metav1.ObjectMeta{Name: "tf-state-tf-r-abc-instance-xyz"}},
		{
			ObjectMeta: metav1.ObjectMeta{Name: "tf-plan-tf-r-abc-instance-xyz-op111"},
			Data: map[string]string{
				"plan-preview": `{"format_version":"1.2","planned_values":{}}`,
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{Name: "tf-plan-tf-r-abc-instance-xyz-op222"},
			Data: map[string]string{
				"plan-preview-error": "Error: Failed to refresh state",
			},
		},
		{
			// Different instance - should not be indexed
			ObjectMeta: metav1.ObjectMeta{Name: "tf-plan-tf-r-abc-instance-other-op333"},
		},
	}

	index := newTerraformConfigMapIndex("instance-xyz", configMaps)

	require.Len(index.planPreviewByResource, 1)
	entries := index.planPreviewByResource["tf-r-abc"]
	require.Len(entries, 2)

	// Check operation suffixes
	opSuffixes := make(map[string]bool)
	for _, e := range entries {
		opSuffixes[e.opSuffix] = true
	}
	require.True(opSuffixes["op111"])
	require.True(opSuffixes["op222"])
}

func TestPlanPreviewsForResource(t *testing.T) {
	require := require.New(t)

	configMaps := []corev1.ConfigMap{
		{
			ObjectMeta: metav1.ObjectMeta{Name: "tf-plan-tf-r-abc-instance-xyz-op111"},
			Data: map[string]string{
				"plan-preview": `{"format_version":"1.2","planned_values":{}}`,
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{Name: "tf-plan-tf-r-abc-instance-xyz-op222"},
			Data: map[string]string{
				"plan-preview-error": "Error: timeout",
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{Name: "tf-plan-tf-r-abc-instance-xyz-op333"},
			Data: map[string]string{
				"plan-preview":       `{"planned_values":{"outputs":{}}}`,
				"plan-preview-error": "Warning: partial failure",
			},
		},
	}

	index := newTerraformConfigMapIndex("instance-xyz", configMaps)

	// r-abc should match via resourceConfigMapKeys ("tf-r-abc" is the first key tried)
	previews, previewErrors := index.planPreviewsForResource("r-abc")
	require.Len(previews, 2)
	require.Equal(`{"format_version":"1.2","planned_values":{}}`, previews["op111"])
	require.Equal(`{"planned_values":{"outputs":{}}}`, previews["op333"])

	require.Len(previewErrors, 2)
	require.Equal("Error: timeout", previewErrors["op222"])
	require.Equal("Warning: partial failure", previewErrors["op333"])
}

func TestPlanPreviewsForResource_NilIndex(t *testing.T) {
	var index *terraformConfigMapIndex
	previews, previewErrors := index.planPreviewsForResource("r-abc")
	require.Empty(t, previews)
	require.Empty(t, previewErrors)
}

func TestPlanPreviewsForResource_NoMatch(t *testing.T) {
	index := newTerraformConfigMapIndex("instance-xyz", nil)
	previews, previewErrors := index.planPreviewsForResource("r-nonexistent")
	require.Empty(t, previews)
	require.Empty(t, previewErrors)
}

func TestMerge_PlanPreviewCMs(t *testing.T) {
	require := require.New(t)

	dpCMs := []corev1.ConfigMap{
		{
			ObjectMeta: metav1.ObjectMeta{Name: "tf-plan-tf-r-dp-instance-abc-op1"},
			Data: map[string]string{
				"plan-preview": `{"dp":"plan"}`,
			},
		},
	}
	cpCMs := []corev1.ConfigMap{
		{
			ObjectMeta: metav1.ObjectMeta{Name: "tf-plan-tf-r-cp-instance-abc-op2"},
			Data: map[string]string{
				"plan-preview": `{"cp":"plan"}`,
			},
		},
	}

	dpIndex := newTerraformConfigMapIndex("instance-abc", dpCMs)
	cpIndex := newTerraformConfigMapIndex("instance-abc", cpCMs)

	dpIndex.merge(cpIndex)

	require.Len(dpIndex.planPreviewByResource, 2)
	require.Len(dpIndex.planPreviewByResource["tf-r-dp"], 1)
	require.Len(dpIndex.planPreviewByResource["tf-r-cp"], 1)
}

func TestIsExactInstanceMatch(t *testing.T) {
	tests := []struct {
		name           string
		instanceAndOp  string
		instanceSuffix string
		instanceID     string
		expected       bool
	}{
		{"matches suffix", "bfwiqdagi", "bfwiqdagi", "instance-bfwiqdagi", true},
		{"matches full ID", "instance-abc", "abc", "instance-abc", true},
		{"has op suffix", "bfwiqdagi-op123", "bfwiqdagi", "instance-bfwiqdagi", false},
		{"different instance", "xyz123", "abc", "instance-abc", false},
		{"empty", "", "", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.expected, isExactInstanceMatch(tt.instanceAndOp, tt.instanceSuffix, tt.instanceID))
		})
	}
}

func TestNewTerraformConfigMapIndex_PlanPreviewMultiOpCM(t *testing.T) {
	require := require.New(t)

	// Format 2: multi-operation CM with operation IDs in data keys
	configMaps := []corev1.ConfigMap{
		{
			ObjectMeta: metav1.ObjectMeta{Name: "tf-plan-tf-r-zpo5rklwsc-instance-bfwiqdagi"},
			Data: map[string]string{
				"807196bfc80f676658519a223477b575d5d56821960e873877463fa358b3cc80.e78a354252039003-plan-preview": `{"format_version":"1.2","planned_values":{}}`,
				"aaa111.bbb222-plan-preview":       `{"planned_values":{"outputs":{}}}`,
				"aaa111.bbb222-plan-preview-error":  "Warning: partial",
			},
		},
		{
			// Format 1 for a different resource (should still work)
			ObjectMeta: metav1.ObjectMeta{Name: "tf-plan-tf-r-other-instance-bfwiqdagi-op999"},
			Data: map[string]string{
				"plan-preview": `{"format1":"data"}`,
			},
		},
	}

	index := newTerraformConfigMapIndex("instance-bfwiqdagi", configMaps)

	// Format 2 CM should be indexed under planPreviewMultiByResource
	require.Len(index.planPreviewMultiByResource, 1)
	require.Len(index.planPreviewMultiByResource["tf-r-zpo5rklwsc"], 1)

	// Format 1 CM should be indexed under planPreviewByResource
	require.Len(index.planPreviewByResource, 1)
	require.Len(index.planPreviewByResource["tf-r-other"], 1)
	require.Equal("op999", index.planPreviewByResource["tf-r-other"][0].opSuffix)
}

func TestPlanPreviewsForResource_MultiOpCM(t *testing.T) {
	require := require.New(t)

	// Format 2: multi-operation CM
	configMaps := []corev1.ConfigMap{
		{
			ObjectMeta: metav1.ObjectMeta{Name: "tf-plan-tf-r-abc-instance-xyz"},
			Data: map[string]string{
				"op1.nonce1-plan-preview":       `{"format_version":"1.2","planned_values":{}}`,
				"op2.nonce2-plan-preview-error":  "Error: timeout",
				"op3.nonce3-plan-preview":        `{"planned_values":{"outputs":{}}}`,
				"op3.nonce3-plan-preview-error":  "Warning: partial failure",
			},
		},
	}

	index := newTerraformConfigMapIndex("instance-xyz", configMaps)
	previews, previewErrors := index.planPreviewsForResource("r-abc")

	require.Len(previews, 2)
	require.Equal(`{"format_version":"1.2","planned_values":{}}`, previews["op1.nonce1"])
	require.Equal(`{"planned_values":{"outputs":{}}}`, previews["op3.nonce3"])

	require.Len(previewErrors, 2)
	require.Equal("Error: timeout", previewErrors["op2.nonce2"])
	require.Equal("Warning: partial failure", previewErrors["op3.nonce3"])
}

func TestPlanPreviewsForResource_MixedFormats(t *testing.T) {
	require := require.New(t)

	// Both Format 1 and Format 2 CMs for the same resource
	configMaps := []corev1.ConfigMap{
		{
			// Format 1: per-operation CM
			ObjectMeta: metav1.ObjectMeta{Name: "tf-plan-tf-r-abc-instance-xyz-op111"},
			Data: map[string]string{
				"plan-preview": `{"format1":"op111"}`,
			},
		},
		{
			// Format 2: multi-operation CM
			ObjectMeta: metav1.ObjectMeta{Name: "tf-plan-tf-r-abc-instance-xyz"},
			Data: map[string]string{
				"op222.nonce222-plan-preview": `{"format2":"op222"}`,
			},
		},
	}

	index := newTerraformConfigMapIndex("instance-xyz", configMaps)
	previews, previewErrors := index.planPreviewsForResource("r-abc")

	require.Len(previews, 2)
	require.Equal(`{"format1":"op111"}`, previews["op111"])
	require.Equal(`{"format2":"op222"}`, previews["op222.nonce222"])
	require.Empty(previewErrors)
}

func TestMerge_PlanPreviewMultiByResource(t *testing.T) {
	require := require.New(t)

	dpCMs := []corev1.ConfigMap{
		{
			ObjectMeta: metav1.ObjectMeta{Name: "tf-plan-tf-r-dp-instance-abc"},
			Data: map[string]string{
				"op1-plan-preview": `{"dp":"plan"}`,
			},
		},
	}
	cpCMs := []corev1.ConfigMap{
		{
			ObjectMeta: metav1.ObjectMeta{Name: "tf-plan-tf-r-cp-instance-abc"},
			Data: map[string]string{
				"op2-plan-preview": `{"cp":"plan"}`,
			},
		},
	}

	dpIndex := newTerraformConfigMapIndex("instance-abc", dpCMs)
	cpIndex := newTerraformConfigMapIndex("instance-abc", cpCMs)

	dpIndex.merge(cpIndex)

	require.Len(dpIndex.planPreviewMultiByResource, 2)
	require.Len(dpIndex.planPreviewMultiByResource["tf-r-dp"], 1)
	require.Len(dpIndex.planPreviewMultiByResource["tf-r-cp"], 1)
}

// TestPlanPreviewsForResource_MultiOpCM_DifferentInstance verifies that a Format 2 CM
// belonging to a different instance is not indexed.
func TestPlanPreviewsForResource_MultiOpCM_DifferentInstance(t *testing.T) {
	require := require.New(t)

	configMaps := []corev1.ConfigMap{
		{
			ObjectMeta: metav1.ObjectMeta{Name: "tf-plan-tf-r-abc-instance-other"},
			Data: map[string]string{
				"op1-plan-preview": `{"other":"instance"}`,
			},
		},
	}

	index := newTerraformConfigMapIndex("instance-xyz", configMaps)
	previews, previewErrors := index.planPreviewsForResource("r-abc")

	require.Empty(previews)
	require.Empty(previewErrors)
}
