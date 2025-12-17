package instance

import (
	"errors"
	"fmt"
	"time"

	"github.com/chelnak/ysmrr"
	"github.com/omnistrate-oss/omnistrate-ctl/cmd/common"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/config"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/dataaccess"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/utils"
	"github.com/spf13/cobra"
)

const (
	listSnapshotsExample = `# List snapshots for an instance
omnistrate-ctl instance list-snapshots instance-abcd1234"`
)

var listSnapshotsCmd = &cobra.Command{
	Use:          "list-snapshots [instance-id]",
	Short:        "List all snapshots for an instance",
	Long:         `This command helps you list all snapshots available for your instance.`,
	Example:      listSnapshotsExample,
	RunE:         runListSnapshots,
	SilenceUsage: true,
}

func init() {
	listSnapshotsCmd.Args = cobra.ExactArgs(1) // Require exactly one argument
}

const snapshotDisplayTimeLayout = "2006-01-02 15:04:05 MST"

type SnapshotDetail struct {
	SnapshotID       string `json:"snapshotId"`
	Status           string `json:"status"`
	Region           string `json:"region"`
	SnapshotType     string `json:"snapshotType"`
	Progress         string `json:"progress"`
	CreatedAt        string `json:"createdAt"`
	CompletedAt      string `json:"completedAt"`
	SourceInstanceID string `json:"sourceInstanceId"`
	ProductTierID    string `json:"productTierId"`
	ProductTierVer   string `json:"productTierVersion"`
	Encrypted        bool   `json:"encrypted"`
}

func runListSnapshots(cmd *cobra.Command, args []string) error {
	defer config.CleanupArgsAndFlags(cmd, &args)

	if len(args) == 0 {
		err := errors.New("instance id is required")
		utils.PrintError(err)
		return err
	}

	// Retrieve args
	instanceID := args[0]

	// Retrieve flags
	output, err := cmd.Flags().GetString("output")
	if err != nil {
		utils.PrintError(err)
		return err
	}

	// Validate user login
	token, err := common.GetTokenWithLogin()
	if err != nil {
		utils.PrintError(err)
		return err
	}

	// Initialize spinner if output is not JSON
	var sm ysmrr.SpinnerManager
	var spinner *ysmrr.Spinner
	if output != "json" {
		sm = ysmrr.NewSpinnerManager()
		msg := "Listing snapshots..."
		spinner = sm.AddSpinner(msg)
		sm.Start()
	}

	// Check if instance exists and get its details
	serviceID, environmentID, _, _, err := getInstance(cmd.Context(), token, instanceID)
	if err != nil {
		utils.HandleSpinnerError(spinner, sm, err)
		return err
	}

	// List snapshots
	result, err := dataaccess.ListResourceInstanceSnapshots(cmd.Context(), token, serviceID, environmentID, instanceID)
	if err != nil {
		utils.HandleSpinnerError(spinner, sm, err)
		return err
	}

	utils.HandleSpinnerSuccess(spinner, sm, "Successfully listed snapshots")

	if output == "json" {
		return utils.PrintTextTableJsonOutput(output, result)
	}

	if result == nil || len(result.Snapshots) == 0 {
		utils.PrintInfo(fmt.Sprintf("No snapshots found for instance %s.", instanceID))
		return nil
	}

	summaries := make([]SnapshotDetail, 0, len(result.Snapshots))
	for _, snapshot := range result.Snapshots {
		summaries = append(summaries, SnapshotDetail{
			SnapshotID:       snapshot.SnapshotId,
			Status:           snapshot.Status,
			Region:           snapshot.Region,
			SnapshotType:     snapshot.SnapshotType,
			Progress:         fmt.Sprintf("%d%%", snapshot.Progress),
			CreatedAt:        formatSnapshotDisplayTime(snapshot.CreatedTime),
			CompletedAt:      formatSnapshotDisplayTime(snapshot.CompleteTime),
			SourceInstanceID: snapshot.SourceInstanceId,
			ProductTierID:    snapshot.ProductTierId,
			ProductTierVer:   snapshot.ProductTierVersion,
			Encrypted:        snapshot.Encrypted,
		})
	}

	return utils.PrintTextTableJsonArrayOutput(output, summaries)
}

func formatSnapshotDisplayTime(raw string) string {
	if raw == "" {
		return ""
	}

	parsed, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		return raw
	}

	return parsed.UTC().Format(snapshotDisplayTimeLayout)
}
