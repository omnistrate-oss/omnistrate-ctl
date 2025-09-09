package subscription

import (
	"context"
	"fmt"

	"github.com/omnistrate-oss/omnistrate-ctl/cmd/common"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/dataaccess"
	"github.com/spf13/cobra"
)

var resumeCmd = &cobra.Command{
	Use:   "resume <subscription-id>",
	Short: "Resume a suspended subscription",
	Long:  "Resume a previously suspended subscription for a service environment.",
	Args:  cobra.ExactArgs(1),
	RunE:  runResume,
}

func init() {
	resumeCmd.Flags().StringP("service-id", "s", "", "Service ID (required)")
	resumeCmd.Flags().StringP("environment-id", "e", "", "Environment ID (required)")

	_ = resumeCmd.MarkFlagRequired("service-id")
	_ = resumeCmd.MarkFlagRequired("environment-id")
}

func runResume(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	subscriptionID := args[0]
	serviceID, _ := cmd.Flags().GetString("service-id")
	environmentID, _ := cmd.Flags().GetString("environment-id")

	token, err := common.GetTokenWithLogin()
	if err != nil {
		return err
	}

	err = dataaccess.ResumeSubscription(ctx, token, serviceID, environmentID, subscriptionID)
	if err != nil {
		return err
	}

	fmt.Printf("Successfully resumed subscription %s\n", subscriptionID)
	return nil
}