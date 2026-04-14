package instance

import (
	"testing"

	openapiclientfleet "github.com/omnistrate-oss/omnistrate-sdk-go/fleet"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateCommandFlags(t *testing.T) {
	require.Equal(t, "Create an instance deployment", createCmd.Short)

	flag := createCmd.Flags().Lookup("customer-account-id")
	require.NotNil(t, flag)
	assert.Contains(t, flag.Usage, "account customer list")
}

func TestApplyCustomerAccountIDParam_NoCustomerAccountID(t *testing.T) {
	params := map[string]any{"existing": "value"}

	updated, err := applyCustomerAccountIDParam(params, &openapiclientfleet.ServiceOffering{}, "")
	require.NoError(t, err)
	assert.Equal(t, params, updated)
}

func TestApplyCustomerAccountIDParam_BYOARequiresCustomerAccount(t *testing.T) {
	_, err := applyCustomerAccountIDParam(
		nil,
		&openapiclientfleet.ServiceOffering{ServiceModelType: serviceModelTypeBYOA},
		"",
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "--customer-account-id")
	assert.Contains(t, err.Error(), "account customer list")
}

func TestApplyCustomerAccountIDParam_BYOAAllowsMagicParam(t *testing.T) {
	params := map[string]any{customerAccountConfigIDParamKey: "instance-existing"}

	updated, err := applyCustomerAccountIDParam(
		params,
		&openapiclientfleet.ServiceOffering{ServiceModelType: serviceModelTypeBYOA},
		"",
	)
	require.NoError(t, err)
	assert.Equal(t, params, updated)
}

func TestApplyCustomerAccountIDParam_RequiresBYOAPlan(t *testing.T) {
	_, err := applyCustomerAccountIDParam(
		nil,
		&openapiclientfleet.ServiceOffering{ServiceModelType: "OMNISTRATE_HOSTED"},
		"instance-abcd1234",
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "only supported for BYOA service plans")
}

func TestApplyCustomerAccountIDParam_RejectsDuplicateMagicParam(t *testing.T) {
	_, err := applyCustomerAccountIDParam(
		map[string]any{customerAccountConfigIDParamKey: "instance-existing"},
		&openapiclientfleet.ServiceOffering{ServiceModelType: serviceModelTypeBYOA},
		"instance-abcd1234",
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), customerAccountConfigIDParamKey)
}

func TestApplyCustomerAccountIDParam_AddsCustomerAccountID(t *testing.T) {
	updated, err := applyCustomerAccountIDParam(
		map[string]any{"existing": "value"},
		&openapiclientfleet.ServiceOffering{ServiceModelType: "byoa"},
		"instance-abcd1234",
	)
	require.NoError(t, err)
	assert.Equal(t, "value", updated["existing"])
	assert.Equal(t, "instance-abcd1234", updated[customerAccountConfigIDParamKey])
}
