package operations

import (
	"context"
	"strings"
	"time"

	"github.com/omnistrate-oss/omnistrate-ctl/cmd/common"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/dataaccess"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/utils"
	"github.com/spf13/cobra"
)

var eventsCmd = &cobra.Command{
	Use:   "events",
	Short: "List operational events",
	Long:  "List operational events for services, environments, and instances with filtering options.",
	RunE:  runEvents,
}

func init() {
	eventsCmd.Flags().String("next-page-token", "", "Token for next page of results")
	eventsCmd.Flags().Int64("page-size", 0, "Number of results per page")
	eventsCmd.Flags().StringP("environment-type", "e", "", "Environment type to filter by")
	eventsCmd.Flags().StringSlice("event-types", []string{}, "Event types to filter by (comma-separated)")
	eventsCmd.Flags().StringP("service-id", "s", "", "Service ID to list events for")
	eventsCmd.Flags().String("service-environment-id", "", "Service environment ID to list events for")
	eventsCmd.Flags().StringP("instance-id", "i", "", "Instance ID to list events for")
	eventsCmd.Flags().String("start-date", "", "Start date of events (RFC3339 format)")
	eventsCmd.Flags().String("end-date", "", "End date of events (RFC3339 format)")
	eventsCmd.Flags().String("product-tier-id", "", "Product tier ID to filter by")
}

func runEvents(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	nextPageToken, _ := cmd.Flags().GetString("next-page-token")
	pageSize, _ := cmd.Flags().GetInt64("page-size")
	environmentType, _ := cmd.Flags().GetString("environment-type")
	eventTypes, _ := cmd.Flags().GetStringSlice("event-types")
	serviceID, _ := cmd.Flags().GetString("service-id")
	serviceEnvironmentID, _ := cmd.Flags().GetString("service-environment-id")
	instanceID, _ := cmd.Flags().GetString("instance-id")
	startDateStr, _ := cmd.Flags().GetString("start-date")
	endDateStr, _ := cmd.Flags().GetString("end-date")
	productTierID, _ := cmd.Flags().GetString("product-tier-id")

	// Parse dates
	var startDate, endDate *time.Time
	if startDateStr != "" {
		parsed, err := time.Parse(time.RFC3339, startDateStr)
		if err != nil {
			return err
		}
		startDate = &parsed
	}
	if endDateStr != "" {
		parsed, err := time.Parse(time.RFC3339, endDateStr)
		if err != nil {
			return err
		}
		endDate = &parsed
	}

	// Clean up event types
	var cleanEventTypes []string
	for _, eventType := range eventTypes {
		if strings.TrimSpace(eventType) != "" {
			cleanEventTypes = append(cleanEventTypes, strings.TrimSpace(eventType))
		}
	}

	token, err := common.GetTokenWithLogin()
	if err != nil {
		return err
	}

	opts := &dataaccess.ListEventsOptions{
		NextPageToken:        nextPageToken,
		EnvironmentType:      environmentType,
		EventTypes:           cleanEventTypes,
		ServiceID:            serviceID,
		ServiceEnvironmentID: serviceEnvironmentID,
		InstanceID:           instanceID,
		StartDate:            startDate,
		EndDate:              endDate,
		ProductTierID:        productTierID,
	}

	if pageSize > 0 {
		opts.PageSize = &pageSize
	}

	result, err := dataaccess.ListOperationsEvents(ctx, token, opts)
	if err != nil {
		return err
	}

	output, _ := cmd.Flags().GetString("output")
	return utils.PrintTextTableJsonArrayOutput(output, []interface{}{result})
}
