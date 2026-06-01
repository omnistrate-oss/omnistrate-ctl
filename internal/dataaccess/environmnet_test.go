package dataaccess

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCleanupId(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{input: "\"se-fhw4UJW8G3\"", expected: "se-fhw4UJW8G3"},
		{input: "\"se-fhw4UJW8G3\"\n", expected: "se-fhw4UJW8G3"},
		{input: "\t\"se-fhw4UJW8G3\"\t", expected: "se-fhw4UJW8G3"},
		{input: "\n\"se-fhw4UJW8G3\"\n", expected: "se-fhw4UJW8G3"},
		{input: "se-fhw4UJW8G3", expected: "se-fhw4UJW8G3"},
	}

	for _, test := range tests {
		result := cleanupId(test.input)
		if result != test.expected {
			t.Errorf("cleanupId(%q) = %q; expected %q", test.input, result, test.expected)
		}
	}
}

func TestNewPromoteServiceEnvironmentRequest(t *testing.T) {
	tests := []struct {
		name          string
		productTierID string
		sourceVersion string
		wantProduct   bool
		wantSource    bool
	}{
		{name: "without source version"},
		{name: "with product tier id", productTierID: "pt-123", wantProduct: true},
		{name: "with source version and product tier id", productTierID: "pt-123", sourceVersion: "1.2.3", wantProduct: true, wantSource: true},
		{name: "without product tier id drops source version", sourceVersion: "1.2.3"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := newPromoteServiceEnvironmentRequest(test.productTierID, test.sourceVersion)
			requestMap, err := request.ToMap()
			require.NoError(t, err)

			_, ok := requestMap["productTierId"]
			require.Equal(t, test.wantProduct, ok)
			if test.wantProduct {
				require.NotNil(t, request.ProductTierId)
				require.Equal(t, test.productTierID, *request.ProductTierId)
			} else {
				require.Nil(t, request.ProductTierId)
			}

			sourceVersion, ok := requestMap["sourceVersion"]
			require.Equal(t, test.wantSource, ok)
			if test.wantSource {
				require.Equal(t, test.sourceVersion, sourceVersion)
			}
		})
	}
}

func TestPromoteServiceEnvironmentRejectsSourceVersionWithoutProductTierID(t *testing.T) {
	err := PromoteServiceEnvironment(t.Context(), "token", "svc-123", "env-123", "", "1.2.3")
	require.EqualError(t, err, "source version can only be provided when product tier ID is provided")
}
