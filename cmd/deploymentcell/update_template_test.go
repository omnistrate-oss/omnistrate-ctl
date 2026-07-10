package deploymentcell

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFormatDeploymentCellUpdateSummaryCountsManagedIdentitiesAsAmenities(t *testing.T) {
	tests := []struct {
		name                   string
		managedAmenitiesCount  int
		customAmenitiesCount   int
		managedIdentitiesCount int
		expected               string
	}{
		{
			name:                   "singular managed identity",
			managedAmenitiesCount:  12,
			customAmenitiesCount:   1,
			managedIdentitiesCount: 1,
			expected:               "Updating with 14 amenities (12 managed, 1 custom, 1 managed identity)\n",
		},
		{
			name:                   "singular total amenity",
			managedAmenitiesCount:  0,
			customAmenitiesCount:   0,
			managedIdentitiesCount: 1,
			expected:               "Updating with 1 amenity (0 managed, 0 custom, 1 managed identity)\n",
		},
		{
			name:                   "zero managed identities",
			managedAmenitiesCount:  12,
			customAmenitiesCount:   1,
			managedIdentitiesCount: 0,
			expected:               "Updating with 13 amenities (12 managed, 1 custom, 0 managed identities)\n",
		},
		{
			name:                   "multiple managed identities",
			managedAmenitiesCount:  12,
			customAmenitiesCount:   1,
			managedIdentitiesCount: 2,
			expected:               "Updating with 15 amenities (12 managed, 1 custom, 2 managed identities)\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			summary := formatDeploymentCellUpdateSummary(
				tt.managedAmenitiesCount,
				tt.customAmenitiesCount,
				tt.managedIdentitiesCount,
			)

			require.Equal(t, tt.expected, summary)
		})
	}
}
