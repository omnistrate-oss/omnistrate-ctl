---
applyTo: "test/**/*.go"
---

# Test Files Instructions

Files under `test/` are smoke tests and integration tests that run against live APIs. They are always guarded by environment variables.

## Smoke Tests (`test/smoke_test/`)

Smoke tests exercise the CLI end-to-end by setting args and executing commands.

### Required Pattern

```go
func Test_entity_basic(t *testing.T) {
    testutils.SmokeTest(t)              // REQUIRED: skip guard
    ctx := context.TODO()
    require := require.New(t)
    defer testutils.Cleanup()           // REQUIRED: cleanup config

    // REQUIRED: login first
    testEmail, testPassword, err := testutils.GetTestAccount()
    require.NoError(err)
    cmd.RootCmd.SetArgs([]string{"login", fmt.Sprintf("--email=%s", testEmail), fmt.Sprintf("--password=%s", testPassword)})
    err = cmd.RootCmd.ExecuteContext(ctx)
    require.NoError(err)

    // Test operations via CLI
    cmd.RootCmd.SetArgs([]string{"entity", "list"})
    err = cmd.RootCmd.ExecuteContext(ctx)
    require.NoError(err)
}
```

### Rules

- Always start with `testutils.SmokeTest(t)` — skips unless `ENABLE_SMOKE_TEST=true`
- Always `defer testutils.Cleanup()`
- Always login before testing operations
- Set CLI args via `cmd.RootCmd.SetArgs([]string{...})`
- Execute via `cmd.RootCmd.ExecuteContext(ctx)`
- Test CRUD lifecycle when applicable (create → describe → list → delete)
- Test both default and JSON output formats
- For async operations, use `testutils.WaitForInstanceToReachStatus()`

## Integration Tests (`test/integration_test/`)

Integration tests call internal Go functions directly.

### Required Pattern

```go
func TestEntityOperation(t *testing.T) {
    testutils.IntegrationTest(t)        // REQUIRED: skip guard

    testEmail, testPassword, err := testutils.GetTestAccount()
    require.NoError(t, err)

    ctx := context.TODO()
    token, err := dataaccess.LoginWithPassword(ctx, testEmail, testPassword)
    require.NoError(t, err)

    tests := []struct {
        name           string
        input          string
        wantErr        bool
        expectedErrMsg string
    }{
        {"valid case", "valid-id", false, ""},
        {"invalid case", "", true, "invalid_format"},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result, err := dataaccess.DescribeEntity(ctx, token, tt.input)
            if tt.wantErr {
                assert.Error(t, err)
                assert.Contains(t, err.Error(), tt.expectedErrMsg)
            } else {
                require.NoError(t, err)
                assert.NotNil(t, result)
            }
        })
    }
}
```

### Rules

- Always start with `testutils.IntegrationTest(t)` — skips unless `ENABLE_INTEGRATION_TEST=true`
- Call dataaccess functions directly (not via CLI)
- Use table-driven tests with `wantErr` and `expectedErrMsg` fields
- Use `require` for setup assertions (login), `assert` for test case assertions
- Test valid operations, invalid tokens, and missing required parameters

## Environment Variables

```bash
ENABLE_SMOKE_TEST=true        # Enable smoke tests
ENABLE_INTEGRATION_TEST=true  # Enable integration tests
TEST_EMAIL=user@example.com   # Test account email
TEST_PASSWORD=secret           # Test account password
```
