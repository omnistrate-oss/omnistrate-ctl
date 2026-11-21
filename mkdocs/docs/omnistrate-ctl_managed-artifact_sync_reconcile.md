## omnistrate-ctl managed-artifact sync reconcile

Run one platform push for a private managed artifact sync

### Synopsis

Run one platform-managed copy and verification for a private MANUAL, FAILED, SKIPPED, or READY sync and its exact bundle version. Automatic private publication must not be explicitly disabled. MANUAL, FAILED, and SKIPPED enter IN_PROGRESS after dispatch, become READY on success, and restore their previous status on failure. READY remains READY throughout and is unchanged if the republish fails. IN_PROGRESS requests and READY requests with an active republish are rejected. This command never retries other syncs or versions in the background.

```
omnistrate-ctl managed-artifact sync reconcile [flags]
```

### Examples

```
omnistrate-ctl managed-artifact sync reconcile --id spabs-123
```

### Options

```
  -h, --help        help for reconcile
      --id string   Private MANUAL, FAILED, SKIPPED, or READY sync ID (spabs-*) (required)
```

### Options inherited from parent commands

```
  -o, --output string   Output format (text|table|json) (default "table")
  -v, --version         Print the version number of omnistrate-ctl
```

### SEE ALSO

* [omnistrate-ctl managed-artifact sync](omnistrate-ctl_managed-artifact_sync.md)	 - Inspect private artifact syncs and public ECR publications

