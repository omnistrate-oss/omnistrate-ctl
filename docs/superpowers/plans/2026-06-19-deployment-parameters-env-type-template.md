# deployment-parameters environment filter + create-params template Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a required `--environment` filter to `instance deployment-parameters` and a `deployment-parameters template` subcommand that emits a fillable JSON params object for `instance create`.

**Architecture:** All changes are in `cmd/instance/deployment_parameters.go` (plus registration in `cmd/instance/instance.go` and a new test file). The matching, validation, and template-building logic is factored into pure functions so they are unit-testable without network calls. A shared `fetchDeploymentParameters` helper is extracted so the parent command and the new subcommand share one resolution code path.

**Tech Stack:** Go, `spf13/cobra`, `stretchr/testify/require`, `omnistrate-sdk-go/fleet` SDK.

## Global Constraints

- Module path: `github.com/omnistrate-oss/omnistrate-ctl`.
- Fleet SDK import alias: `openapiclientfleet "github.com/omnistrate-oss/omnistrate-sdk-go/fleet"`.
- `--environment` matches an environment name OR ID, case-insensitively — same semantics as `instance create --environment`. Reuse the existing `matchesIDOrName(id, name, arg string) bool` helper (already defined in `cmd/instance/create.go`, same package).
- `ResourceSearchRecord.ServiceEnvironmentName` and `.ServiceEnvironmentId` are plain `string` fields (not pointers).
- Pointer helper: `utils.ToPtr(...)` from `github.com/omnistrate-oss/omnistrate-ctl/internal/utils`.
- Run unit tests with: `go test ./cmd/instance/...`
- The existing `DeploymentParametersOutput` JSON schema (`--output=json`) must NOT change.

---

### Task 1: Environment filtering on `deployment-parameters`

Add the required `--environment` flag and a pure `findMatchingResource` function. Refactor `getServiceAndPlanInfo` to use the new match function with environment filtering.

**Files:**
- Modify: `cmd/instance/deployment_parameters.go`
- Test: `cmd/instance/deployment_parameters_test.go` (create)

**Interfaces:**
- Consumes: `openapiclientfleet.ResourceSearchRecord` (fields: `Id`, `Name`, `ServiceName`, `ProductTierName string`; `ServiceId`, `ProductTierId string`; `ServiceEnvironmentId`, `ServiceEnvironmentName string`); `matchesIDOrName(id, name, arg string) bool` (existing helper in `cmd/instance/create.go`).
- Produces:
  - `func findMatchingResource(records []openapiclientfleet.ResourceSearchRecord, serviceName, planName, resourceName, environment string) (*openapiclientfleet.ResourceSearchRecord, error)`
  - `func getServiceAndPlanInfo(ctx context.Context, token, serviceName, planName, resourceName, version, environment string) (serviceID, resourceID, productTierID, productTierVersion string, err error)` — note the new trailing `environment` parameter.

- [ ] **Step 1: Write the failing tests**

Create `cmd/instance/deployment_parameters_test.go`:

```go
package instance

import (
	"testing"

	openapiclientfleet "github.com/omnistrate-oss/omnistrate-sdk-go/fleet"
	"github.com/stretchr/testify/require"
)

func TestFindMatchingResource(t *testing.T) {
	require := require.New(t)

	records := []openapiclientfleet.ResourceSearchRecord{
		{
			Id:                     "res-dev",
			Name:                   "mySQL",
			ServiceName:            "mysql",
			ProductTierName:        "mysql",
			ServiceId:              "svc-1",
			ProductTierId:          "pt-dev",
			ServiceEnvironmentId:   "env-dev",
			ServiceEnvironmentName: "Dev",
		},
		{
			Id:                     "res-prod",
			Name:                   "mySQL",
			ServiceName:            "mysql",
			ProductTierName:        "mysql",
			ServiceId:              "svc-1",
			ProductTierId:          "pt-prod",
			ServiceEnvironmentId:   "env-prod",
			ServiceEnvironmentName: "Production",
		},
	}

	// Match by environment name (case-insensitive)
	match, err := findMatchingResource(records, "mysql", "mysql", "mySQL", "production")
	require.NoError(err)
	require.Equal("res-prod", match.Id)
	require.Equal("pt-prod", match.ProductTierId)

	match, err = findMatchingResource(records, "mysql", "mysql", "mySQL", "Dev")
	require.NoError(err)
	require.Equal("res-dev", match.Id)

	// Match by environment ID
	match, err = findMatchingResource(records, "mysql", "mysql", "mySQL", "env-prod")
	require.NoError(err)
	require.Equal("res-prod", match.Id)

	// No environment match -> not found
	_, err = findMatchingResource(records, "mysql", "mysql", "mySQL", "staging")
	require.Error(err)
	require.Contains(err.Error(), "not found")
}
```

- [ ] **Step 2: Run the tests to verify they fail**

Run: `go test ./cmd/instance/ -run TestFindMatchingResource -v`
Expected: FAIL — `undefined: findMatchingResource`.

- [ ] **Step 3: Add the match helper**

In `cmd/instance/deployment_parameters.go`, add after the `var (...)` block (the file already imports `context`, `fmt`, `sort`, `strings`):

```go
// findMatchingResource selects the resource record matching the service, plan,
// resource name, and environment (name or ID). Comparisons are case-insensitive.
func findMatchingResource(
	records []openapiclientfleet.ResourceSearchRecord,
	serviceName, planName, resourceName, environment string,
) (*openapiclientfleet.ResourceSearchRecord, error) {
	for i := range records {
		res := records[i]
		if res.Id == "" {
			continue
		}
		if !strings.EqualFold(res.Name, resourceName) ||
			!strings.EqualFold(res.ServiceName, serviceName) ||
			!strings.EqualFold(res.ProductTierName, planName) {
			continue
		}
		if !matchesIDOrName(res.ServiceEnvironmentId, res.ServiceEnvironmentName, environment) {
			continue
		}
		return &records[i], nil
	}
	return nil, fmt.Errorf(
		"resource '%s' not found for service '%s', plan '%s', and environment '%s'",
		resourceName, serviceName, planName, environment)
}
```

Add the fleet SDK import to the import block:

```go
	openapiclientfleet "github.com/omnistrate-oss/omnistrate-sdk-go/fleet"
```

- [ ] **Step 4: Refactor `getServiceAndPlanInfo` to use the match helper**

Replace the entire existing `getServiceAndPlanInfo` function with:

```go
// getServiceAndPlanInfo uses SearchInventory to find service, plan, and resource info
func getServiceAndPlanInfo(ctx context.Context, token, serviceName, planName, resourceName, version, environment string) (serviceID, resourceID, productTierID, productTierVersion string, err error) {
	// Search for the specific resource
	searchRes, err := dataaccess.SearchInventory(ctx, token, fmt.Sprintf("resource:%s", resourceName))
	if err != nil {
		return "", "", "", "", fmt.Errorf("failed to search inventory: %w", err)
	}

	match, err := findMatchingResource(searchRes.ResourceResults, serviceName, planName, resourceName, environment)
	if err != nil {
		return "", "", "", "", err
	}

	serviceID = match.ServiceId
	resourceID = match.Id
	productTierID = match.ProductTierId

	// Handle version: only pin a concrete version, leave empty for preferred/latest
	if version != "preferred" && version != "latest" {
		productTierVersion = version
	}

	return serviceID, resourceID, productTierID, productTierVersion, nil
}
```

- [ ] **Step 5: Add the `--environment` flag and pass it into `runDeploymentParameters`**

In `init()`, add the flag registration after the `output` flag line:

```go
	deploymentParametersCmd.Flags().String("environment", "", "Environment name or ID")
```

And mark it required (add after the existing `MarkFlagRequired("resource")` block):

```go
	if err := deploymentParametersCmd.MarkFlagRequired("environment"); err != nil {
		utils.PrintError(err)
	}
```

In `runDeploymentParameters`, after the `resourceName` flag is read (right before `token, err := common.GetTokenWithLogin()`), add:

```go
	environment, err := cmd.Flags().GetString("environment")
	if err != nil {
		utils.PrintError(err)
		return err
	}
```

Then update the `getServiceAndPlanInfo` call to pass `environment`:

```go
	serviceID, resourceID, productTierID, productTierVersion, err := getServiceAndPlanInfo(ctx, token, serviceName, planName, resourceName, version, environment)
```

Also update the command `Use` and `Example` strings to include the flag:

```go
	Use:   "deployment-parameters --service=[service] --environment=[environment] --plan=[plan] --version=[version] --resource=[resource]",
```

```go
	Example: `  omnistrate-ctl instance deployment-parameters --service=mysql --environment=dev --plan=mysql --version=latest --resource=mySQL
  omnistrate-ctl instance deployment-parameters --service=mysql --environment=dev --plan=mysql --version=latest --resource=mySQL --output=json`,
```

- [ ] **Step 6: Run the tests to verify they pass**

Run: `go test ./cmd/instance/ -run TestFindMatchingResource -v`
Expected: PASS.

- [ ] **Step 7: Verify the package builds**

Run: `go build ./cmd/instance/...`
Expected: no output, exit 0.

- [ ] **Step 8: Commit**

```bash
git add cmd/instance/deployment_parameters.go cmd/instance/deployment_parameters_test.go
git commit -m "feat: add required --environment filter to instance deployment-parameters

Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>"
```

---

### Task 2: Template value helpers

Add the pure functions that turn a `ParameterInfo` into a JSON-ready template value.

**Files:**
- Modify: `cmd/instance/deployment_parameters.go`
- Test: `cmd/instance/deployment_parameters_test.go`

**Interfaces:**
- Consumes: `ParameterInfo` (existing struct in this file; relevant fields: `Key string`, `Type string`, `IsList bool`, `DefaultValue *string`).
- Produces:
  - `func placeholderForType(valueType string) any`
  - `func coerceParamValue(value, valueType string, isList bool) any`
  - `func templateValueForParam(p ParameterInfo) any`
  - `func buildParamTemplate(params []ParameterInfo) map[string]any`

- [ ] **Step 1: Write the failing tests**

Add the `utils` import to `cmd/instance/deployment_parameters_test.go`'s import block (the tests below use `utils.ToPtr`):

```go
	"github.com/omnistrate-oss/omnistrate-ctl/internal/utils"
```

Then append:

```go
func TestTemplateValueForParam(t *testing.T) {
	require := require.New(t)

	// String with no default -> empty string placeholder
	require.Equal("", templateValueForParam(ParameterInfo{Key: "name", Type: "String"}))

	// String with default -> the default
	require.Equal("default", templateValueForParam(ParameterInfo{
		Key: "databaseName", Type: "String", DefaultValue: utils.ToPtr("default"),
	}))

	// Bool with no default -> false; with default -> parsed bool
	require.Equal(false, templateValueForParam(ParameterInfo{Key: "tls", Type: "Boolean"}))
	require.Equal(true, templateValueForParam(ParameterInfo{
		Key: "tls", Type: "Boolean", DefaultValue: utils.ToPtr("true"),
	}))

	// Number with no default -> 0; with default -> parsed float
	require.Equal(0, templateValueForParam(ParameterInfo{Key: "size", Type: "Float64"}))
	require.Equal(float64(2.5), templateValueForParam(ParameterInfo{
		Key: "size", Type: "Float64", DefaultValue: utils.ToPtr("2.5"),
	}))

	// Integer with default -> parsed int64
	require.Equal(int64(8080), templateValueForParam(ParameterInfo{
		Key: "port", Type: "Integer", DefaultValue: utils.ToPtr("8080"),
	}))

	// List with no default -> empty slice
	require.Equal([]any{}, templateValueForParam(ParameterInfo{Key: "zones", Type: "String", IsList: true}))

	// List with CSV default -> string slice
	require.Equal([]string{"a", "b"}, templateValueForParam(ParameterInfo{
		Key: "zones", Type: "String", IsList: true, DefaultValue: utils.ToPtr("a,b"),
	}))

	// Malformed numeric default -> falls back to raw string
	require.Equal("not-a-number", templateValueForParam(ParameterInfo{
		Key: "size", Type: "Float64", DefaultValue: utils.ToPtr("not-a-number"),
	}))
}

func TestBuildParamTemplate(t *testing.T) {
	require := require.New(t)

	params := []ParameterInfo{
		{Key: "databaseName", Type: "String", DefaultValue: utils.ToPtr("default")},
		{Key: "password", Type: "Password"},
		{Key: "port", Type: "Integer", DefaultValue: utils.ToPtr("3306")},
	}

	template := buildParamTemplate(params)

	require.Len(template, 3)
	require.Equal("default", template["databaseName"])
	require.Equal("", template["password"])
	require.Equal(int64(3306), template["port"])
}
```

- [ ] **Step 2: Run the tests to verify they fail**

Run: `go test ./cmd/instance/ -run 'TestTemplateValueForParam|TestBuildParamTemplate' -v`
Expected: FAIL — `undefined: templateValueForParam` / `undefined: buildParamTemplate`.

- [ ] **Step 3: Add the template helpers**

Add `"strconv"` to the import block, then add these functions to `cmd/instance/deployment_parameters.go`:

```go
// placeholderForType returns a typed zero value for a parameter with no default.
func placeholderForType(valueType string) any {
	switch strings.ToLower(strings.TrimSpace(valueType)) {
	case "bool", "boolean":
		return false
	case "int", "int32", "int64", "integer":
		return 0
	case "float", "float32", "float64", "double", "number":
		return 0
	case "object", "json", "map":
		return map[string]any{}
	default:
		return ""
	}
}

// coerceParamValue converts a string default into a typed JSON value based on the
// parameter type. On any parse failure it returns the raw string unchanged.
func coerceParamValue(value, valueType string, isList bool) any {
	value = strings.TrimSpace(value)
	if isList {
		if strings.HasPrefix(value, "[") {
			var parsed any
			if err := json.Unmarshal([]byte(value), &parsed); err == nil {
				return parsed
			}
			return value
		}
		parts := strings.Split(value, ",")
		values := make([]string, 0, len(parts))
		for _, part := range parts {
			part = strings.TrimSpace(part)
			if part != "" {
				values = append(values, part)
			}
		}
		return values
	}

	switch strings.ToLower(strings.TrimSpace(valueType)) {
	case "bool", "boolean":
		if b, err := strconv.ParseBool(value); err == nil {
			return b
		}
	case "int", "int32", "int64", "integer":
		if i, err := strconv.ParseInt(value, 10, 64); err == nil {
			return i
		}
	case "float", "float32", "float64", "double", "number":
		if f, err := strconv.ParseFloat(value, 64); err == nil {
			return f
		}
	case "object", "json", "map":
		var parsed any
		if err := json.Unmarshal([]byte(value), &parsed); err == nil {
			return parsed
		}
	}
	return value
}

// templateValueForParam returns the JSON template value for a single parameter:
// its (typed) default when present, otherwise a typed placeholder.
func templateValueForParam(p ParameterInfo) any {
	if p.DefaultValue != nil && strings.TrimSpace(*p.DefaultValue) != "" {
		return coerceParamValue(*p.DefaultValue, p.Type, p.IsList)
	}
	if p.IsList {
		return []any{}
	}
	return placeholderForType(p.Type)
}

// buildParamTemplate builds a key -> value map suitable for `instance create --param-file`.
func buildParamTemplate(params []ParameterInfo) map[string]any {
	template := make(map[string]any, len(params))
	for _, p := range params {
		template[p.Key] = templateValueForParam(p)
	}
	return template
}
```

- [ ] **Step 4: Run the tests to verify they pass**

Run: `go test ./cmd/instance/ -run 'TestTemplateValueForParam|TestBuildParamTemplate' -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add cmd/instance/deployment_parameters.go cmd/instance/deployment_parameters_test.go
git commit -m "feat: add create-params template value helpers

Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>"
```

---

### Task 3: Extract shared fetch helper and add the `template` subcommand

Factor the resolution + parameter assembly out of `runDeploymentParameters` into `fetchDeploymentParameters`, then add the `template` subcommand that builds and prints the JSON template.

**Files:**
- Modify: `cmd/instance/deployment_parameters.go`
- Modify: `cmd/instance/instance.go` (register the subcommand)
- Test: `cmd/instance/deployment_parameters_test.go`

**Interfaces:**
- Consumes: `getServiceAndPlanInfo` (Task 1), `buildParamTemplate` (Task 2), `dataaccess.DescribeServiceOffering`, `dataaccess.ListInputParameters`, `common.GetTokenWithLogin`, `config.CleanupArgsAndFlags`, `utils.PrintError`.
- Produces:
  - `func fetchDeploymentParameters(ctx context.Context, token, serviceName, planName, version, resourceName, environment string) (DeploymentParametersOutput, error)`
  - `var templateCmd *cobra.Command` (name `template`, registered under `deploymentParametersCmd`)
  - `func runDeploymentParametersTemplate(cmd *cobra.Command, args []string) error`

- [ ] **Step 1: Write the failing test**

Append to `cmd/instance/deployment_parameters_test.go` (add `"github.com/spf13/cobra"` to its imports):

```go
func TestDeploymentParametersTemplateCommand(t *testing.T) {
	require := require.New(t)

	// template subcommand is registered under deployment-parameters
	var found *cobra.Command
	for _, c := range deploymentParametersCmd.Commands() {
		if c.Name() == "template" {
			found = c
			break
		}
	}
	require.NotNil(found, "template subcommand should be registered")

	// required flags exist on the template subcommand
	require.NotNil(found.Flag("service"))
	require.NotNil(found.Flag("plan"))
	require.NotNil(found.Flag("resource"))
	require.NotNil(found.Flag("environment"))
	require.NotNil(found.Flag("version"))

	// parent command has the environment flag (from Task 1)
	require.NotNil(deploymentParametersCmd.Flag("environment"))
}
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `go test ./cmd/instance/ -run TestDeploymentParametersTemplateCommand -v`
Expected: FAIL — `found` is nil (template subcommand not registered).

- [ ] **Step 3: Extract `fetchDeploymentParameters`**

In `cmd/instance/deployment_parameters.go`, add this function. Move the body currently in `runDeploymentParameters` that runs between `getServiceAndPlanInfo` and the construction of the `output` variable (the version-resolution block, the `ListInputParameters` call, the `[]ParameterInfo` assembly, and the `sort.Slice`) into it:

```go
// fetchDeploymentParameters resolves the resource and returns its configurable
// deployment parameters. Shared by the table/json output path and the template path.
func fetchDeploymentParameters(ctx context.Context, token, serviceName, planName, version, resourceName, environment string) (DeploymentParametersOutput, error) {
	serviceID, resourceID, productTierID, productTierVersion, err := getServiceAndPlanInfo(ctx, token, serviceName, planName, resourceName, version, environment)
	if err != nil {
		return DeploymentParametersOutput{}, fmt.Errorf("failed to get service and plan info: %w", err)
	}

	// If we don't have a product tier version, get it from the service offering
	if productTierVersion == "" {
		serviceOfferingResult, err := dataaccess.DescribeServiceOffering(ctx, token, serviceID, productTierID, "")
		if err != nil {
			return DeploymentParametersOutput{}, fmt.Errorf("failed to describe service offering to get version: %w", err)
		}

		serviceOffering := serviceOfferingResult.GetConsumptionDescribeServiceOfferingResult()
		for _, offering := range serviceOffering.GetOfferings() {
			if offering.GetProductTierID() == productTierID {
				productTierVersion = offering.GetProductTierVersion()
				break
			}
		}

		if productTierVersion == "" {
			return DeploymentParametersOutput{}, fmt.Errorf("could not determine product tier version for plan %s", planName)
		}
	}

	parametersResult, err := dataaccess.ListInputParameters(ctx, token, serviceID, resourceID, productTierID, productTierVersion)
	if err != nil {
		return DeploymentParametersOutput{}, fmt.Errorf("failed to list input parameters: %w", err)
	}

	var parameters []ParameterInfo
	for _, param := range parametersResult.GetInputParameters() {
		var defaultValue *string
		if val, ok := param.GetDefaultValueOk(); ok && val != nil {
			defaultValue = val
		}

		var regex *string
		if val, ok := param.GetRegexOk(); ok && val != nil {
			regex = val
		}

		var options []string
		if param.GetOptions() != nil {
			options = param.GetOptions()
		}

		parameters = append(parameters, ParameterInfo{
			Key:          param.GetKey(),
			DisplayName:  param.GetName(),
			Description:  param.GetDescription(),
			Type:         param.GetType(),
			Required:     param.GetRequired(),
			Modifiable:   param.GetModifiable(),
			IsList:       param.GetIsList(),
			DefaultValue: defaultValue,
			Options:      options,
			Regex:        regex,
			Custom:       false,
			API:          "create",
		})
	}

	sort.Slice(parameters, func(i, j int) bool {
		return parameters[i].Key < parameters[j].Key
	})

	return DeploymentParametersOutput{
		ServiceName:  serviceName,
		PlanName:     planName,
		Version:      version,
		ResourceName: resourceName,
		Parameters:   parameters,
	}, nil
}
```

Now replace that moved block in `runDeploymentParameters` so it calls the helper. After the `environment` flag read (Task 1) and `token`/`ctx` setup, the body becomes:

```go
	output, err := fetchDeploymentParameters(ctx, token, serviceName, planName, version, resourceName, environment)
	if err != nil {
		return err
	}

	// Handle output format
	switch deploymentParamsOutputFlag {
	case "json":
		jsonData, err := json.MarshalIndent(output, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal output to JSON: %w", err)
		}
		fmt.Println(string(jsonData))
	case "table":
		printParametersTable(output)
	default:
		return fmt.Errorf("unsupported output format: %s. Supported formats are table and json", deploymentParamsOutputFlag)
	}

	return nil
```

(Delete the now-moved version-resolution, `ListInputParameters`, parameter-assembly, `sort.Slice`, and inline `output := DeploymentParametersOutput{...}` lines from `runDeploymentParameters`.)

- [ ] **Step 4: Add the `template` subcommand**

Add to `cmd/instance/deployment_parameters.go`:

```go
var templateCmd = &cobra.Command{
	Use:   "template --service=[service] --environment=[environment] --plan=[plan] --version=[version] --resource=[resource]",
	Short: "Generate a JSON parameter template for instance create",
	Long:  `This command generates a JSON parameter template for the given resource that can be filled in and passed to 'instance create --param-file'. Parameters with defaults are pre-populated; others get typed placeholders.`,
	Example: `  omnistrate-ctl instance deployment-parameters template --service=mysql --environment=dev --plan=mysql --version=latest --resource=mySQL > params.json
  omnistrate-ctl instance create --service=mysql --environment=dev --plan=mysql --resource=mySQL --cloud-provider=aws --region=us-east-2 --param-file params.json`,
	RunE:         runDeploymentParametersTemplate,
	SilenceUsage: true,
}

func runDeploymentParametersTemplate(cmd *cobra.Command, args []string) error {
	defer config.CleanupArgsAndFlags(cmd, &args)

	serviceName, err := cmd.Flags().GetString("service")
	if err != nil {
		utils.PrintError(err)
		return err
	}
	planName, err := cmd.Flags().GetString("plan")
	if err != nil {
		utils.PrintError(err)
		return err
	}
	version, err := cmd.Flags().GetString("version")
	if err != nil {
		utils.PrintError(err)
		return err
	}
	resourceName, err := cmd.Flags().GetString("resource")
	if err != nil {
		utils.PrintError(err)
		return err
	}
	environment, err := cmd.Flags().GetString("environment")
	if err != nil {
		utils.PrintError(err)
		return err
	}

	token, err := common.GetTokenWithLogin()
	if err != nil {
		return fmt.Errorf("failed to get token: %w", err)
	}

	output, err := fetchDeploymentParameters(context.Background(), token, serviceName, planName, version, resourceName, environment)
	if err != nil {
		return err
	}

	template := buildParamTemplate(output.Parameters)
	jsonData, err := json.MarshalIndent(template, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal template to JSON: %w", err)
	}
	fmt.Println(string(jsonData))

	return nil
}
```

Register the subcommand and its flags inside the existing `init()` (at the end):

```go
	templateCmd.Flags().String("service", "", "Service name")
	templateCmd.Flags().String("plan", "", "Service plan name")
	templateCmd.Flags().String("version", "preferred", "Service plan version (latest|preferred|1.0 etc.)")
	templateCmd.Flags().String("resource", "", "Resource name")
	templateCmd.Flags().String("environment", "", "Environment name or ID")

	for _, required := range []string{"service", "plan", "resource", "environment"} {
		if err := templateCmd.MarkFlagRequired(required); err != nil {
			utils.PrintError(err)
		}
	}

	deploymentParametersCmd.AddCommand(templateCmd)
```

- [ ] **Step 5: Run the test to verify it passes**

Run: `go test ./cmd/instance/ -run TestDeploymentParametersTemplateCommand -v`
Expected: PASS.

- [ ] **Step 6: Run the full instance package test suite and build**

Run: `go test ./cmd/instance/... && go build ./...`
Expected: PASS, build succeeds with no output.

- [ ] **Step 7: Commit**

```bash
git add cmd/instance/deployment_parameters.go cmd/instance/deployment_parameters_test.go cmd/instance/instance.go
git commit -m "feat: add 'instance deployment-parameters template' subcommand

Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>"
```

---

## Notes for the implementer

- `cmd/instance/instance.go` already calls `Cmd.AddCommand(deploymentParametersCmd)`. The `template` subcommand is registered onto `deploymentParametersCmd` in Task 3 Step 4, so no change to `instance.go` is strictly required — but verify the registration line landed and the import set still compiles. If you prefer to register in `instance.go` for consistency with sibling commands, that is acceptable; do not register in both places.
- The `template` output is the raw `key -> value` map (no surrounding metadata) so it can be piped straight into `instance create --param-file`.
- Keep the existing `--output=json` schema of the parent command unchanged.
