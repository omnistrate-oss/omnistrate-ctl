package cost

import (
	"fmt"
	"strings"
	"time"

	"github.com/omnistrate-oss/omnistrate-ctl/internal/dataaccess"
	openapiclientfleet "github.com/omnistrate-oss/omnistrate-sdk-go/fleet"
	"github.com/spf13/cobra"
)

// Common flag names
const (
	flagStartDate   = "start-date"
	flagEndDate     = "end-date"
	flagEnvironment = "environment"
	flagFrequency   = "frequency"
	flagTopN        = "top"
	flagOutput      = "output"
)

// addCommonTimeFlags adds common time-related flags to a command
func addCommonTimeFlags(cmd *cobra.Command) {
	cmd.Flags().String(flagStartDate, "", "Start date for cost analysis (RFC3339 format, e.g., 2024-01-01T00:00:00Z)")
	cmd.Flags().String(flagEndDate, "", "End date for cost analysis (RFC3339 format, e.g., 2024-01-31T23:59:59Z)")
	cmd.Flags().StringP(flagEnvironment, "e", "", "Environment type (valid: dev, qa, staging, canary, prod, private)")
	cmd.Flags().StringP(flagFrequency, "f", "daily", "Frequency of cost data (daily, weekly, monthly)")

	_ = cmd.MarkFlagRequired(flagStartDate)
	_ = cmd.MarkFlagRequired(flagEndDate)
	_ = cmd.MarkFlagRequired(flagEnvironment)
}

// parseTimeFlags parses common time flags from command
func parseTimeFlags(cmd *cobra.Command) (startDate, endDate time.Time, environmentType, frequency string, err error) {
	startDateStr, _ := cmd.Flags().GetString(flagStartDate)
	endDateStr, _ := cmd.Flags().GetString(flagEndDate)
	environmentType, _ = cmd.Flags().GetString(flagEnvironment)
	frequency, _ = cmd.Flags().GetString(flagFrequency)

	if startDateStr != "" {
		startDate, err = time.Parse(time.RFC3339, startDateStr)
		if err != nil {
			return time.Time{}, time.Time{}, "", "", fmt.Errorf("invalid start-date format, expected RFC3339 (e.g., '2006-01-02T15:04:05Z07:00'): %w", err)
		}
	}

	if endDateStr != "" {
		endDate, err = time.Parse(time.RFC3339, endDateStr)
		if err != nil {
			return time.Time{}, time.Time{}, "", "", fmt.Errorf("invalid end-date format, expected RFC3339 (e.g., '2006-01-02T15:04:05Z07:00'): %w", err)
		}
	}

	// Environment type must be uppercase for the API
	environmentType = strings.ToUpper(environmentType)

	return startDate, endDate, environmentType, frequency, nil
}

// buildCostOptions creates CostOptions from parsed flags
func buildCostOptions(startDate, endDate time.Time, environmentType, frequency string) dataaccess.CostOptions {
	return dataaccess.CostOptions{
		StartDate:       startDate,
		EndDate:         endDate,
		EnvironmentType: environmentType,
		Frequency:       frequency,
	}
}

// aggregateInstanceTypeCosts aggregates cost data by instance type across all deployment cells
func aggregateInstanceTypeCosts(result *openapiclientfleet.DescribeDeploymentCellCostResult) map[string]*InstanceTypeAggregate {
	aggregates := make(map[string]*InstanceTypeAggregate)

	if result == nil || result.DeploymentCellCosts == nil {
		return aggregates
	}

	for _, cellCost := range *result.DeploymentCellCosts {
		if cellCost.InstancesCost == nil {
			continue
		}

		for _, instanceCost := range cellCost.InstancesCost {
			if instanceCost.CostByInstanceType == nil {
				continue
			}

			for instanceType, typeCost := range *instanceCost.CostByInstanceType {
				if agg, exists := aggregates[instanceType]; exists {
					agg.NumVMs += typeCost.NumVMs
					agg.TotalCost += typeCost.TotalCost
					agg.TotalUptimeHours += typeCost.TotalUptimeHours
					agg.NumInstances++
				} else {
					aggregates[instanceType] = &InstanceTypeAggregate{
						InstanceType:     instanceType,
						NumVMs:           typeCost.NumVMs,
						TotalCost:        typeCost.TotalCost,
						TotalUptimeHours: typeCost.TotalUptimeHours,
						NumInstances:     1,
					}
				}
			}
		}
	}

	return aggregates
}

// InstanceTypeAggregate represents aggregated cost data for an instance type
type InstanceTypeAggregate struct {
	InstanceType     string  `json:"instanceType"`
	NumVMs           int64   `json:"numVMs"`
	TotalCost        float64 `json:"totalCost"`
	TotalUptimeHours float64 `json:"totalUptimeHours"`
	NumInstances     int     `json:"numInstances"` // Number of instances using this type
}

// aggregateInstanceCosts collects all instance costs from deployment cells
func aggregateInstanceCosts(result *openapiclientfleet.DescribeDeploymentCellCostResult) []openapiclientfleet.PerInstanceCost {
	var instances []openapiclientfleet.PerInstanceCost

	if result == nil || result.DeploymentCellCosts == nil {
		return instances
	}

	for _, cellCost := range *result.DeploymentCellCosts {
		if cellCost.InstancesCost == nil {
			continue
		}
		instances = append(instances, cellCost.InstancesCost...)
	}

	return instances
}
