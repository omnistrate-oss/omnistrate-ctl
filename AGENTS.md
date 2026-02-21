# CLAUDE.md - CLI Development Guidelines

## Key Documentation

**Read [DEVELOPMENT.md](DEVELOPMENT.md)** for the full architecture guide, step-by-step walkthrough for adding new commands, and SDK API reference. The sections below are a quick reference.

## Build Commands

- Build: `make build`
- Run all tests: `make unit-test`
- Run single test: `go test ./path/to/package -run TestName`
- Run smoke tests: `make smoke-test` (requires TEST_EMAIL, TEST_PASSWORD)
- Lint code: `make lint` (install with `make lint-install`)
- Format code: `make pretty` (uses prettier and go fmt)
- Generate docs: `make gen-doc` (regenerates CLI documentation)
- Run everything: `make all` (includes tidy, build, test, lint, check-dependencies, gen-doc, pretty)

## Documentation Generation Requirements

**IMPORTANT**: After making ANY changes to CLI commands, flags, or help text, you MUST run `make gen-doc` to regenerate the documentation. This ensures the docs in `mkdocs/docs/` stay synchronized with the actual CLI behavior.

- The `gen-doc` target runs `go run doc-gen/main.go` which auto-generates markdown files
- Documentation files are automatically removed and regenerated to stay current
- Always run `make gen-doc` or `make all` after modifying:
  - Command definitions in `cmd/` directory
  - Flag descriptions or help text
  - Command usage examples
  - Any cobra command configurations

## Code Style Guidelines

- Follow Go standard naming conventions: camelCase for variables, PascalCase for exports
- Use error handling with appropriate checks - wrap errors with context when needed
- Format imports with standard Go style (stdlib first, then external, then internal)
- Use descriptive variable/function names and maintain consistent indentation
- Add tests for all new functionality and maintain high coverage
- Follow project structure with cmd/ for commands and internal/ for implementation
- Commit messages should be clear and descriptive (feature/fix/chore: message)

## Project-Specific Patterns

- Prefer functional options for configuration
- Use cobra for CLI commands with consistent flags/args pattern
- Use tabwriter for formatted terminal output

## Adding New CLI Commands

Refer to **DEVELOPMENT.md** for the full guide. Summary of the three-layer architecture:

1. **Model** (`internal/model/<entity>.go`): Display struct with JSON tags
2. **Data Access** (`internal/dataaccess/<entity>.go`): SDK API wrapper functions
3. **Command** (`cmd/<entity>/`): Cobra commands with parent + subcommands

### Creating a New Entity — Quick Checklist

1. Create `internal/model/<entity>.go` — display struct
2. Create `internal/dataaccess/<entity>.go` — SDK wrapper functions using `getV1Client()` or `getFleetClient()`
3. Create `cmd/<entity>/<entity>.go` — parent command with `var Cmd` + `init()` wiring
4. Create `cmd/<entity>/list.go`, `describe.go`, `delete.go`, etc. — subcommands
5. Register in `cmd/root.go`: add import + `RootCmd.AddCommand(<entity>.Cmd)`
6. Create unit tests in `cmd/<entity>/<entity>_test.go` — verify command structure, flags, and pure logic
7. Create smoke tests in `test/smoke_test/<entity>/<entity>_test.go` — end-to-end CLI tests
8. Create integration tests in `test/integration_test/dataaccess/<entity>_test.go` — direct API tests
9. Run `make build && make unit-test && make gen-doc`

### SDK API Discovery

Browse available SDK operations:
```bash
# V1 APIs (service management)
ls $(go env GOMODCACHE)/github.com/omnistrate-oss/omnistrate-sdk-go@v0.0.90/v1/api_*.go

# Fleet APIs (fleet/operational)
ls $(go env GOMODCACHE)/github.com/omnistrate-oss/omnistrate-sdk-go@v0.0.90/fleet/api_*.go

# Find methods on a specific API
grep "func (a \*<EntityName>ApiAPIService)" \
  $(go env GOMODCACHE)/github.com/omnistrate-oss/omnistrate-sdk-go@v0.0.90/v1/api_<entity>_api.go
```

## Agent Instructions: Adding New Entity Operations

When asked to add CLI support for a new Omnistrate entity, follow this systematic workflow:

### Phase 1: SDK Discovery

1. **Identify the entity name** from the user request (e.g., "notification", "webhook", "report")
2. **Find the matching SDK API file** by searching for the entity name in both V1 and Fleet API directories:
   ```bash
   ls $(go env GOMODCACHE)/github.com/omnistrate-oss/omnistrate-sdk-go@v0.0.90/v1/api_*<entity>*.go
   ls $(go env GOMODCACHE)/github.com/omnistrate-oss/omnistrate-sdk-go@v0.0.90/fleet/api_*<entity>*.go
   ```
3. **Extract available operations** by reading the API file and listing all exported methods:
   ```bash
   grep "func (a \*" <api_file_path>
   ```
4. **Find request/response model types** used by those operations:
   ```bash
   ls $(go env GOMODCACHE)/github.com/omnistrate-oss/omnistrate-sdk-go@v0.0.90/v1/model_*<entity>*.go
   ls $(go env GOMODCACHE)/github.com/omnistrate-oss/omnistrate-sdk-go@v0.0.90/fleet/model_*<entity>*.go
   ```
5. **Read model files** to understand fields available for the display struct

### Phase 2: Implementation

Follow the patterns documented in DEVELOPMENT.md:

1. **Create `internal/model/<entity>.go`** — Define a display struct with string fields and JSON tags. Pick the most useful fields from the SDK response model for table display.

2. **Create `internal/dataaccess/<entity>.go`** — For each SDK operation, create a wrapper function:
   - Determine the correct client: `getV1Client()` or `getFleetClient()`
   - Determine the correct error handler: `handleV1Error()` or `handleFleetError()`
   - Follow the `ctxWithToken → apiClient → Execute() → defer close → handle error` pattern
   - Match the function signature to existing patterns in the codebase (see `internal/dataaccess/secret.go` as a clean example)

3. **Create `cmd/<entity>/` directory** with:
   - `<entity>.go` — Parent command (`var Cmd`) wiring subcommands in `init()`
   - One file per subcommand (list.go, describe.go, delete.go, create.go, etc.)
   - Each subcommand must:
     - Use `RunE` (not `Run`) to return errors
     - Call `defer config.CleanupArgsAndFlags(cmd, &args)` first
     - Call `common.GetTokenWithLogin()` for auth
     - Use spinner for non-JSON output on list/describe operations
     - Convert SDK response to model struct(s) for output
     - Include `Example` and `SilenceUsage: true`

4. **Register in `cmd/root.go`** — Add import and `RootCmd.AddCommand()`

5. **Create unit tests** (`cmd/<entity>/<entity>_test.go`) — REQUIRED for every new command:
   - Test parent command metadata (`Use`, `Short`, `Long`)
   - Test all subcommands are registered via `Cmd.Commands()`
   - Test each subcommand's flags exist with correct type, default value, and shorthand
   - Test pure logic/helper functions with table-driven tests (valid inputs, invalid inputs, edge cases)
   - Use `testify/require` for assertions
   - Follow patterns in `cmd/operations/operations_test.go` (structure tests) and `cmd/instance/common_test.go` (table-driven logic tests)

6. **Create smoke tests** (`test/smoke_test/<entity>/<entity>_test.go`) — REQUIRED for every new command:
   - Guard with `testutils.SmokeTest(t)`, cleanup with `defer testutils.Cleanup()`
   - Login using `testutils.GetTestAccount()` then `cmd.RootCmd.SetArgs(["login", ...])` + `ExecuteContext`
   - Test CRUD lifecycle: set args via `cmd.RootCmd.SetArgs()`, execute via `cmd.RootCmd.ExecuteContext(ctx)`, assert no error
   - Test both default and JSON output formats
   - Follow pattern in `test/smoke_test/secret/secret_test.go`

7. **Create integration tests** (`test/integration_test/dataaccess/<entity>_test.go`) — REQUIRED for every new command:
   - Guard with `testutils.IntegrationTest(t)`
   - Call `dataaccess` functions directly (not via CLI)
   - Use table-driven tests with `wantErr` and `expectedErrMsg` fields
   - Test valid operations, invalid tokens, missing required parameters
   - Follow pattern in `test/integration_test/dataaccess/signin_test.go`

### Phase 3: Validation

1. Run `make build` to verify compilation
2. Run `make unit-test` to verify all unit tests pass (including new tests)
3. Run `make gen-doc` to regenerate CLI documentation
4. Verify the new command appears in help: `./dist/omnistrate-ctl-* <entity> --help`
5. Run a single new test to confirm: `go test ./cmd/<entity>/ -run Test<Entity>Commands -v`

### Reference Examples by Complexity

- **Simple CRUD (args-based)**: `cmd/secret/` + `internal/dataaccess/secret.go` + `internal/model/secret.go`
- **Simple CRUD (flags-based)**: `cmd/service/` + `internal/dataaccess/service.go` + `internal/model/service.go`
- **Complex with many subcommands**: `cmd/instance/` + `internal/dataaccess/resourceinstance.go` + `internal/model/instance.go`
- **Fleet API entity**: `cmd/workflow/` + `internal/dataaccess/workflow.go` + `internal/model/workflow.go`

### Reference Test Examples

- **Command structure unit tests**: `cmd/operations/operations_test.go`, `cmd/workflow/workflow_test.go`
- **Table-driven logic unit tests**: `cmd/instance/common_test.go` (parseCustomTags, formatTags, matchesTagFilters)
- **Smoke tests (CLI end-to-end)**: `test/smoke_test/secret/secret_test.go`
- **Integration tests (direct API)**: `test/integration_test/dataaccess/signin_test.go`
- **Test utilities**: `test/testutils/testutils.go` (guards, auth, cleanup)
