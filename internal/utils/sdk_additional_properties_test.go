package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSnapshotMetadataFromSDKUsesTypedField(t *testing.T) {
	model := struct {
		SnapshotMetadata map[string]interface{}
	}{
		SnapshotMetadata: map[string]interface{}{
			"backupId": "backup-typed",
		},
	}

	metadata := SnapshotMetadataFromSDK(model, map[string]interface{}{
		"snapshotMetadata": map[string]interface{}{
			"backupId": "backup-additional",
		},
	})

	assert.Equal(t, map[string]interface{}{"backupId": "backup-typed"}, metadata)
}

func TestSnapshotMetadataFromSDKFallsBackToAdditionalProperties(t *testing.T) {
	metadata := SnapshotMetadataFromSDK(struct{}{}, map[string]interface{}{
		"snapshotMetadata": map[string]interface{}{
			"backupId":   "20260530T080000",
			"backupName": "backup-20260530080000",
		},
	})

	assert.Equal(t, map[string]interface{}{
		"backupId":   "20260530T080000",
		"backupName": "backup-20260530080000",
	}, metadata)
}

func TestSnapshotMetadataFromSDKIgnoresUnexpectedShape(t *testing.T) {
	metadata := SnapshotMetadataFromSDK(struct{}{}, map[string]interface{}{
		"snapshotMetadata": "invalid",
	})

	assert.Nil(t, metadata)
}
