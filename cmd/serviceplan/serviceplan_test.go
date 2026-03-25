package serviceplan

import (
	"github.com/omnistrate-oss/omnistrate-ctl/internal/utils"
	openapiclientfleet "github.com/omnistrate-oss/omnistrate-sdk-go/fleet"

	"testing"

	"github.com/stretchr/testify/require"
)

func TestFilterLatestNVersions(t *testing.T) {
	require := require.New(t)

	tests := []struct {
		name         string
		servicePlans []openapiclientfleet.ServicePlanSearchRecord
		latestN      int
		expected     []openapiclientfleet.ServicePlanSearchRecord
	}{
		{
			name: "latestN is -1, return all service plans",
			servicePlans: []openapiclientfleet.ServicePlanSearchRecord{
				{ReleasedAt: utils.ToPtr("2023-08-01T00:00:00Z")},
				{ReleasedAt: utils.ToPtr("2023-07-01T00:00:00Z")},
			},
			latestN: -1,
			expected: []openapiclientfleet.ServicePlanSearchRecord{
				{ReleasedAt: utils.ToPtr("2023-08-01T00:00:00Z")},
				{ReleasedAt: utils.ToPtr("2023-07-01T00:00:00Z")},
			},
		},
		{
			name: "latestN is greater than available plans, return all service plans",
			servicePlans: []openapiclientfleet.ServicePlanSearchRecord{
				{ReleasedAt: utils.ToPtr("2023-08-01T00:00:00Z")},
				{ReleasedAt: utils.ToPtr("2023-07-01T00:00:00Z")},
			},
			latestN: 5,
			expected: []openapiclientfleet.ServicePlanSearchRecord{
				{ReleasedAt: utils.ToPtr("2023-08-01T00:00:00Z")},
				{ReleasedAt: utils.ToPtr("2023-07-01T00:00:00Z")},
			},
		},
		{
			name: "latestN is 1, return only the latest service plan",
			servicePlans: []openapiclientfleet.ServicePlanSearchRecord{
				{ReleasedAt: utils.ToPtr("2023-08-01T00:00:00Z")},
				{ReleasedAt: utils.ToPtr("2023-07-01T00:00:00Z")},
			},
			latestN: 1,
			expected: []openapiclientfleet.ServicePlanSearchRecord{
				{ReleasedAt: utils.ToPtr("2023-08-01T00:00:00Z")},
			},
		},
		{
			name: "latestN is 2, return the latest 2 service plans",
			servicePlans: []openapiclientfleet.ServicePlanSearchRecord{
				{ReleasedAt: utils.ToPtr("2023-08-01T00:00:00Z")},
				{ReleasedAt: utils.ToPtr("2023-07-01T00:00:00Z")},
				{ReleasedAt: utils.ToPtr("2023-06-01T00:00:00Z")},
			},
			latestN: 2,
			expected: []openapiclientfleet.ServicePlanSearchRecord{
				{ReleasedAt: utils.ToPtr("2023-08-01T00:00:00Z")},
				{ReleasedAt: utils.ToPtr("2023-07-01T00:00:00Z")},
			},
		},
		{
			name: "plans with nil release dates, sort properly",
			servicePlans: []openapiclientfleet.ServicePlanSearchRecord{
				{ReleasedAt: utils.ToPtr("2023-08-01T00:00:00Z")},
				{ReleasedAt: utils.ToPtr("2023-07-01T00:00:00Z")},
				{ReleasedAt: nil},
			},
			latestN: 2,
			expected: []openapiclientfleet.ServicePlanSearchRecord{
				{ReleasedAt: utils.ToPtr("2023-08-01T00:00:00Z")},
				{ReleasedAt: utils.ToPtr("2023-07-01T00:00:00Z")},
			},
		},
		{
			name: "mix of plans with and without release dates, return correct latest plans",
			servicePlans: []openapiclientfleet.ServicePlanSearchRecord{
				{ReleasedAt: utils.ToPtr("2023-08-01T00:00:00Z")},
				{ReleasedAt: nil},
				{ReleasedAt: utils.ToPtr("2023-07-01T00:00:00Z")},
			},
			latestN: 2,
			expected: []openapiclientfleet.ServicePlanSearchRecord{
				{ReleasedAt: utils.ToPtr("2023-08-01T00:00:00Z")},
				{ReleasedAt: utils.ToPtr("2023-07-01T00:00:00Z")},
			},
		},
		{
			name: "all plans have nil release dates, return the first N plans",
			servicePlans: []openapiclientfleet.ServicePlanSearchRecord{
				{ReleasedAt: nil},
				{ReleasedAt: nil},
			},
			latestN: 1,
			expected: []openapiclientfleet.ServicePlanSearchRecord{
				{ReleasedAt: nil},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := filterLatestNVersions(tt.servicePlans, tt.latestN)
			require.Equal(len(tt.expected), len(result))
			for i := range tt.expected {
				if tt.expected[i].ReleasedAt == nil {
					require.Nil(result[i].ReleasedAt)
				} else {
					require.NotNil(result[i].ReleasedAt)
					require.Equal(tt.expected[i].ReleasedAt, result[i].ReleasedAt)
				}
			}
		})
	}
}

func TestValidateUpdateVersionNameArguments(t *testing.T) {
	require := require.New(t)

	tests := []struct {
		name      string
		args      []string
		serviceID string
		planID    string
		wantErr   bool
		errMsg    string
	}{
		{
			name:      "valid args - both name and plan provided",
			args:      []string{"service1", "plan1"},
			serviceID: "",
			planID:    "",
			wantErr:   false,
		},
		{
			name:      "valid args - both IDs provided",
			args:      []string{},
			serviceID: "service-id",
			planID:    "plan-id",
			wantErr:   false,
		},
		{
			name:      "invalid args - missing service and plan names, no IDs",
			args:      []string{},
			serviceID: "",
			planID:    "",
			wantErr:   true,
			errMsg:    "please provide the service name and service plan name or the service ID and service plan ID",
		},
		{
			name:      "invalid args - only one argument provided",
			args:      []string{"service1"},
			serviceID: "",
			planID:    "",
			wantErr:   true,
			errMsg:    "invalid arguments: service1. Need 2 arguments: [service-name] [plan-name]",
		},
		{
			name:      "invalid args - too many arguments",
			args:      []string{"service1", "plan1", "extra"},
			serviceID: "",
			planID:    "",
			wantErr:   true,
			errMsg:    "invalid arguments: service1 plan1 extra. Need 2 arguments: [service-name] [plan-name]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateUpdateArguments(tt.args, tt.serviceID, tt.planID)
			if tt.wantErr {
				require.Error(err)
				require.Contains(err.Error(), tt.errMsg)
			} else {
				require.NoError(err)
			}
		})
	}
}
