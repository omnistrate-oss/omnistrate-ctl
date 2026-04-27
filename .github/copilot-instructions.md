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


<!-- security-checklist-managed -->

## Security Checklist

Apply this checklist to every code change. If a control is not applicable, briefly say why in the PR description.

### Authentication & Token Handling
- Store auth tokens in the OS keychain when available (macOS Keychain, Windows Credential Manager, Secret Service / libsecret). Fall back to a file with `0600` permissions in the user's home directory only when no keychain is available, and warn the user.
- Never write tokens, refresh tokens, or session cookies to stdout/stderr or log files. Redact `Authorization` headers from any debug/verbose output.
- Honor token TTLs: refresh proactively, clear on `logout`, and never persist a token after `--logout` / `--clear-credentials`.
- When prompting for credentials, read from a TTY without echo (`term.ReadPassword` / equivalent). Do not accept passwords as command-line flags.

### Authorization
- The CLI must not assume the local user has any role — let the server return 401/403 and surface a clear error. Do not cache role decisions across sessions.
- Destructive commands (`delete`, `rotate`, `purge`) must require an explicit confirmation flag (`--yes`) or interactive prompt. No silent destructive defaults.

### Input Validation
- For commands intended to operate within a base directory, canonicalize paths, resolve symlinks, and enforce that the result stays under the allowed base. (Blanket rejection of `..` is not appropriate for a general-purpose CLI.)
- Validate URLs (scheme allowlist: `https`, optionally `http` only for `localhost`). Reject `file://`, `javascript:`, etc.
- Use safe YAML/JSON loaders. For YAML, disallow custom tags / arbitrary type construction. Cap input file sizes.
- When templating user input into config, escape per the target format (shell, YAML, JSON, URL) — never `fmt.Sprintf` user input into a shell command.

### Subprocess Execution
- Use `exec.Command(name, args...)` with explicit arguments. Never invoke `sh -c` / `cmd /c` with interpolated user input.
- Pass secrets to subprocesses via env vars or stdin, not via command-line arguments (which appear in `ps`).
- Validate or hash-pin any binary the CLI downloads and executes (e.g., `kubectl`, `helm`, `terraform`).

### Secrets Handling
- No secrets in source, fixtures, or test recordings. Scrub HTTP recordings (`go-vcr`, etc.) before committing.
- Files written by the CLI that contain credentials, kubeconfigs, or signed URLs must be `0600` and in a per-user directory.
- Do not include secrets in telemetry, crash reports, or `--debug` output.

### Logging & Output Hygiene
- `--verbose` / `--debug` must not enable secret logging. Maintain a redaction layer that runs even at the highest verbosity.
- Errors shown to the user should be actionable and free of internal stack traces, raw HTTP responses, or upstream tokens.
- Progress output to TTY only; structured logs (JSON) for non-TTY / CI use.

### Dependencies & Supply Chain
- Run `govulncheck ./...` (or `npm audit` for TS) on changes touching dependencies. Address findings.
- Pin module versions. Justify any new third-party dependency in the PR description.
- Releases must publish checksums and, where possible, signatures (cosign, Sigstore). Homebrew / installer scripts must verify checksums before installing.
- Pin GitHub Actions to commit SHAs.

### Update & Distribution Safety
- Self-update / install scripts must verify the integrity (checksum + signature) of the downloaded artifact before replacing the binary.
- The default installer should use HTTPS only, with certificate validation enabled.

### What to do when unsure
- If a change touches credential storage, subprocess execution, or download/update flows, call it out explicitly in the PR description and request a security review.
