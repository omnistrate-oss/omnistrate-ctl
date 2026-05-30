package instance

import (
	"testing"

	openapiclientfleet "github.com/omnistrate-oss/omnistrate-sdk-go/fleet"
	"github.com/stretchr/testify/assert"
)

func TestNewSnapshotDetailIncludesSnapshotMetadata(t *testing.T) {
	snapshot := openapiclientfleet.FleetDescribeInstanceSnapshotResult{
		AdditionalProperties: map[string]interface{}{
			"snapshotMetadata": map[string]interface{}{
				"backupId":   "20260530T080000",
				"backupName": "backup-20260530080000",
			},
		},
	}
	snapshot.SetSnapshotId("snapshot-123")
	snapshot.SetStatus("Complete")

	detail := newSnapshotDetail(snapshot)

	assert.Equal(t, "snapshot-123", detail.SnapshotID)
	assert.Equal(t, "Complete", detail.Status)
	assert.Equal(t, map[string]interface{}{
		"backupId":   "20260530T080000",
		"backupName": "backup-20260530080000",
	}, detail.SnapshotMetadata)
}
