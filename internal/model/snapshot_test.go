package model

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSnapshotDetailJSONTags(t *testing.T) {
	detail := SnapshotDetail{
		SnapshotID:       "instance-ss-f2qlfhnvv",
		Status:           "COMPLETED",
		Region:           "us-east-1",
		SnapshotType:     "manual",
		Progress:         "100%",
		CreatedAt:        "2024-01-15T10:30:00Z",
		CompletedAt:      "2024-01-15T10:45:00Z",
		SourceInstanceID: "instance-abcd",
		ProductTierID:    "pt-123",
		ProductTierVer:   "1.0",
		Encrypted:        true,
		SnapshotMetadata: `{"backupId":"20260619T133609"}`,
	}

	data, err := json.Marshal(detail)
	require.NoError(t, err)

	assert.JSONEq(t, `{
		"snapshotId": "instance-ss-f2qlfhnvv",
		"status": "COMPLETED",
		"region": "us-east-1",
		"snapshotType": "manual",
		"progress": "100%",
		"createdAt": "2024-01-15T10:30:00Z",
		"completedAt": "2024-01-15T10:45:00Z",
		"sourceInstanceId": "instance-abcd",
		"productTierId": "pt-123",
		"productTierVersion": "1.0",
		"encrypted": true,
		"snapshotMetadata": "{\"backupId\":\"20260619T133609\"}"
	}`, string(data))
}

func TestSnapshotDetailOmitsEmptyMetadata(t *testing.T) {
	data, err := json.Marshal(SnapshotDetail{
		SnapshotID: "instance-ss-no-metadata",
	})
	require.NoError(t, err)

	assert.NotContains(t, string(data), "snapshotMetadata")
}
