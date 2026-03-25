package utils

import (
	"testing"

	"github.com/omnistrate-oss/omnistrate-ctl/internal/model"
	"github.com/stretchr/testify/require"
)

func TestParseFilters(t *testing.T) {
	require := require.New(t)

	tests := []struct {
		name                string
		filters             []string
		supportedFilterKeys []string
		expectedFilterMaps  []map[string]string
		expectError         bool
		expectedErrorMsg    string
	}{
		{
			name:                "Valid filters with supported keys",
			filters:             []string{"service:s1,environment:e2", "plan:p1,version:v1"},
			supportedFilterKeys: []string{"service", "environment", "plan", "version"},
			expectedFilterMaps: []map[string]string{
				{"service": "s1", "environment": "e2"},
				{"plan": "p1", "version": "v1"},
			},
			expectError: false,
		},
		{
			name:                "Invalid filter format",
			filters:             []string{"service:s1,environment"},
			supportedFilterKeys: []string{"service", "environment"},
			expectedFilterMaps:  nil,
			expectError:         true,
			expectedErrorMsg:    "invalid filter format: environment, expected key:value",
		},
		{
			name:                "Unsupported filter key",
			filters:             []string{"unsupported:u1,service:s1"},
			supportedFilterKeys: []string{"service", "environment"},
			expectedFilterMaps:  nil,
			expectError:         true,
			expectedErrorMsg:    "unsupported filter key: unsupported",
		},
		{
			name:                "Mixed valid and invalid filters",
			filters:             []string{"service:s1,unsupported:u1"},
			supportedFilterKeys: []string{"service", "environment"},
			expectedFilterMaps:  nil,
			expectError:         true,
			expectedErrorMsg:    "unsupported filter key: unsupported",
		},
		{
			name:                "Empty filters",
			filters:             []string{},
			supportedFilterKeys: []string{"service", "environment"},
			expectedFilterMaps:  []map[string]string{},
			expectError:         false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			filterMaps, err := ParseFilters(test.filters, test.supportedFilterKeys)

			if test.expectError {
				require.Error(err)
				require.ErrorContains(err, test.expectedErrorMsg)
			} else {
				require.NoError(err)
				require.Equal(test.expectedFilterMaps, filterMaps)
			}
		})
	}
}

func TestMatchesFilters(t *testing.T) {
	require := require.New(t)
	tests := []struct {
		name             string
		model            interface{}
		filters          []map[string]string
		ok               bool
		expectError      bool
		expectedErrorMsg string
	}{
		{
			name: "Instance matches all filters",
			model: model.Instance{
				InstanceID:    "123",
				Service:       "s1",
				Environment:   "e2",
				Plan:          "p1",
				Version:       "v1",
				Resource:      "r1",
				CloudProvider: "cp1",
				Region:        "rc1",
				Status:        "active",
			},
			filters: []map[string]string{
				{"instance_id": "123", "service": "s1", "environment": "e2", "plan": "p1", "version": "v1", "resource": "r1", "cloud_provider": "cp1", "region": "rc1", "status": "active"},
			},
			ok:          true,
			expectError: false,
		},
		{
			name: "Instance does not match any filters",
			model: model.Instance{
				InstanceID:    "123",
				Service:       "s1",
				Environment:   "e2",
				Plan:          "p1",
				Version:       "v1",
				Resource:      "r1",
				CloudProvider: "cp1",
				Region:        "rc1",
				Status:        "active",
			},
			filters: []map[string]string{
				{"service": "s2", "environment": "e2"},
				{"instance_id": "999", "status": "inactive"},
			},
			ok:          false,
			expectError: false,
		},
		{
			name: "Instance matches one of the filters",
			model: model.Instance{
				InstanceID:    "123",
				Service:       "s1",
				Environment:   "e2",
				Plan:          "p1",
				Version:       "v1",
				Resource:      "r1",
				CloudProvider: "cp1",
				Region:        "rc1",
				Status:        "active",
			},
			filters: []map[string]string{
				{"service": "s1", "environment": "e2"},
				{"instance_id": "123"},
			},
			ok:          true,
			expectError: false,
		},
		{
			name: "Empty filters",
			model: model.Instance{
				InstanceID:    "123",
				Service:       "s1",
				Environment:   "e2",
				Plan:          "p1",
				Version:       "v1",
				Resource:      "r1",
				CloudProvider: "cp1",
				Region:        "rc1",
				Status:        "active",
			},
			filters:     []map[string]string{},
			ok:          true,
			expectError: false,
		},
		{
			name: "Invalid filter key",
			model: model.Instance{
				InstanceID:    "123",
				Service:       "s1",
				Environment:   "e2",
				Plan:          "p1",
				Version:       "v1",
				Resource:      "r1",
				CloudProvider: "cp1",
				Region:        "rc1",
				Status:        "active",
			},
			filters: []map[string]string{
				{"invalid_field": "value"},
			},
			ok:               false,
			expectError:      true,
			expectedErrorMsg: "invalid JSON field name: invalid_field",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := MatchesFilters(test.model, test.filters)
			if test.expectError {
				require.Error(err)
				require.ErrorContains(err, test.expectedErrorMsg)
			} else {
				require.NoError(err)
				require.Equal(test.ok, got, "MatchesFilters(%v, %v) = %v; want %v", test.model, test.filters, got, test.ok)
			}
		})
	}
}
