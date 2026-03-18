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
		clientset := fake.NewSimpleClientset(objects...)
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
