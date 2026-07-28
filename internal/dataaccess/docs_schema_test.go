package dataaccess

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolveComposeSpecJSONSchemaType(t *testing.T) {
	tests := []struct {
		name string
		tag  string
		want string
	}{
		{"extension name", "x-omnistrate-compute", "x-omnistrate-compute"},
		{"mixed case", "X-Omnistrate-Compute", "x-omnistrate-compute"},
		{"nested field", "x-omnistrate-capabilities.sidecars", "x-omnistrate-capabilities"},
		{"deeply nested field", "x-omnistrate-compute.instanceTypes.configurationOverrides", "x-omnistrate-compute"},
		{"service plan nested field", "x-omnistrate-service-plan.features.CUSTOM_NETWORKS", "x-omnistrate-service-plan"},
		// The docs heading is singular; the API type is plural.
		{"singular registry heading", "x-omnistrate-image-registry-attribute", "x-omnistrate-image-registry-attributes"},
		{"plural registry heading", "x-omnistrate-image-registry-attributes", "x-omnistrate-image-registry-attributes"},
		{"deprecated note in heading", "x-omnistrate-integrations (deprecated)", "x-omnistrate-integrations"},
		{"code span", "`x-omnistrate-storage`", "x-omnistrate-storage"},
		{"whole spec type", "compose", "compose"},
		{"customer integrations", "x-customer-integrations.logs", "x-customer-integrations"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := ResolveComposeSpecJSONSchemaType(test.tag)
			require.NoError(t, err)
			assert.Equal(t, test.want, got)
		})
	}
}

func TestResolveComposeSpecJSONSchemaTypeUnknown(t *testing.T) {
	// Native Docker Compose tags and retired extensions have no schema of their own.
	for _, tag := range []string{"ports", "volumes", "networks", "x-omnistrate-byoa", "x-omnistrate-my-account (deprecated)", ""} {
		t.Run(tag, func(t *testing.T) {
			_, err := ResolveComposeSpecJSONSchemaType(tag)
			require.Error(t, err)
			var unknown *ErrUnknownJSONSchemaType
			require.ErrorAs(t, err, &unknown)
			// The error has to be actionable: it names the types that do work.
			assert.Contains(t, err.Error(), "x-omnistrate-compute")
			assert.Contains(t, err.Error(), "service-plan")
		})
	}
}

func TestResolvePlanSpecJSONSchemaType(t *testing.T) {
	// Every plan spec heading documents a definition inside the service-plan schema,
	// so all of them resolve to that schema rather than failing.
	for _, tag := range []string{
		"Root schema",
		"`Compute schema`",
		"Helm chart configuration schema",
		"Features schema",
		"Operator CRD schema",
		"Load Balancers schema",
		"Terraform per-cloud-provider configuration schema",
	} {
		t.Run(tag, func(t *testing.T) {
			got, err := ResolvePlanSpecJSONSchemaType(tag)
			require.NoError(t, err)
			assert.Equal(t, "service-plan", got)
		})
	}
}

func TestResolvePlanSpecJSONSchemaTypeExactTypeWins(t *testing.T) {
	got, err := ResolvePlanSpecJSONSchemaType("system-parameters")
	require.NoError(t, err)
	assert.Equal(t, "system-parameters", got)
}

func TestResolvePlanSpecJSONSchemaTypeRejectsEmpty(t *testing.T) {
	_, err := ResolvePlanSpecJSONSchemaType("  ")
	require.Error(t, err)
}

func TestListJSONSchemaTypes(t *testing.T) {
	types := ListJSONSchemaTypes()
	require.NotEmpty(t, types)

	names := make([]string, 0, len(types))
	for _, entry := range types {
		assert.NotEmpty(t, entry.Type, "every listed type needs a name")
		assert.NotEmpty(t, entry.Description, "every listed type needs a description")
		names = append(names, entry.Type)
	}

	assert.IsIncreasing(t, names, "types should be listed in sorted order")
	assert.Contains(t, names, "compose")
	assert.Contains(t, names, "service-plan")
	assert.Contains(t, names, "system-parameters")
	assert.Contains(t, names, "x-omnistrate-compute")
}

func TestListJSONSchemaTypesReturnsCopy(t *testing.T) {
	types := ListJSONSchemaTypes()
	types[0].Type = "mutated"
	assert.NotEqual(t, "mutated", ListJSONSchemaTypes()[0].Type)
}

func TestIsValidJSONSchemaType(t *testing.T) {
	assert.True(t, IsValidJSONSchemaType("compose"))
	assert.True(t, IsValidJSONSchemaType("x-omnistrate-image-registry-attributes"))
	assert.False(t, IsValidJSONSchemaType("x-omnistrate-image-registry-attribute"))
	assert.False(t, IsValidJSONSchemaType("ports"))
	assert.False(t, IsValidJSONSchemaType(""))
}
