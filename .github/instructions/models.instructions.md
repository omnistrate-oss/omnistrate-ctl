---
applyTo: "internal/model/**/*.go"
---

# Model Layer Instructions

When creating or modifying files in `internal/model/`, follow these patterns.

## Purpose

Model structs are display-only types used to format CLI output. They convert SDK response types into a simple flat structure for table/JSON rendering.

## Struct Pattern

```go
package model

type Widget struct {
    ID          string `json:"id"`
    Name        string `json:"name"`
    Status      string `json:"status"`
    Description string `json:"description"`
}
```

## Rules

1. **All fields must be strings** — even numeric or boolean values should be converted to strings for consistent table rendering
2. **Use JSON tags** on every field — these control both JSON output and table column headers
3. **Use `json:"field_name"` format** — snake_case for JSON tag names
4. **Use `json:"field,omitempty"`** for optional fields that may be empty
5. **Keep structs flat** — no nested structs; flatten complex SDK responses into simple fields
6. **Pick useful fields** — not all SDK response fields need to be in the model; select what's most useful for CLI display
7. **Add list wrapper structs** when needed for proper JSON output:
   ```go
   type WidgetList struct {
       Widgets []Widget `json:"widgets"`
   }
   ```

## Usage in Commands

Models are constructed in `cmd/` layer by reading SDK response getters:

```go
widget := model.Widget{
    ID:     result.GetId(),
    Name:   result.GetName(),
    Status: result.GetStatus(),
}
```

## Output Functions

- `utils.PrintTextTableJsonOutput(output, singleObject)` — for single items
- `utils.PrintTextTableJsonArrayOutput(output, sliceOfObjects)` — for lists

## Existing References

- `internal/model/service.go` — simple 3-field struct
- `internal/model/secret.go` — struct with omitempty + list wrapper
- `internal/model/instance.go` — larger struct with many fields
