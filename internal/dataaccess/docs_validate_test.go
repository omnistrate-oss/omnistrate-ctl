package dataaccess

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDetectSpecType(t *testing.T) {
	tests := []struct {
		name string
		spec string
		want string
	}{
		{
			name: "compose via x-omnistrate extension",
			spec: "x-omnistrate-service-plan:\n  name: p\nservices:\n  api:\n    image: nginx\n",
			want: "compose",
		},
		{
			name: "compose via services mapping",
			spec: "version: \"3.9\"\nservices:\n  api:\n    image: nginx\n",
			want: "compose",
		},
		{
			name: "compose via customer integrations",
			spec: "x-customer-integrations:\n  logs: {}\nservices:\n  api:\n    image: nginx\n",
			want: "compose",
		},
		{
			name: "plan via services sequence",
			spec: "name: My Plan\nservices:\n  - name: db\n",
			want: "service-plan",
		},
		{
			name: "plan via systemWorkflows without services",
			spec: "name: My Plan\nsystemWorkflows:\n  create: {}\n",
			want: "service-plan",
		},
		{
			name: "ambiguous",
			spec: "name: something\ndescription: no services at all\n",
			want: "",
		},
		{
			name: "not yaml",
			spec: "this: [is: not: valid: yaml\n",
			want: "",
		},
		{
			name: "empty",
			spec: "",
			want: "",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.want, DetectSpecType([]byte(test.spec)))
		})
	}
}

// writeSchema writes a small self-contained schema exercising the features the real
// ones rely on: $defs, $ref, additionalProperties:false, required, and enum.
func writeSchema(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "schema.json")
	require.NoError(t, os.WriteFile(path, []byte(`{
      "$schema": "https://json-schema.org/draft/2020-12/schema",
      "$ref": "#/$defs/Root",
      "$defs": {
        "Root": {
          "type": "object",
          "additionalProperties": false,
          "required": ["name"],
          "properties": {
            "name": {"type": "string"},
            "tenancyType": {"type": "string", "enum": ["CUSTOM_TENANCY"]},
            "services": {"type": "array", "items": {"$ref": "#/$defs/Service"}}
          }
        },
        "Service": {
          "type": "object",
          "additionalProperties": false,
          "properties": {"name": {"type": "string"}, "replicas": {"type": "integer"}}
        }
      }
    }`), 0o600))
	return path
}

func TestValidateSpecWithSchemaFileAcceptsValidSpec(t *testing.T) {
	spec := "name: My Plan\ntenancyType: CUSTOM_TENANCY\nservices:\n  - name: db\n    replicas: 2\n"

	result, err := ValidateSpecWithSchemaFile("plan.yaml", []byte(spec), "service-plan", writeSchema(t))
	require.NoError(t, err)
	assert.True(t, result.Valid)
	assert.Empty(t, result.Violations)
	assert.Equal(t, "plan.yaml", result.File)
	assert.Equal(t, "service-plan", result.SpecType)
}

func TestValidateSpecWithSchemaFileReportsUnknownField(t *testing.T) {
	spec := "name: My Plan\nversion: \"1.0\"\n"

	result, err := ValidateSpecWithSchemaFile("plan.yaml", []byte(spec), "service-plan", writeSchema(t))
	require.NoError(t, err)
	require.False(t, result.Valid)
	require.Len(t, result.Violations, 1)
	assert.Equal(t, "/", result.Violations[0].Path)
	assert.Contains(t, result.Violations[0].Message, "version")
}

func TestValidateSpecWithSchemaFileReportsNestedViolationWithPath(t *testing.T) {
	spec := "name: My Plan\nservices:\n  - name: db\n    replicaCount: 2\n"

	result, err := ValidateSpecWithSchemaFile("plan.yaml", []byte(spec), "service-plan", writeSchema(t))
	require.NoError(t, err)
	require.False(t, result.Valid)
	// The path has to point at the offending element, not just the document root.
	require.Len(t, result.Violations, 1)
	assert.Equal(t, "/services/0", result.Violations[0].Path)
	assert.Contains(t, result.Violations[0].Message, "replicaCount")
}

func TestValidateSpecWithSchemaFileReportsEnumViolation(t *testing.T) {
	spec := "name: My Plan\ntenancyType: NOT_A_TENANCY\n"

	result, err := ValidateSpecWithSchemaFile("plan.yaml", []byte(spec), "service-plan", writeSchema(t))
	require.NoError(t, err)
	require.False(t, result.Valid)
	require.Len(t, result.Violations, 1)
	assert.Equal(t, "/tenancyType", result.Violations[0].Path)
	assert.Contains(t, result.Violations[0].Message, "CUSTOM_TENANCY")
}

func TestValidateSpecWithSchemaFileReportsMissingRequired(t *testing.T) {
	result, err := ValidateSpecWithSchemaFile("plan.yaml", []byte("tenancyType: CUSTOM_TENANCY\n"), "service-plan", writeSchema(t))
	require.NoError(t, err)
	require.False(t, result.Valid)
	require.Len(t, result.Violations, 1)
	assert.Contains(t, result.Violations[0].Message, "name")
}

func TestValidateSpecWithSchemaFileSortsViolationsByPath(t *testing.T) {
	spec := "name: My Plan\ntenancyType: NOPE\nservices:\n  - name: db\n    bogus: 1\n"

	result, err := ValidateSpecWithSchemaFile("plan.yaml", []byte(spec), "service-plan", writeSchema(t))
	require.NoError(t, err)
	require.False(t, result.Valid)
	require.Len(t, result.Violations, 2)
	assert.Equal(t, "/services/0", result.Violations[0].Path)
	assert.Equal(t, "/tenancyType", result.Violations[1].Path)
}

func TestValidateSpecWithSchemaFileRejectsUnreadableSchema(t *testing.T) {
	_, err := ValidateSpecWithSchemaFile("plan.yaml", []byte("name: x\n"), "service-plan",
		filepath.Join(t.TempDir(), "missing.json"))
	require.Error(t, err)
}

func TestValidateSpecWithSchemaFileRejectsMalformedSchema(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bad.json")
	require.NoError(t, os.WriteFile(path, []byte("{not json"), 0o600))

	_, err := ValidateSpecWithSchemaFile("plan.yaml", []byte("name: x\n"), "service-plan", path)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse the schema")
}

func TestValidateSpecWithSchemaFileRejectsMalformedYAML(t *testing.T) {
	_, err := ValidateSpecWithSchemaFile("plan.yaml", []byte("name: [unclosed\n"), "service-plan", writeSchema(t))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "plan.yaml")
}
