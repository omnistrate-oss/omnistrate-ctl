## omnistrate-ctl managed-artifact sync refresh-status

Verify a customer-populated MANUAL sync destination

### Synopsis

Refresh the status of one private MANUAL sync after the customer copies its artifacts. The control plane only verifies the existing destination and never starts a platform copy. A complete destination transitions to READY; an incomplete destination remains MANUAL with per-artifact results.

```
omnistrate-ctl managed-artifact sync refresh-status [flags]
```

### Examples

```
omnistrate-ctl managed-artifact sync refresh-status --id spabs-123
```

### Options

```
  -h, --help        help for refresh-status
      --id string   Private MANUAL sync ID (spabs-*) (required)
```

### Options inherited from parent commands

```
  -o, --output string   Output format (text|table|json) (default "table")
  -v, --version         Print the version number of omnistrate-ctl
```

### SEE ALSO

* [omnistrate-ctl managed-artifact sync](omnistrate-ctl_managed-artifact_sync.md)	 - Inspect private artifact syncs and public ECR publications

