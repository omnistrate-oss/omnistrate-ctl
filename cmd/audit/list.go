package audit

import (
	"context"
	"strings"
	"time"

	"github.com/omnistrate-oss/omnistrate-ctl/cmd/common"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/dataaccess"
	"github.com/omnistrate-oss/omnistrate-ctl/internal/utils"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List audit events",
	Long:  "List audit events with detailed filtering options for services, instances, and time ranges.",
	RunE:  runList,
}

func init() {
	listCmd.Flags().String("next-page-token", "", "Token for next page of results")
	listCmd.Flags().Int64("page-size", 0, "Number of results per page")
	listCmd.Flags().StringP("service-id", "s", "", "Service ID to filter by")
	listCmd.Flags().StringP("environment-type", "e", "", "Environment type to filter by")
	listCmd.Flags().StringSlice("event-source-types", []string{}, "Event source types to filter by (comma-separated)")
	listCmd.Flags().StringP("instance-id", "i", "", "Instance ID to filter by")
	listCmd.Flags().String("product-tier-id", "", "Product tier ID to filter by")
	listCmd.Flags().String("start-date", "", "Start date for events (RFC3339 format)")
	listCmd.Flags().String("end-date", "", "End date for events (RFC3339 format)")
}

func runList(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	nextPageToken, _ := cmd.Flags().GetString("next-page-token")
	pageSize, _ := cmd.Flags().GetInt64("page-size")
	serviceID, _ := cmd.Flags().GetString("service-id")
	environmentType, _ := cmd.Flags().GetString("environment-type")
	eventSourceTypes, _ := cmd.Flags().GetStringSlice("event-source-types")
	instanceID, _ := cmd.Flags().GetString("instance-id")
	productTierID, _ := cmd.Flags().GetString("product-tier-id")
	startDateStr, _ := cmd.Flags().GetString("start-date")
	endDateStr, _ := cmd.Flags().GetString("end-date")

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

	// Clean up event source types
	var cleanEventSourceTypes []string
	for _, eventType := range eventSourceTypes {
		if strings.TrimSpace(eventType) != "" {
			cleanEventSourceTypes = append(cleanEventSourceTypes, strings.TrimSpace(eventType))
		}
	}

	token, err := common.GetTokenWithLogin()
	if err != nil {
		return err
	}

	opts := &dataaccess.ListAuditEventsOptions{
		NextPageToken:    nextPageToken,
		ServiceID:        serviceID,
		EnvironmentType:  environmentType,
		EventSourceTypes: cleanEventSourceTypes,
		InstanceID:       instanceID,
		ProductTierID:    productTierID,
		StartDate:        startDate,
		EndDate:          endDate,
	}

	if pageSize > 0 {
		opts.PageSize = &pageSize
	}

	result, err := dataaccess.ListAuditEvents(ctx, token, opts)
	if err != nil {
		return err
	}

	output, _ := cmd.Flags().GetString("output")
	return utils.PrintTextTableJsonArrayOutput(output, []interface{}{result})
}