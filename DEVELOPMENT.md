# Development Guide — Adding New CLI Commands

This guide explains the `omnistrate-ctl` project structure and provides step-by-step instructions for adding new CLI commands backed by Omnistrate SDK operations.

## Project Architecture

```
omnistrate-ctl/
├── main.go                          # Entry point — calls cmd.Execute()
├── cmd/
│   ├── root.go                      # Root cobra command, wires all top-level commands
│   ├── common/
│   │   ├── token.go                 # GetTokenWithLogin() — auth helper
│   │   └── flags.go                 # Shared flag helpers (FormatParams, etc.)
│   ├── <entity>/                    # One package per entity (service, secret, instance, etc.)
│   │   ├── <entity>.go             # Parent command + init() wiring subcommands
│   │   ├── list.go                 # List subcommand
│   │   ├── describe.go             # Describe/Get subcommand
│   │   ├── create.go               # Create subcommand (if applicable)
│   │   └── delete.go               # Delete subcommand
│   └── ...
├── internal/
│   ├── dataaccess/
│   │   ├── client.go               # HTTP client setup (V1 + Fleet API clients)
│   │   ├── <entity>.go             # API call functions per entity
│   │   └── ...
│   ├── model/
│   │   ├── <entity>.go             # Display structs with JSON tags
│   │   └── ...
│   ├── config/                      # Configuration, auth, env vars
│   └── utils/                       # Output formatting, filtering, error handling
├── doc-gen/main.go                  # Auto-generates CLI docs from cobra commands
├── Makefile                         # Build, test, lint, doc-gen targets
└── go.mod                           # Dependencies (includes omnistrate-sdk-go)
```

## SDK Clients

The project uses two OpenAPI-generated SDK clients from `github.com/omnistrate-oss/omnistrate-sdk-go`:

| Client | Package Import Alias | Getter Function | Use Case |
|--------|---------------------|-----------------|----------|
| **V1 API** | `openapiclientv1 "github.com/omnistrate-oss/omnistrate-sdk-go/v1"` | `getV1Client()` | Service management: services, secrets, environments, service plans, subscriptions, etc. |
| **Fleet API** | `openapiclientfleet "github.com/omnistrate-oss/omnistrate-sdk-go/fleet"` | `getFleetClient()` | Fleet/operational: inventory, resource instances, workflows, custom networks, operations, costs, etc. |

### Available V1 SDK API Groups

Each group is accessible as a field on the V1 API client (e.g., `apiClient.ServiceApiAPI`):

- `AccountConfigApiAPI` — Account configuration
- `AuditEventsApiAPI` — Audit event tracking
- `CloudProviderApiAPI` — Cloud provider operations
- `ComposeGenApiAPI` — Compose spec generation
- `ComputeConfigApiAPI` — Compute configuration
- `CustomDomainApiAPI` — Custom domain management
- `CustomNetworkApiAPI` — Custom network management
- `DeploymentConfigApiAPI` — Deployment configuration
- `ExpressionEvaluatorApiAPI` — Expression evaluation
- `HelmPackageApiAPI` — Helm package management
- `InputParameterApiAPI` / `OutputParameterApiAPI` — Parameter management
- `ProductTierApiAPI` — Product tier management
- `ResourceApiAPI` — Resource management
- `ResourceInstanceApiAPI` — Resource instance management
- `SecretsApiAPI` — Secrets management
- `ServiceApiAPI` — Service CRUD
- `ServiceEnvironmentApiAPI` — Service environment management
- `ServiceModelApiAPI` — Service model management
- `ServiceOfferingApiAPI` — Service offering management
- `ServicePlanApiAPI` — Service plan management
- `ServicesOrchestrationApiAPI` — Services orchestration
- `SigninApiAPI` — Authentication
- `SpOrganizationApiAPI` — Service provider organization
- `SubscriptionApiAPI` — Subscription management
- `TierVersionSetApiAPI` — Tier version set management
- `UsersApiAPI` — User management

### Available Fleet SDK API Groups

Accessible on the Fleet API client (e.g., `apiClient.InventoryApiAPI`):

- `AuditEventsApiAPI` — Fleet audit events
- `CostApiAPI` — Cost management
- `EventsApiAPI` — Event tracking
- `FleetCustomNetworkApiAPI` — Fleet custom networks
- `FleetWorkflowsApiAPI` — Workflow management
- `HelmPackageApiAPI` — Fleet helm operations
- `HostclusterApiAPI` — Host cluster management
- `InventoryApiAPI` — Inventory search and management
- `NotificationsApiAPI` — Notification management
- `OperationsApiAPI` — Operations management

## Request Flow

Every CLI command follows this flow:

```
User runs command
  → Cobra parses flags/args
  → runXxx() function executes:
      1. defer config.CleanupArgsAndFlags(cmd, &args)
      2. Parse flags via cmd.Flags().GetString("flag-name")
      3. Get auth token via common.GetTokenWithLogin()
      4. (Optional) Start spinner for non-JSON output
      5. Call dataaccess.XxxFunction() — makes SDK API call
      6. Convert SDK response → model.Xxx struct(s)
      7. Print via utils.PrintTextTableJsonOutput() or utils.PrintTextTableJsonArrayOutput()
```

## Step-by-Step: Adding a New Entity Command

This walkthrough uses a hypothetical entity called `widget` with list, describe, and delete operations.

### Step 1: Create the Model (`internal/model/widget.go`)

Define a display struct with JSON tags. Fields should be strings for table rendering.

```go
package model

type Widget struct {
    ID          string `json:"id"`
    Name        string `json:"name"`
    Status      string `json:"status"`
    Description string `json:"description"`
}
```

### Step 2: Create Data Access Functions (`internal/dataaccess/widget.go`)

Wrap SDK calls. Use `getV1Client()` for V1 APIs or `getFleetClient()` for Fleet APIs.

```go
package dataaccess

import (
    "context"

    openapiclientv1 "github.com/omnistrate-oss/omnistrate-sdk-go/v1"
)

// ListWidgets retrieves all widgets
func ListWidgets(ctx context.Context, token string) (*openapiclientv1.ListWidgetsResult, error) {
    ctxWithToken := context.WithValue(ctx, openapiclientv1.ContextAccessToken, token)
    apiClient := getV1Client()

    resp, r, err := apiClient.WidgetApiAPI.WidgetApiListWidgets(ctxWithToken).Execute()
    defer func() {
        if r != nil {
            _ = r.Body.Close()
        }
    }()
    if err != nil {
        return nil, handleV1Error(err)
    }
    return resp, nil
}

// DescribeWidget retrieves details for a specific widget
func DescribeWidget(ctx context.Context, token, widgetID string) (*openapiclientv1.DescribeWidgetResult, error) {
    ctxWithToken := context.WithValue(ctx, openapiclientv1.ContextAccessToken, token)
    apiClient := getV1Client()

    resp, r, err := apiClient.WidgetApiAPI.WidgetApiDescribeWidget(ctxWithToken, widgetID).Execute()
    defer func() {
        if r != nil {
            _ = r.Body.Close()
        }
    }()
    if err != nil {
        return nil, handleV1Error(err)
    }
    return resp, nil
}

// DeleteWidget deletes a widget by ID
func DeleteWidget(ctx context.Context, token, widgetID string) error {
    ctxWithToken := context.WithValue(ctx, openapiclientv1.ContextAccessToken, token)
    apiClient := getV1Client()

    r, err := apiClient.WidgetApiAPI.WidgetApiDeleteWidget(ctxWithToken, widgetID).Execute()
    defer func() {
        if r != nil {
            _ = r.Body.Close()
        }
    }()
    if err != nil {
        return handleV1Error(err)
    }
    return nil
}
```

**Key patterns:**
- Always create `ctxWithToken` using `context.WithValue` with the access token
- Always `defer` closing `r.Body` with a nil check
- Use `handleV1Error()` for V1 client errors, `handleFleetError()` for Fleet client errors
- Functions that return data return `(*SdkResponseType, error)`
- Functions that mutate without response return `error`

### Step 3: Create the Parent Command (`cmd/widget/widget.go`)

```go
package widget

import (
    "github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
    Use:          "widget [operation] [flags]",
    Short:        "Manage widgets",
    Long:         `This command helps you manage widgets for your account.`,
    Run:          runWidget,
    SilenceUsage: true,
}

func init() {
    Cmd.AddCommand(listCmd)
    Cmd.AddCommand(describeCmd)
    Cmd.AddCommand(deleteCmd)
}

func runWidget(cmd *cobra.Command, args []string) {
    err := cmd.Help()
    if err != nil {
        return
    }
}
```

### Step 4: Create Subcommands

#### `cmd/widget/list.go`

```go
package widget

import (
    "github.com/chelnak/ysmrr"
    "github.com/omnistrate-oss/omnistrate-ctl/cmd/common"
    "github.com/omnistrate-oss/omnistrate-ctl/internal/config"
    "github.com/omnistrate-oss/omnistrate-ctl/internal/dataaccess"
    "github.com/omnistrate-oss/omnistrate-ctl/internal/model"
    "github.com/omnistrate-oss/omnistrate-ctl/internal/utils"
    "github.com/spf13/cobra"
)

const listExample = `# List all widgets
omnistrate-ctl widget list

# List widgets with JSON output
omnistrate-ctl widget list --output json`

var listCmd = &cobra.Command{
    Use:          "list [flags]",
    Short:        "List widgets",
    Long:         `This command lists all widgets for your account.`,
    Example:      listExample,
    RunE:         runList,
    SilenceUsage: true,
}

func init() {
    listCmd.Args = cobra.NoArgs
}

func runList(cmd *cobra.Command, args []string) error {
    defer config.CleanupArgsAndFlags(cmd, &args)

    output, _ := cmd.Flags().GetString("output")

    token, err := common.GetTokenWithLogin()
    if err != nil {
        utils.PrintError(err)
        return err
    }

    // Spinner for non-JSON output
    var sm ysmrr.SpinnerManager
    var spinner *ysmrr.Spinner
    if output != "json" {
        sm = ysmrr.NewSpinnerManager()
        spinner = sm.AddSpinner("Listing widgets...")
        sm.Start()
    }

    result, err := dataaccess.ListWidgets(cmd.Context(), token)
    if err != nil {
        utils.HandleSpinnerError(spinner, sm, err)
        return err
    }

    widgets := make([]model.Widget, 0, len(result.GetWidgets()))
    for _, w := range result.GetWidgets() {
        widgets = append(widgets, model.Widget{
            ID:     w.GetId(),
            Name:   w.GetName(),
            Status: w.GetStatus(),
        })
    }

    if len(widgets) == 0 {
        utils.HandleSpinnerSuccess(spinner, sm, "No widgets found")
    } else {
        utils.HandleSpinnerSuccess(spinner, sm, "Successfully retrieved widgets")
    }

    return utils.PrintTextTableJsonArrayOutput(output, widgets)
}
```

#### `cmd/widget/describe.go`

```go
package widget

import (
    "github.com/omnistrate-oss/omnistrate-ctl/cmd/common"
    "github.com/omnistrate-oss/omnistrate-ctl/internal/config"
    "github.com/omnistrate-oss/omnistrate-ctl/internal/dataaccess"
    "github.com/omnistrate-oss/omnistrate-ctl/internal/model"
    "github.com/omnistrate-oss/omnistrate-ctl/internal/utils"
    "github.com/spf13/cobra"
)

const describeExample = `# Describe a widget
omnistrate-ctl widget describe --id widget-123`

var describeCmd = &cobra.Command{
    Use:          "describe [flags]",
    Short:        "Describe a widget",
    Long:         `This command retrieves details for a specific widget.`,
    Example:      describeExample,
    RunE:         runDescribe,
    SilenceUsage: true,
}

func init() {
    describeCmd.Flags().String("id", "", "Widget ID (required)")
    describeCmd.MarkFlagRequired("id")
    describeCmd.Args = cobra.NoArgs
}

func runDescribe(cmd *cobra.Command, args []string) error {
    defer config.CleanupArgsAndFlags(cmd, &args)

    widgetID, _ := cmd.Flags().GetString("id")
    output, _ := cmd.Flags().GetString("output")

    token, err := common.GetTokenWithLogin()
    if err != nil {
        utils.PrintError(err)
        return err
    }

    result, err := dataaccess.DescribeWidget(cmd.Context(), token, widgetID)
    if err != nil {
        utils.PrintError(err)
        return err
    }

    widget := model.Widget{
        ID:          result.GetId(),
        Name:        result.GetName(),
        Status:      result.GetStatus(),
        Description: result.GetDescription(),
    }

    return utils.PrintTextTableJsonOutput(output, widget)
}
```

#### `cmd/widget/delete.go`

```go
package widget

import (
    "fmt"

    "github.com/omnistrate-oss/omnistrate-ctl/cmd/common"
    "github.com/omnistrate-oss/omnistrate-ctl/internal/config"
    "github.com/omnistrate-oss/omnistrate-ctl/internal/dataaccess"
    "github.com/omnistrate-oss/omnistrate-ctl/internal/utils"
    "github.com/spf13/cobra"
)

const deleteExample = `# Delete a widget
omnistrate-ctl widget delete --id widget-123`

var deleteCmd = &cobra.Command{
    Use:          "delete [flags]",
    Short:        "Delete a widget",
    Long:         `This command deletes a widget by its ID.`,
    Example:      deleteExample,
    RunE:         runDelete,
    SilenceUsage: true,
}

func init() {
    deleteCmd.Flags().String("id", "", "Widget ID (required)")
    deleteCmd.MarkFlagRequired("id")
    deleteCmd.Args = cobra.NoArgs
}

func runDelete(cmd *cobra.Command, args []string) error {
    defer config.CleanupArgsAndFlags(cmd, &args)

    widgetID, _ := cmd.Flags().GetString("id")

    token, err := common.GetTokenWithLogin()
    if err != nil {
        utils.PrintError(err)
        return err
    }

    err = dataaccess.DeleteWidget(cmd.Context(), token, widgetID)
    if err != nil {
        utils.PrintError(err)
        return err
    }

    fmt.Printf("Successfully deleted widget '%s'\n", widgetID)
    return nil
}
```

### Step 5: Register in Root Command (`cmd/root.go`)

Add the import and registration:

```go
import (
    // ... existing imports ...
    "github.com/omnistrate-oss/omnistrate-ctl/cmd/widget"
)

func init() {
    // ... existing commands ...
    RootCmd.AddCommand(widget.Cmd)
}
```

### Step 6: Create Unit Tests (`cmd/widget/widget_test.go`)

Every new command package **must** have unit tests that verify command structure, subcommand registration, and flag definitions. Pure logic functions should use table-driven tests.

```go
package widget

import (
    "testing"

    "github.com/stretchr/testify/require"
)

func TestWidgetCommands(t *testing.T) {
    require := require.New(t)

    // Verify parent command metadata
    require.Equal("widget [operation] [flags]", Cmd.Use)
    require.Equal("Manage widgets", Cmd.Short)
    require.Contains(Cmd.Long, "manage widgets")

    // Verify all subcommands are registered
    expectedCommands := []string{"list", "describe", "delete"}
    actualCommands := make([]string, 0)
    for _, cmd := range Cmd.Commands() {
        actualCommands = append(actualCommands, cmd.Name())
    }
    for _, expected := range expectedCommands {
        require.Contains(actualCommands, expected, "Expected subcommand %s not found", expected)
    }
}

func TestListCommandFlags(t *testing.T) {
    require := require.New(t)

    // list command should have no required flags
    require.Equal("list [flags]", listCmd.Use)
    require.Equal("List widgets", listCmd.Short)
}

func TestDescribeCommandFlags(t *testing.T) {
    require := require.New(t)

    // Verify required flags exist with correct types
    idFlag := describeCmd.Flags().Lookup("id")
    require.NotNil(idFlag, "Expected flag 'id' not found")
    require.Equal("string", idFlag.Value.Type())
    require.Equal("", idFlag.DefValue)
}

func TestDeleteCommandFlags(t *testing.T) {
    require := require.New(t)

    idFlag := deleteCmd.Flags().Lookup("id")
    require.NotNil(idFlag, "Expected flag 'id' not found")
    require.Equal("string", idFlag.Value.Type())
}
```

#### Testing Pure Logic Functions

If your commands contain helper functions with business logic (parsing, formatting, validation), test them with table-driven tests:

```go
func TestFormatWidgetStatus(t *testing.T) {
    tests := []struct {
        name     string
        input    string
        expected string
    }{
        {"active status", "ACTIVE", "active"},
        {"inactive status", "INACTIVE", "inactive"},
        {"empty status", "", "unknown"},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result := formatWidgetStatus(tt.input)
            if result != tt.expected {
                t.Errorf("expected %q, got %q", tt.expected, result)
            }
        })
    }
}
```

### Step 7: Create Smoke Tests (`test/smoke_test/widget/widget_test.go`)

Smoke tests exercise the full CLI end-to-end against a live API. They are guarded by the `ENABLE_SMOKE_TEST` environment variable and require `TEST_EMAIL` and `TEST_PASSWORD` to be set.

```go
package widget

import (
    "context"
    "fmt"
    "testing"

    "github.com/omnistrate-oss/omnistrate-ctl/cmd"
    "github.com/omnistrate-oss/omnistrate-ctl/test/testutils"

    "github.com/stretchr/testify/require"
)

func Test_widget_basic(t *testing.T) {
    testutils.SmokeTest(t)

    ctx := context.TODO()
    require := require.New(t)
    defer testutils.Cleanup()

    // Login
    testEmail, testPassword, err := testutils.GetTestAccount()
    require.NoError(err)
    cmd.RootCmd.SetArgs([]string{
        "login",
        fmt.Sprintf("--email=%s", testEmail),
        fmt.Sprintf("--password=%s", testPassword),
    })
    err = cmd.RootCmd.ExecuteContext(ctx)
    require.NoError(err)

    // PASS: list widgets
    cmd.RootCmd.SetArgs([]string{"widget", "list"})
    err = cmd.RootCmd.ExecuteContext(ctx)
    require.NoError(err)

    // PASS: list widgets with JSON output
    cmd.RootCmd.SetArgs([]string{"widget", "list", "--output", "json"})
    err = cmd.RootCmd.ExecuteContext(ctx)
    require.NoError(err)
}
```

### Step 8: Create Integration Tests (`test/integration_test/dataaccess/widget_test.go`)

Integration tests call internal Go functions directly (not via CLI) against a live API. They are guarded by `ENABLE_INTEGRATION_TEST`. Use table-driven tests with `wantErr` patterns.

```go
package dataaccess

import (
    "context"
    "testing"

    "github.com/omnistrate-oss/omnistrate-ctl/internal/dataaccess"
    "github.com/omnistrate-oss/omnistrate-ctl/test/testutils"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func TestListWidgets(t *testing.T) {
    testutils.IntegrationTest(t)

    testEmail, testPassword, err := testutils.GetTestAccount()
    require.NoError(t, err)

    ctx := context.TODO()
    token, err := dataaccess.LoginWithPassword(ctx, testEmail, testPassword)
    require.NoError(t, err)
    require.NotEmpty(t, token)

    // Test listing widgets with valid token
    result, err := dataaccess.ListWidgets(ctx, token)
    assert.NoError(t, err)
    assert.NotNil(t, result)

    // Test with invalid token
    _, err = dataaccess.ListWidgets(ctx, "invalid-token")
    assert.Error(t, err)
}
```

### Step 9: Build and Generate Docs

```bash
make build       # Verify it compiles
make unit-test   # Run unit tests
make gen-doc     # Regenerate CLI documentation
# Or run everything:
make all
```

## Testing Guide

### Test Categories

| Category | Location | Guard | Run Command |
|----------|----------|-------|-------------|
| **Unit tests** | `cmd/<entity>/*_test.go`, `internal/**/*_test.go` | None | `make unit-test` |
| **Smoke tests** | `test/smoke_test/<entity>/*_test.go` | `ENABLE_SMOKE_TEST=true` | `make smoke-test` |
| **Integration tests** | `test/integration_test/**/*_test.go` | `ENABLE_INTEGRATION_TEST=true` | `make integration-test` |

### Unit Test Patterns

Unit tests run without any external dependencies and verify:

1. **Command structure** — parent command metadata (`Use`, `Short`, `Long`), subcommand registration
2. **Flag definitions** — flag existence, type, default value, shorthand
3. **Pure logic functions** — parsing, formatting, validation using table-driven tests

**Conventions:**
- Use `testify/require` for assertions that must pass (fatal on failure)
- Use `testify/assert` for non-fatal assertions
- Place test files in the same package as the code being tested (e.g., `package widget`)
- Name test files `*_test.go` (e.g., `widget_test.go`, `common_test.go`)
- Use table-driven tests for any function with multiple input/output scenarios
- Test both valid and invalid inputs, including edge cases

**Example — Table-driven test with error cases:**

```go
tests := []struct {
    name          string
    input         string
    expected      string
    expectError   bool
    errorContains string
}{
    {"valid input", "hello", "HELLO", false, ""},
    {"empty input", "", "", true, "input cannot be empty"},
}
for _, tt := range tests {
    t.Run(tt.name, func(t *testing.T) {
        result, err := transform(tt.input)
        if tt.expectError {
            require.Error(t, err)
            require.Contains(t, err.Error(), tt.errorContains)
            return
        }
        require.NoError(t, err)
        require.Equal(t, tt.expected, result)
    })
}
```

### Smoke Test Patterns

Smoke tests exercise the CLI end-to-end by setting args via `cmd.RootCmd.SetArgs()` and calling `cmd.RootCmd.ExecuteContext()`.

**Conventions:**
- Always start with `testutils.SmokeTest(t)` guard
- Always `defer testutils.Cleanup()` to remove config dir
- Always login first using `testutils.GetTestAccount()`
- Test CRUD lifecycle: create → describe → list → update → delete
- Use `testutils.WaitForInstanceToReachStatus()` for async operations
- Test with both default and JSON output formats

### Integration Test Patterns

Integration tests call `internal/dataaccess` functions directly against the API.

**Conventions:**
- Always start with `testutils.IntegrationTest(t)` guard
- Use table-driven tests with `wantErr` and `expectedErrMsg` fields
- Test valid and invalid credentials, missing required parameters
- Use `assert` for non-fatal checks within subtests, `require` for setup

### Test Utilities (`test/testutils/`)

| Function | Description |
|----------|-------------|
| `SmokeTest(t)` | Skips test unless `ENABLE_SMOKE_TEST=true` |
| `IntegrationTest(t)` | Skips test unless `ENABLE_INTEGRATION_TEST=true` |
| `GetTestAccount()` | Returns `(email, password, error)` from env vars `TEST_EMAIL`, `TEST_PASSWORD` |
| `Cleanup()` | Removes the config directory |
| `WaitForInstanceToReachStatus(ctx, id, status)` | Polls instance status with exponential backoff |

### Running Tests

```bash
# Run all unit tests
make unit-test

# Run a single test
go test ./cmd/widget/ -run TestWidgetCommands

# Run smoke tests (requires API access)
export ENABLE_SMOKE_TEST=true
export TEST_EMAIL="your-email"
export TEST_PASSWORD="your-password"
make smoke-test

# Run integration tests (requires API access)
export ENABLE_INTEGRATION_TEST=true
export TEST_EMAIL="your-email"
export TEST_PASSWORD="your-password"
make integration-test
```

## Finding SDK Operations

To discover which SDK operations are available for a new entity:

1. **Browse the SDK source** in the Go module cache:
   ```bash
   # List all V1 API files
   ls $(go env GOMODCACHE)/github.com/omnistrate-oss/omnistrate-sdk-go@v0.0.90/v1/api_*.go

   # List all Fleet API files
   ls $(go env GOMODCACHE)/github.com/omnistrate-oss/omnistrate-sdk-go@v0.0.90/fleet/api_*.go
   ```

2. **Read an API file** to see available methods:
   ```bash
   # Example: see what operations exist for secrets
   grep "func (a \*SecretsApiAPIService)" \
     $(go env GOMODCACHE)/github.com/omnistrate-oss/omnistrate-sdk-go@v0.0.90/v1/api_secrets_api.go
   ```

3. **Find request/response types** for a specific operation:
   ```bash
   # Search for model types related to an entity
   ls $(go env GOMODCACHE)/github.com/omnistrate-oss/omnistrate-sdk-go@v0.0.90/v1/model_*.go | grep -i widget
   ```

4. **Check existing dataaccess files** for patterns to follow:
   ```bash
   ls internal/dataaccess/
   ```

## Common Patterns and Conventions

### Flag vs Positional Arguments

- **Flags** (preferred for most cases): `cmd.Flags().String("id", "", "Widget ID")`
- **Positional args** (for simple commands): `cobra.ExactArgs(2)` with `args[0]`, `args[1]`
- Use `cmd.MarkFlagRequired("flag-name")` for required flags

### Output Formatting

The global `--output` flag (`-o`) supports three formats:
- `table` (default) — formatted table via `go-pretty`
- `text` — indented JSON-like text
- `json` — raw JSON

Use these helpers from `internal/utils/print.go`:
- `PrintTextTableJsonOutput(output, singleObject)` — for single items
- `PrintTextTableJsonArrayOutput(output, sliceOfObjects)` — for lists

### Spinner Pattern

For long-running operations, show a spinner in non-JSON mode:

```go
var sm ysmrr.SpinnerManager
var spinner *ysmrr.Spinner
if output != "json" {
    sm = ysmrr.NewSpinnerManager()
    spinner = sm.AddSpinner("Loading...")
    sm.Start()
}
// ... do work ...
utils.HandleSpinnerSuccess(spinner, sm, "Done")
// On error:
utils.HandleSpinnerError(spinner, sm, err)
```

### Error Handling

- Always call `utils.PrintError(err)` before returning errors — it prints a red-colored message
- Use `handleV1Error(err)` / `handleFleetError(err)` in dataaccess to extract API error details
- Always `return err` after printing so cobra captures the exit code

### Filtering Support

To add filter support to a list command (see `cmd/service/list.go` for a full example):

```go
// In init():
listCmd.Flags().StringArrayP("filter", "f", []string{}, "Filter expression")

// In runList():
filters, _ := cmd.Flags().GetStringArray("filter")
filterMaps, err := utils.ParseFilters(filters, utils.GetSupportedFilterKeys(model.Widget{}))
// Then for each item:
match, err := utils.MatchesFilters(widget, filterMaps)
```

## Build Commands Reference

| Command | Description |
|---------|-------------|
| `make build` | Build binary for current OS/arch to `dist/` |
| `make unit-test` | Run all unit tests with coverage |
| `make lint` | Run golangci-lint |
| `make gen-doc` | Regenerate CLI docs in `mkdocs/docs/` |
| `make all` | Full pipeline: tidy, pretty, build, test, lint, check-deps, gen-doc |
| `make ctl` | Build binaries for all 6 platforms |
| `go test ./path/to/package -run TestName` | Run a single test |

## Checklist for New Commands

- [ ] Create `internal/model/<entity>.go` with display struct
- [ ] Create `internal/dataaccess/<entity>.go` with SDK wrapper functions
- [ ] Create `cmd/<entity>/` directory with parent command and subcommands
- [ ] Register the parent command in `cmd/root.go`
- [ ] Add examples to each subcommand
- [ ] Create unit tests in `cmd/<entity>/<entity>_test.go` (command structure, flags, pure logic)
- [ ] Create smoke tests in `test/smoke_test/<entity>/<entity>_test.go` (end-to-end CLI)
- [ ] Create integration tests in `test/integration_test/dataaccess/<entity>_test.go` (direct API)
- [ ] Run `make build` to verify compilation
- [ ] Run `make unit-test` to verify all tests pass
- [ ] Run `make gen-doc` to regenerate documentation
