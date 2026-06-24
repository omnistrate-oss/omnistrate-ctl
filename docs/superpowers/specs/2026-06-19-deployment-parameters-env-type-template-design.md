# Design: environment filter + create-params template for `instance deployment-parameters`

Date: 2026-06-19
Branch: `fix-dep-params-env`

## Summary

Extend the `omnistrate-ctl instance deployment-parameters` command in three ways:

1. Add a **required** `--environment` flag that disambiguates which service
   environment a resource lookup resolves to. The flag name and matching
   semantics mirror `instance create --environment` (matches on environment
   name or ID, case-insensitive).
2. Keep the existing `--output=json` machine-readable output (already present; no
   functional change, documented here for completeness).
3. Add a `deployment-parameters template` subcommand that emits a JSON object
   matching the shape consumed by `instance create --param-file`, pre-populated
   with defaults where available and typed placeholders otherwise.

## Motivation

- A resource name can exist across multiple service environments (e.g. `dev` and
  `prod`). The current `getServiceAndPlanInfo` matches only on service + plan +
  resource name, so multi-environment services can resolve ambiguously. The
  `ResourceSearchRecord` already carries `ServiceEnvironmentName` and
  `ServiceEnvironmentId`, so we can filter on the environment without any new
  API calls.
- Using `--environment` (rather than `--environment-type`) keeps this flag
  consistent with the sibling `instance create` command, which already uses
  `--environment` to identify the target environment by name/ID.
- Users assembling `instance create` invocations must hand-author the `--param`
  / `--param-file` JSON. A generated template removes guesswork about which keys
  exist, which are required, and what the defaults are.

## Scope and non-goals

- In scope: the `deployment-parameters` command and a new `template` subcommand
  under it.
- Not in scope: changing `instance create`, the `--output=json` format of the
  parent command, or the `ListInputParameters` API.

## Behavior

### `--environment` (required)

- Added to the existing `deploymentParametersCmd` and to the new `template`
  subcommand.
- Accepts an environment name or ID, matched case-insensitively — identical
  semantics to `instance create --environment`. There is no fixed valid-value
  set; any environment the user owns is acceptable.
- Threaded into `getServiceAndPlanInfo`, which adds an environment match
  (`matchesIDOrName(ServiceEnvironmentId, ServiceEnvironmentName, requested)`,
  reusing the existing helper) as a criterion alongside service/plan/resource
  name.
- If no resource matches the given environment, the existing "resource not
  found" error is returned, naming the environment.

**Breaking change (accepted):** invocations of `deployment-parameters` that
omitted an environment now fail until `--environment` is supplied. This was
explicitly chosen over an optional/defaulted flag.

### JSON output (unchanged)

The parent command keeps `--output=table|json`. No change to the existing
`DeploymentParametersOutput` JSON schema.

### `deployment-parameters template` subcommand

```
omnistrate-ctl instance deployment-parameters template \
  --service=mysql --plan=mysql --version=latest --resource=mySQL --environment=dev
```

- Flags: `--service`, `--plan`, `--resource`, `--environment` (all
  required), `--version` (optional, default `preferred`).
- Output: a single JSON object printed to stdout — exactly the shape
  `instance create --param-file` consumes (a flat `key -> value` map, no
  surrounding metadata), so it can be redirected straight to a file:

  ```
  omnistrate-ctl instance deployment-parameters template ... > params.json
  omnistrate-ctl instance create ... --param-file params.json
  ```

- Includes **all** input parameters returned for the resource (these are the
  create-time parameters; `ListInputParameters` already returns create scope).
- Value for each key:
  - **default present** → the default value, coerced to the parameter's type.
  - **no default** → a typed placeholder:
    - `string` / `password` / `secret` → `""`
    - `bool` / `boolean` → `false`
    - `int` / `int32` / `int64` / `integer` → `0`
    - `float` / `float32` / `float64` / `double` / `number` → `0`
    - object / `json` / `map` → `{}`
    - list parameters (`IsList`) → `[]`
- Keys are emitted in sorted order for stable output.

## Implementation plan

All changes in `cmd/instance/deployment_parameters.go` plus registration in
`cmd/instance/instance.go`.

1. **Add `--environment` flag** to `deploymentParametersCmd`, mark required.
   No value-set validation is needed (environment names are arbitrary).

2. **Update `getServiceAndPlanInfo`** to accept an `environment` argument and
   include `matchesIDOrName(res.ServiceEnvironmentId, res.ServiceEnvironmentName, environment)`
   in the match (reusing the existing helper from `create.go`).

3. **Extract a shared fetch helper.** Factor the resource-resolution +
   `ListInputParameters` + `[]ParameterInfo` assembly out of
   `runDeploymentParameters` into a helper (e.g.
   `fetchDeploymentParameters(ctx, token, service, plan, version, resource, environment)`)
   returning the parameters and resolved metadata, so both the parent command and
   the `template` subcommand share one code path.

4. **Add the `template` subcommand** (`templateCmd`), registered via
   `deploymentParametersCmd.AddCommand(templateCmd)`. It calls the shared helper,
   builds the `map[string]any` template, and prints indented JSON.

5. **Template value typing.** Reuse the existing type-coercion logic (mirroring
   `parseServicePlanDeploymentParamValue` in `cmd/serviceplan/browser.go`) for
   default values; produce typed placeholders for parameters without defaults.
   If the coercion logic is duplicated rather than imported, keep it minimal and
   local to this file.

## Error handling

- Missing `--environment` → cobra required-flag error.
- No resource matching the environment → "resource not found" error naming
  the service, plan, resource, and environment.
- Default value that fails type coercion → fall back to emitting the raw default
  string (do not fail template generation over a malformed default).

## Testing

Unit tests (table-driven where practical):

- Environment filtering: a resource present in multiple environments
  resolves to exactly the requested one; mismatched environment yields
  not-found.
- Template value typing: default present (coerced per type), no default
  (placeholder per type), list parameter (`[]`), bool (`false`), number (`0`).
- Template output is valid JSON and round-trips through `common.FormatParams`.
