package model

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDeploymentCell_ToTableView_PrivateLink(t *testing.T) {
	tests := []struct {
		name               string
		privateLinkEnabled *bool
		wantPrivateLink    string
	}{
		{
			name:               "nil renders dash",
			privateLinkEnabled: nil,
			wantPrivateLink:    "-",
		},
		{
			name:               "true renders Yes",
			privateLinkEnabled: ptr(true),
			wantPrivateLink:    "Yes",
		},
		{
			name:               "false renders No",
			privateLinkEnabled: ptr(false),
			wantPrivateLink:    "No",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dc := DeploymentCell{
				Key:                "dataplane-test",
				ID:                 "dc-123",
				Status:             "READY",
				Type:               "managed",
				CloudProvider:      "aws",
				Region:             "us-east-1",
				PrivateLinkEnabled: tt.privateLinkEnabled,
			}
			view := dc.ToTableView()
			assert.Equal(t, tt.wantPrivateLink, view.PrivateLink)
		})
	}
}

func TestDeploymentCell_ToTableView_AdoptedClusterUsesKeyAsID(t *testing.T) {
	dc := DeploymentCell{
		Key:    "custom-key-abc",
		ID:     "original-id",
		Status: "READY",
	}
	view := dc.ToTableView()
	assert.Equal(t, "custom-key-abc", view.ID, "adopted clusters (non-dataplane- key) should use Key as ID")
}

func TestDeploymentCell_ToDescribeTableView_ManagedArtifactVersions(t *testing.T) {
	currentVersion := "r0000020"
	effectiveVersion := "r0000021"
	dc := DeploymentCell{
		Key:                             "dataplane-test",
		ID:                              "dc-123",
		CurrentManagedArtifactVersion:   &currentVersion,
		EffectiveManagedArtifactVersion: &effectiveVersion,
	}

	view := dc.ToDescribeTableView()
	assert.Equal(t, &currentVersion, view.CurrentManagedArtifactVersion)
	assert.Equal(t, &effectiveVersion, view.EffectiveManagedArtifactVersion)
}

func TestDeploymentCell_JSONOmitsMissingManagedArtifactVersions(t *testing.T) {
	data, err := json.Marshal(DeploymentCell{ID: "dc-123"})
	assert.NoError(t, err)
	assert.NotContains(t, string(data), "current_managed_artifact_version")
	assert.NotContains(t, string(data), "effective_managed_artifact_version")
}

func ptr[T any](v T) *T {
	return &v
}
