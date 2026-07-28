package docs

import (
	"context"
	"testing"

	"github.com/omnistrate-oss/omnistrate-ctl/cmd"
	"github.com/omnistrate-oss/omnistrate-ctl/test/testutils"
	"github.com/stretchr/testify/require"
)

func Test_docs_compose_spec(t *testing.T) {
	testutils.SmokeTest(t)

	ctx := context.TODO()
	require := require.New(t)
	defer testutils.Cleanup()

	// PASS: list all compose-spec tags
	cmd.RootCmd.SetArgs([]string{"docs", "compose-spec"})
	err := cmd.RootCmd.ExecuteContext(ctx)
	require.NoError(err)

	// PASS: list all compose-spec tags with JSON output
	cmd.RootCmd.SetArgs([]string{"docs", "compose-spec", "--output", "json"})
	err = cmd.RootCmd.ExecuteContext(ctx)
	require.NoError(err)

	// PASS: search for a specific tag
	cmd.RootCmd.SetArgs([]string{"docs", "compose-spec", "networks"})
	err = cmd.RootCmd.ExecuteContext(ctx)
	require.NoError(err)

	// PASS: search for a specific tag with JSON output
	cmd.RootCmd.SetArgs([]string{"docs", "compose-spec", "networks", "--output", "json"})
	err = cmd.RootCmd.ExecuteContext(ctx)
	require.NoError(err)

	// PASS: get the JSON schema for an extension tag
	cmd.RootCmd.SetArgs([]string{"docs", "compose-spec", "x-omnistrate-compute", "--json-schema-only", "--output", "json"})
	err = cmd.RootCmd.ExecuteContext(ctx)
	require.NoError(err)

	// PASS: a nested tag resolves to the schema of its root extension
	cmd.RootCmd.SetArgs([]string{"docs", "compose-spec", "x-omnistrate-capabilities.sidecars", "--json-schema-only", "--output", "json"})
	err = cmd.RootCmd.ExecuteContext(ctx)
	require.NoError(err)

	// FAIL: a native compose tag has no schema of its own
	cmd.RootCmd.SetArgs([]string{"docs", "compose-spec", "ports", "--json-schema-only", "--output", "json"})
	err = cmd.RootCmd.ExecuteContext(ctx)
	require.Error(err)
}

func Test_docs_plan_spec(t *testing.T) {
	testutils.SmokeTest(t)

	ctx := context.TODO()
	require := require.New(t)
	defer testutils.Cleanup()

	// PASS: list all plan-spec tags
	cmd.RootCmd.SetArgs([]string{"docs", "plan-spec"})
	err := cmd.RootCmd.ExecuteContext(ctx)
	require.NoError(err)

	// PASS: list all plan-spec tags with JSON output
	cmd.RootCmd.SetArgs([]string{"docs", "plan-spec", "--output", "json"})
	err = cmd.RootCmd.ExecuteContext(ctx)
	require.NoError(err)

	// PASS: search for a specific tag
	cmd.RootCmd.SetArgs([]string{"docs", "plan-spec", "compute"})
	err = cmd.RootCmd.ExecuteContext(ctx)
	require.NoError(err)

	// PASS: search for a specific tag with JSON output
	cmd.RootCmd.SetArgs([]string{"docs", "plan-spec", "compute", "--output", "json"})
	err = cmd.RootCmd.ExecuteContext(ctx)
	require.NoError(err)

	// PASS: plan spec tags resolve to the ServicePlanSpec schema that defines them
	cmd.RootCmd.SetArgs([]string{"docs", "plan-spec", "helm chart configuration", "--json-schema-only", "--output", "json"})
	err = cmd.RootCmd.ExecuteContext(ctx)
	require.NoError(err)
}

func Test_docs_json_schema(t *testing.T) {
	testutils.SmokeTest(t)

	ctx := context.TODO()
	require := require.New(t)
	defer testutils.Cleanup()

	// PASS: list the schema types that can be requested
	cmd.RootCmd.SetArgs([]string{"docs", "json-schema"})
	err := cmd.RootCmd.ExecuteContext(ctx)
	require.NoError(err)

	// PASS: list the schema types with JSON output
	cmd.RootCmd.SetArgs([]string{"docs", "json-schema", "--output", "json"})
	err = cmd.RootCmd.ExecuteContext(ctx)
	require.NoError(err)

	// PASS: every listed type resolves to a schema
	for _, schemaType := range []string{"compose", "service-plan", "system-parameters", "x-omnistrate-compute"} {
		cmd.RootCmd.SetArgs([]string{"docs", "json-schema", schemaType, "--output", "json"})
		err = cmd.RootCmd.ExecuteContext(ctx)
		require.NoError(err, "schema type %s should resolve", schemaType)
	}

	// FAIL: unknown schema type
	cmd.RootCmd.SetArgs([]string{"docs", "json-schema", "not-a-schema-type", "--output", "json"})
	err = cmd.RootCmd.ExecuteContext(ctx)
	require.Error(err)
}
