package formatter

import (
	"encoding/json"
	"testing"

	openapiclientfleet "github.com/omnistrate-oss/omnistrate-sdk-go/fleet"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFormatSnapshotDisplayTime(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"valid RFC3339", "2024-01-15T10:30:00Z", "2024-01-15 10:30:00 UTC"},
		{"empty string", "", ""},
		{"invalid format returns raw", "not-a-date", "not-a-date"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatSnapshotDisplayTime(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFormatSnapshotSummariesIncludesMetadataWhenPresent(t *testing.T) {
	summaries := FormatSnapshotSummaries([]openapiclientfleet.FleetDescribeInstanceSnapshotResult{
		{
			SnapshotId: ptr("instance-ss-f2qlfhnvv"),
			SnapshotMetadata: map[string]interface{}{
				"backupId":   "20260619T133609",
				"backupName": "backup-20260619133609",
			},
		},
		{
			SnapshotId: ptr("instance-ss-no-metadata"),
		},
	})

	require.Len(t, summaries, 2)
	assert.Equal(t, `{"backupId":"20260619T133609","backupName":"backup-20260619133609"}`, summaries[0].SnapshotMetadata)

	withMetadata, err := json.Marshal(summaries[0])
	require.NoError(t, err)
	assert.Contains(t, string(withMetadata), "snapshotMetadata")
	assert.Contains(t, string(withMetadata), "backupId")
	assert.NotContains(t, string(withMetadata), "map[")

	withoutMetadata, err := json.Marshal(summaries[1])
	require.NoError(t, err)
	assert.NotContains(t, string(withoutMetadata), "snapshotMetadata")
}

func ptr[T any](value T) *T {
	return &value
}
