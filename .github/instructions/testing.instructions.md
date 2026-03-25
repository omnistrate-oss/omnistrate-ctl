---
applyTo: "**/*_test.go"
---

# Testing Instructions

All new functionality MUST include tests. Follow these patterns based on test category.

## Unit Tests (`cmd/<entity>/*_test.go`, `internal/**/*_test.go`)

Unit tests run without external dependencies. They verify structure and logic.

### Command Structure Tests

Test that commands, subcommands, and flags are correctly defined:

```go
package widget

import (
    "testing"
    "github.com/stretchr/testify/require"
)

func TestWidgetCommands(t *testing.T) {
    require := require.New(t)

    require.Equal("widget [operation] [flags]", Cmd.Use)
    require.Equal("Manage widgets", Cmd.Short)

    expectedCommands := []string{"list", "describe", "delete"}
    actualCommands := make([]string, 0)
    for _, cmd := range Cmd.Commands() {
        actualCommands = append(actualCommands, cmd.Name())
    }
    for _, expected := range expectedCommands {
        require.Contains(actualCommands, expected)
    }
}

func TestListCommandFlags(t *testing.T) {
    require := require.New(t)

    flag := listCmd.Flags().Lookup("filter")
    require.NotNil(flag)
    require.Equal("stringArray", flag.Value.Type())
}
```

### Table-Driven Logic Tests

Test pure functions with comprehensive input/output cases:

```go
func TestFormatStatus(t *testing.T) {
    tests := []struct {
        name          string
        input         string
        expected      string
        expectError   bool
        errorContains string
    }{
        {"valid", "ACTIVE", "active", false, ""},
        {"empty", "", "", true, "cannot be empty"},
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result, err := formatStatus(tt.input)
            if tt.expectError {
                require.Error(t, err)
                require.Contains(t, err.Error(), tt.errorContains)
                return
            }
            require.NoError(t, err)
            require.Equal(t, tt.expected, result)
        })
    }
}
```

## Smoke Tests (`test/smoke_test/<entity>/*_test.go`)

Smoke tests exercise the CLI end-to-end against a live API.

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
    testutils.SmokeTest(t)             // Skips unless ENABLE_SMOKE_TEST=true

    ctx := context.TODO()
    require := require.New(t)
    defer testutils.Cleanup()          // Remove config dir after test

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

    // Test list operation
    cmd.RootCmd.SetArgs([]string{"widget", "list"})
    err = cmd.RootCmd.ExecuteContext(ctx)
    require.NoError(err)

    // Test with JSON output
    cmd.RootCmd.SetArgs([]string{"widget", "list", "--output", "json"})
    err = cmd.RootCmd.ExecuteContext(ctx)
    require.NoError(err)
}
```

## Integration Tests (`test/integration_test/dataaccess/*_test.go`)

Integration tests call dataaccess functions directly against a live API.

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
    testutils.IntegrationTest(t)       // Skips unless ENABLE_INTEGRATION_TEST=true

    testEmail, testPassword, err := testutils.GetTestAccount()
    require.NoError(t, err)

    ctx := context.TODO()
    token, err := dataaccess.LoginWithPassword(ctx, testEmail, testPassword)
    require.NoError(t, err)

    // Valid token
    result, err := dataaccess.ListWidgets(ctx, token)
    assert.NoError(t, err)
    assert.NotNil(t, result)

    // Invalid token
    _, err = dataaccess.ListWidgets(ctx, "invalid")
    assert.Error(t, err)
}
```

## Conventions

- Use `testify/require` for fatal assertions, `testify/assert` for non-fatal
- Place unit test files in the same package as the code under test
- Name test files `*_test.go`
- Use table-driven tests for functions with multiple scenarios
- Always test both valid and invalid inputs, including edge cases
- Smoke/integration tests are guarded by env vars — they skip by default

## Test Utilities (`test/testutils/`)

| Function | Description |
|----------|-------------|
| `SmokeTest(t)` | Skip unless `ENABLE_SMOKE_TEST=true` |
| `IntegrationTest(t)` | Skip unless `ENABLE_INTEGRATION_TEST=true` |
| `GetTestAccount()` | Returns `(email, password, error)` from env vars |
| `Cleanup()` | Removes config directory |
| `WaitForInstanceToReachStatus(ctx, id, status)` | Polls with exponential backoff |

## Running Tests

```bash
make unit-test                                    # All unit tests
go test ./cmd/widget/ -run TestWidgetCommands -v  # Single test
make smoke-test                                   # Requires ENABLE_SMOKE_TEST + credentials
make integration-test                             # Requires ENABLE_INTEGRATION_TEST + credentials
```
