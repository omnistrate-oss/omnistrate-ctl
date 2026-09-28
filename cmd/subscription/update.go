package subscription

import (
	"fmt"

	"github.com/omnistrate-oss/omnistrate-ctl/cmd/common"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/config"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/dataaccess"
	"github.com/spf13/cobra"
)

var updateCmd = &cobra.Command{
	Use:   "update <subscription-id>",
	Short: "Update a subscription",
	Long:  "Update a subscription for a service environment.",
	Args:  cobra.ExactArgs(1),
	RunE:  runUpdate,
}

func init() {
	updateCmd.Flags().StringP("service-id", "s", "", "Service ID (required)")
	updateCmd.Flags().StringP("environment-id", "e", "", "Environment ID (required)")
	updateCmd.Flags().String(allowedDeploymentLocationsFlag, "", "Subscription deployment location restriction as a JSON array. Set to [] to inherit product tier deployment locations")

	_ = updateCmd.MarkFlagRequired("service-id")
	_ = updateCmd.MarkFlagRequired("environment-id")
	_ = updateCmd.MarkFlagRequired(allowedDeploymentLocationsFlag)
}

func runUpdate(cmd *cobra.Command, args []string) error {
	defer config.CleanupArgsAndFlags(cmd, &args)

	subscriptionID := args[0]
	serviceID, _ := cmd.Flags().GetString("service-id")
	environmentID, _ := cmd.Flags().GetString("environment-id")
	allowedDeploymentLocationsStr, _ := cmd.Flags().GetString(allowedDeploymentLocationsFlag)

	allowedDeploymentLocations, err := parseAllowedDeploymentLocations(allowedDeploymentLocationsStr)
	if err != nil {
		return err
	}

	token, err := common.GetTokenWithLogin()
	if err != nil {
		return err
	}

	err = dataaccess.UpdateSubscription(cmd.Context(), token, serviceID, environmentID, subscriptionID, &dataaccess.UpdateSubscriptionOptions{
		AllowedDeploymentLocations: allowedDeploymentLocations,
	})
	if err != nil {
		return err
	}

	fmt.Printf("Successfully updated subscription %s\n", subscriptionID)
	return nil
}
