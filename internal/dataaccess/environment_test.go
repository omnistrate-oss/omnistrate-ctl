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
		name               string
		productTierID      string
		productTierVersion string
		wantProduct        bool
		wantVersion        bool
	}{
		{name: "without product tier version"},
		{name: "with product tier id", productTierID: "pt-123", wantProduct: true},
		{name: "with product tier version and product tier id", productTierID: "pt-123", productTierVersion: "1.2.3", wantProduct: true, wantVersion: true},
		{name: "without product tier id drops product tier version", productTierVersion: "1.2.3"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := newPromoteServiceEnvironmentRequest(test.productTierID, test.productTierVersion)
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

			_, ok = requestMap["productTierVersion"]
			require.Equal(t, test.wantVersion, ok)
			if test.wantVersion {
				require.NotNil(t, request.ProductTierVersion)
				require.Equal(t, test.productTierVersion, *request.ProductTierVersion)
			} else {
				require.Nil(t, request.ProductTierVersion)
			}
		})
	}
}

func TestPromoteServiceEnvironmentRejectsProductTierVersionWithoutProductTierID(t *testing.T) {
	err := PromoteServiceEnvironment(t.Context(), "token", "svc-123", "env-123", "", "1.2.3")
	require.EqualError(t, err, "product tier version can only be provided when product tier ID is provided")
}
