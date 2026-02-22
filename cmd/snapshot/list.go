package snapshot

import (
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
	listExample = `# List all snapshots for a service environment
omnistrate-ctl snapshot list --service-id service-abcd --environment-id env-1234`
)

var listCmd = &cobra.Command{
	Use:          "list --service-id <service-id> --environment-id <environment-id>",
	Short:        "List all snapshots for a service environment",
	Long:         `This command helps you list all snapshots available across all instances in a service environment.`,
	Example:      listExample,
	RunE:         runList,
	SilenceUsage: true,
}

func init() {
	listCmd.Args = cobra.NoArgs
	listCmd.Flags().String("service-id", "", "The ID of the service (required)")
	listCmd.Flags().String("environment-id", "", "The ID of the environment (required)")

	if err := listCmd.MarkFlagRequired("service-id"); err != nil {
		return
	}
	if err := listCmd.MarkFlagRequired("environment-id"); err != nil {
		return
	}
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

func runList(cmd *cobra.Command, args []string) error {
	defer config.CleanupArgsAndFlags(cmd, &args)

	serviceID, err := cmd.Flags().GetString("service-id")
	if err != nil {
		utils.PrintError(err)
		return err
	}

	environmentID, err := cmd.Flags().GetString("environment-id")
	if err != nil {
		utils.PrintError(err)
		return err
	}

	output, err := cmd.Flags().GetString("output")
	if err != nil {
		utils.PrintError(err)
		return err
	}

	token, err := common.GetTokenWithLogin()
	if err != nil {
		utils.PrintError(err)
		return err
	}

	var sm ysmrr.SpinnerManager
	var spinner *ysmrr.Spinner
	if output != "json" {
		sm = ysmrr.NewSpinnerManager()
		spinner = sm.AddSpinner("Listing snapshots...")
		sm.Start()
	}

	result, err := dataaccess.ListAllSnapshots(cmd.Context(), token, serviceID, environmentID)
	if err != nil {
		utils.HandleSpinnerError(spinner, sm, err)
		return err
	}

	utils.HandleSpinnerSuccess(spinner, sm, "Successfully listed snapshots")

	if output == "json" {
		return utils.PrintTextTableJsonOutput(output, result)
	}

	if result == nil || len(result.Snapshots) == 0 {
		utils.PrintInfo("No snapshots found.")
		return nil
	}

	summaries := make([]SnapshotDetail, 0, len(result.Snapshots))
	for _, snap := range result.Snapshots {
		summaries = append(summaries, SnapshotDetail{
			SnapshotID:       utils.FromPtr(snap.SnapshotId),
			Status:           utils.FromPtr(snap.Status),
			Region:           utils.FromPtr(snap.Region),
			SnapshotType:     utils.FromPtr(snap.SnapshotType),
			Progress:         fmt.Sprintf("%d%%", utils.FromPtr(snap.Progress)),
			CreatedAt:        formatSnapshotDisplayTime(utils.FromPtr(snap.CreatedTime)),
			CompletedAt:      formatSnapshotDisplayTime(utils.FromPtr(snap.CompleteTime)),
			SourceInstanceID: utils.FromPtr(snap.SourceInstanceId),
			ProductTierID:    utils.FromPtr(snap.ProductTierId),
			ProductTierVer:   utils.FromPtr(snap.ProductTierVersion),
			Encrypted:        utils.FromPtr(snap.Encrypted),
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
