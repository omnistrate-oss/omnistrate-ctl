---
applyTo: "internal/dataaccess/**/*.go"
---

# Data Access Layer Instructions

When creating or modifying files in `internal/dataaccess/`, follow these patterns strictly.

## SDK Clients

Two clients are available via helper functions in `client.go`:

- **`getV1Client()`** — Returns `*openapiclientv1.APIClient` for service management APIs
  - Import: `openapiclientv1 "github.com/omnistrate-oss/omnistrate-sdk-go/v1"`
  - Error handler: `handleV1Error(err)`
  - APIs: ServiceApiAPI, SecretsApiAPI, ServicePlanApiAPI, SubscriptionApiAPI, CustomDomainApiAPI, etc.

- **`getFleetClient()`** — Returns `*openapiclientfleet.APIClient` for fleet/operational APIs
  - Import: `openapiclientfleet "github.com/omnistrate-oss/omnistrate-sdk-go/fleet"`
  - Error handler: `handleFleetError(err)`
  - APIs: InventoryApiAPI, FleetWorkflowsApiAPI, CostApiAPI, OperationsApiAPI, etc.

## Function Pattern (Read Operations)

```go
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
```

## Function Pattern (Write Operations — No Response Body)

```go
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

## Function Pattern (Write Operations — With Request Body)

```go
func CreateWidget(ctx context.Context, token string, request openapiclientv1.CreateWidgetRequest) (*openapiclientv1.CreateWidgetResult, error) {
    ctxWithToken := context.WithValue(ctx, openapiclientv1.ContextAccessToken, token)
    apiClient := getV1Client()

    resp, r, err := apiClient.WidgetApiAPI.WidgetApiCreateWidget(ctxWithToken).
        CreateWidgetRequest(request).
        Execute()
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
```

## Critical Rules

1. **Always** create `ctxWithToken` using `context.WithValue` with the SDK access token constant
2. **Always** `defer` closing `r.Body` with a nil check on `r`
3. **Always** use the matching error handler: `handleV1Error()` for V1, `handleFleetError()` for Fleet
4. **Never** return raw SDK errors — always wrap through the error handler
5. Functions returning data use `(*SdkType, error)` signature
6. Functions mutating without response use `error` signature

## Finding SDK Operations

```bash
# List available API methods for an entity
grep "func (a \*<Entity>ApiAPIService)" \
  $(go env GOMODCACHE)/github.com/omnistrate-oss/omnistrate-sdk-go@v0.0.90/v1/api_<entity>_api.go

# Find model types
ls $(go env GOMODCACHE)/github.com/omnistrate-oss/omnistrate-sdk-go@v0.0.90/v1/model_*<entity>*.go
```

## Testing (REQUIRED)

Every new dataaccess file MUST have corresponding tests:
- Integration tests in `test/integration_test/dataaccess/<entity>_test.go`
- Guard with `testutils.IntegrationTest(t)`
- Use table-driven tests with `wantErr` pattern
- Test valid operations and error cases (invalid tokens, missing params)
- Reference: `test/integration_test/dataaccess/signin_test.go`
