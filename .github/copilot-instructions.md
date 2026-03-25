# GitHub Copilot Instructions for omnistrate-ctl

## Project Overview

omnistrate-ctl is a Go CLI built with [cobra](https://github.com/spf13/cobra) for managing Omnistrate SaaS deployments. It follows a three-layer architecture:

- **`cmd/`** — Cobra command definitions (one package per entity)
- **`internal/dataaccess/`** — SDK API wrapper functions
- **`internal/model/`** — Display structs for output formatting
- **`internal/config/`** — Configuration, auth, environment variables
- **`internal/utils/`** — Output formatting, filtering, error handling

For the full architecture guide, see [DEVELOPMENT.md](../DEVELOPMENT.md).

## Build Commands

```bash
make build          # Build binary for current OS/arch
make unit-test      # Run all unit tests with coverage
make lint           # Run golangci-lint
make gen-doc        # Regenerate CLI docs in mkdocs/docs/
make all            # Full pipeline: tidy, pretty, build, test, lint, check-deps, gen-doc
go test ./path/to/package -run TestName   # Run a single test
```

## Documentation Generation

**CRITICAL**: After making ANY changes to CLI commands, flags, or help text, you MUST run `make gen-doc` to regenerate documentation. This keeps `mkdocs/docs/` synchronized with CLI behavior.

## Code Style

- Follow Go standard naming: camelCase for unexported, PascalCase for exported
- Format imports: stdlib first, then external, then internal
- Use `testify/require` for test assertions
- Use cobra for CLI commands with consistent flags/args patterns
- Use tabwriter for formatted terminal output
- Wrap errors with context when needed

## Adding New CLI Commands

See [DEVELOPMENT.md](../DEVELOPMENT.md) for the complete step-by-step guide. Summary:

1. Create `internal/model/<entity>.go` — display struct with JSON tags
2. Create `internal/dataaccess/<entity>.go` — SDK wrapper functions
3. Create `cmd/<entity>/` — parent command + subcommands
4. Register in `cmd/root.go`
5. Create unit tests in `cmd/<entity>/<entity>_test.go`
6. Create smoke tests in `test/smoke_test/<entity>/`
7. Create integration tests in `test/integration_test/dataaccess/`
8. Run `make build && make unit-test && make gen-doc`

## Testing Requirements

**All new functionality MUST include tests at all three levels:**

### Unit Tests (`cmd/<entity>/*_test.go`, `internal/**/*_test.go`)
- Verify command structure (Use, Short, Long), subcommand registration, flag definitions
- Test pure logic functions with table-driven tests
- Use `testify/require` for assertions
- Reference: `cmd/operations/operations_test.go`, `cmd/instance/common_test.go`

### Smoke Tests (`test/smoke_test/<entity>/*_test.go`)
- Guard with `testutils.SmokeTest(t)`, cleanup with `defer testutils.Cleanup()`
- Exercise full CLI via `cmd.RootCmd.SetArgs()` + `ExecuteContext()`
- Test CRUD lifecycle end-to-end
- Reference: `test/smoke_test/secret/secret_test.go`

### Integration Tests (`test/integration_test/dataaccess/*_test.go`)
- Guard with `testutils.IntegrationTest(t)`
- Call `dataaccess` functions directly with table-driven tests
- Test valid operations, invalid tokens, missing parameters
- Reference: `test/integration_test/dataaccess/signin_test.go`

## SDK Clients

Two OpenAPI-generated clients from `github.com/omnistrate-oss/omnistrate-sdk-go`:

- **V1 API** (`getV1Client()`): Service management — services, secrets, environments, plans, subscriptions
- **Fleet API** (`getFleetClient()`): Fleet/operational — inventory, instances, workflows, custom networks, operations

## Key Patterns

### Data Access Function Pattern
```go
func ListWidgets(ctx context.Context, token string) (*openapiclientv1.ListWidgetsResult, error) {
    ctxWithToken := context.WithValue(ctx, openapiclientv1.ContextAccessToken, token)
    apiClient := getV1Client()
    resp, r, err := apiClient.WidgetApiAPI.WidgetApiListWidgets(ctxWithToken).Execute()
    defer func() { if r != nil { _ = r.Body.Close() } }()
    if err != nil { return nil, handleV1Error(err) }
    return resp, nil
}
```

### Command Run Function Pattern
```go
func runList(cmd *cobra.Command, args []string) error {
    defer config.CleanupArgsAndFlags(cmd, &args)
    output, _ := cmd.Flags().GetString("output")
    token, err := common.GetTokenWithLogin()
    // ... call dataaccess, convert to model, print output
}
```

