package dataaccess

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
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

// servicePlanSchemaFixture is a snapshot of the authoritative ServicePlanSpec schema
// the platform serves at /schema/service-spec-schema.json. It is generated, never
// hand-edited; regenerate it after any change to the Go models the schema is
// reflected from, or to the schema normalization that runs before publication:
//
//	cd <workspace>/ows-orchestration
//	SERVICE_PLAN_SCHEMA_FIXTURE=<workspace>/ctl/internal/dataaccess/testdata/service-plan-schema.json \
//	  go test ./v1/pkg/registration/service/servicebuild/serviceplan/ \
//	    -run TestWriteServicePlanSchemaFixture -count=1
//
// That generator reads extension.GetSchemaJSON("service-plan"), the same accessor the
// schema endpoint returns, so the fixture cannot drift from what a user downloads.
const servicePlanSchemaFixture = "testdata/service-plan-schema.json"

// helmPlanSpec renders a minimal valid plan whose single service is a Helm chart with
// the given source block, so a verdict can only come from the source rules.
func helmPlanSpec(sourceBlock string) []byte {
	lines := strings.Split(strings.TrimRight(sourceBlock, "\n"), "\n")
	for i, line := range lines {
		if line != "" {
			lines[i] = "      " + line
		}
	}

	return []byte(`name: Helm Source Plan
tenancyType: OMNISTRATE_DEDICATED_TENANCY
services:
  - name: Chart
    helmChartConfiguration:
` + strings.Join(lines, "\n") + "\n")
}

// TestValidateSpec_HelmArtifactSource is the CLI half of the Helm source alignment:
// `omnistrate-ctl docs validate` has to accept exactly the specs the platform builds.
// It is structural validation, so no artifact needs to exist or have been uploaded.
func TestValidateSpec_HelmArtifactSource(t *testing.T) {
	tests := []struct {
		name       string
		sourceYAML string
		valid      bool
	}{
		{
			name: "repository backed chart",
			sourceYAML: `chartName: nginx
chartVersion: 1.0.0
chartRepoName: bitnami
chartRepoURL: https://charts.bitnami.com/bitnami`,
			valid: true,
		},
		{
			name:       "artifact backed chart with a relative path",
			sourceYAML: `artifactRelativePath: charts/my-chart.tgz`,
			valid:      true,
		},
		{
			name:       "artifact backed chart with the legacy alias",
			sourceYAML: `artifactsLocalPath: ./charts/my-chart.tgz`,
			valid:      true,
		},
		{
			name: "artifact backed chart may still name the chart",
			sourceYAML: `artifactRelativePath: charts/my-chart.tgz
chartName: my-chart
chartVersion: 0.1.0`,
			valid: true,
		},
		{
			name: "artifact backed chart with blank repository fields",
			sourceYAML: `artifactRelativePath: charts/my-chart.tgz
chartRepoName: ""
chartRepoURL: "  "`,
			valid: true,
		},
		{
			name: "repository backed chart missing its repository url",
			sourceYAML: `chartName: nginx
chartVersion: 1.0.0
chartRepoName: bitnami`,
		},
		{
			name: "repository backed chart with a whitespace-only repository name",
			sourceYAML: `chartName: nginx
chartVersion: 1.0.0
chartRepoName: "   "
chartRepoURL: https://charts.bitnami.com/bitnami`,
		},
		{
			name: "mixed repository and artifact source",
			sourceYAML: `artifactRelativePath: charts/my-chart.tgz
chartName: nginx
chartVersion: 1.0.0
chartRepoName: bitnami
chartRepoURL: https://charts.bitnami.com/bitnami`,
		},
		{
			name: "artifact source with a repository name only",
			sourceYAML: `artifactRelativePath: charts/my-chart.tgz
chartRepoName: bitnami`,
		},
		{
			name: "both artifact aliases set",
			sourceYAML: `artifactRelativePath: charts/my-chart.tgz
artifactsLocalPath: /local/charts/my-chart.tgz`,
		},
		{
			name: "both artifact aliases set to the same path",
			sourceYAML: `artifactRelativePath: charts/my-chart.tgz
artifactsLocalPath: charts/my-chart.tgz`,
		},
		{
			name: "artifact source with repository credentials",
			sourceYAML: `artifactRelativePath: charts/my-chart.tgz
authProvider:
  username: user
  password: secret`,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result, err := ValidateSpecWithSchemaFile("plan.yaml", helmPlanSpec(test.sourceYAML),
				"service-plan", servicePlanSchemaFixture)
			require.NoError(t, err)
			assert.Equal(t, test.valid, result.Valid, "violations: %+v", result.Violations)
		})
	}
}

// TestValidateSpec_HelmArtifactSourceFixtureIsUsable compiles the generated fixture
// and checks it is the corrected schema, so a stale or partially regenerated snapshot
// fails loudly instead of quietly passing every case above.
func TestValidateSpec_HelmArtifactSourceFixtureIsUsable(t *testing.T) {
	raw, err := os.ReadFile(servicePlanSchemaFixture)
	require.NoError(t, err)

	var document map[string]any
	require.NoError(t, json.Unmarshal(raw, &document))

	definitions, ok := document["$defs"].(map[string]any)
	require.True(t, ok, "expected $defs in the generated fixture")

	helmChart, ok := definitions["HelmChartConfiguration"].(map[string]any)
	require.True(t, ok, "expected a HelmChartConfiguration definition")
	assert.NotContains(t, helmChart, "required",
		"the repository coordinates must not be unconditionally required")

	alternatives, ok := helmChart["allOf"].([]any)
	require.True(t, ok, "expected the source rule under allOf")
	require.Len(t, alternatives, 1)
	branches, ok := alternatives[0].(map[string]any)["oneOf"].([]any)
	require.True(t, ok)
	assert.Len(t, branches, 3, "repository, artifactRelativePath and artifactsLocalPath")

	// An escaped backslash-u would match the letter "u" rather than whitespace.
	assert.NotContains(t, string(raw), `\\u`)

	// Operator Helm dependencies must not have been relaxed along with the chart.
	operator, ok := definitions["OperatorHelmChartDependency"].(map[string]any)
	require.True(t, ok)
	assert.Len(t, operator["required"], 4)
}
