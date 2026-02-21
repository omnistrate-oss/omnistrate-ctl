---
applyTo: "cmd/**/*.go"
---

# Command Layer Instructions

When creating or modifying files in `cmd/`, follow these patterns strictly.

## Parent Command File (`cmd/<entity>/<entity>.go`)

- Export `var Cmd = &cobra.Command{...}` with `Use`, `Short`, `Long`, `Run`, `SilenceUsage: true`
- Wire all subcommands in `init()` via `Cmd.AddCommand(subCmd)`
- The `Run` function should just call `cmd.Help()`

## Subcommand Files (`cmd/<entity>/list.go`, etc.)

- Use unexported `var listCmd = &cobra.Command{...}`
- Always use `RunE` (not `Run`) to return errors
- Always set `SilenceUsage: true`
- Always include `Example` with realistic usage examples
- Define flags and args constraints in `init()`

## Run Function Pattern (MUST follow this order)

```go
func runXxx(cmd *cobra.Command, args []string) error {
    defer config.CleanupArgsAndFlags(cmd, &args)  // 1. Always first

    output, _ := cmd.Flags().GetString("output")   // 2. Parse flags

    token, err := common.GetTokenWithLogin()        // 3. Auth
    if err != nil {
        utils.PrintError(err)
        return err
    }

    // 4. Optional: spinner for non-JSON output
    var sm ysmrr.SpinnerManager
    var spinner *ysmrr.Spinner
    if output != "json" {
        sm = ysmrr.NewSpinnerManager()
        spinner = sm.AddSpinner("Loading...")
        sm.Start()
    }

    // 5. Call dataaccess function
    result, err := dataaccess.ListXxx(cmd.Context(), token)
    if err != nil {
        utils.HandleSpinnerError(spinner, sm, err)
        return err
    }

    // 6. Convert SDK response to model structs
    items := make([]model.Xxx, 0)
    for _, item := range result.GetItems() {
        items = append(items, model.Xxx{...})
    }

    utils.HandleSpinnerSuccess(spinner, sm, "Done")

    // 7. Print output
    return utils.PrintTextTableJsonArrayOutput(output, items)
}
```

## Flags

- Use `cmd.Flags().String("name", "default", "description")` for optional flags
- Use `cmd.MarkFlagRequired("name")` for required flags
- Use `cmd.Flags().StringP("name", "n", "", "description")` for flags with shorthand
- Use `cmd.Args = cobra.NoArgs` or `cobra.ExactArgs(N)` for positional args

## Error Handling

- Always call `utils.PrintError(err)` before returning errors
- Always `return err` after printing so cobra captures exit code

## Registration

After creating a new command package, register it in `cmd/root.go`:
1. Add import: `"github.com/omnistrate-oss/omnistrate-ctl/cmd/<entity>"`
2. Add in `init()`: `RootCmd.AddCommand(<entity>.Cmd)`

## Testing (REQUIRED)

Every new command MUST have a unit test file `cmd/<entity>/<entity>_test.go` that:
- Verifies parent command metadata (Use, Short, Long)
- Verifies all subcommands are registered
- Verifies each subcommand's flags exist with correct type and defaults
- Tests any pure logic helper functions with table-driven tests
- Uses `testify/require` for assertions

Reference: `cmd/operations/operations_test.go`, `cmd/workflow/workflow_test.go`

## Documentation

After ANY changes, run `make gen-doc` to regenerate CLI docs.
