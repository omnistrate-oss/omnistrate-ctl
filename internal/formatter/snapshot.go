package formatter

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/omnistrate-oss/omnistrate-ctl/internal/model"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/utils"
	openapiclientfleet "github.com/omnistrate-oss/omnistrate-sdk-go/fleet"
)

func FormatSnapshotSummaries(snapshots []openapiclientfleet.FleetDescribeInstanceSnapshotResult) []model.SnapshotDetail {
	summaries := make([]model.SnapshotDetail, 0, len(snapshots))
	for _, snapshot := range snapshots {
		summary := model.SnapshotDetail{
			SnapshotID:       utils.FromPtr(snapshot.SnapshotId),
			Status:           utils.FromPtr(snapshot.Status),
			Region:           utils.FromPtr(snapshot.Region),
			SnapshotType:     utils.FromPtr(snapshot.SnapshotType),
			Progress:         fmt.Sprintf("%d%%", utils.FromPtr(snapshot.Progress)),
			CreatedAt:        FormatSnapshotDisplayTime(utils.FromPtr(snapshot.CreatedTime)),
			CompletedAt:      FormatSnapshotDisplayTime(utils.FromPtr(snapshot.CompleteTime)),
			SourceInstanceID: utils.FromPtr(snapshot.SourceInstanceId),
			ProductTierID:    utils.FromPtr(snapshot.ProductTierId),
			ProductTierVer:   utils.FromPtr(snapshot.ProductTierVersion),
			Encrypted:        utils.FromPtr(snapshot.Encrypted),
		}
		if len(snapshot.SnapshotMetadata) > 0 {
			summary.SnapshotMetadata = FormatSnapshotMetadata(snapshot.SnapshotMetadata)
		}
		summaries = append(summaries, summary)
	}
	return summaries
}

func FormatSnapshotMetadata(metadata map[string]interface{}) string {
	if len(metadata) == 0 {
		return ""
	}

	data, err := json.Marshal(metadata)
	if err != nil {
		return fmt.Sprintf("%v", metadata)
	}
	return string(data)
}

func FormatSnapshotDisplayTime(raw string) string {
	if raw == "" {
		return ""
	}

	parsed, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		return raw
	}

	return parsed.UTC().Format(time.RFC3339)
}
