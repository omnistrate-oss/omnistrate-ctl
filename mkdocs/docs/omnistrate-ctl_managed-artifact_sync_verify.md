## omnistrate-ctl managed-artifact sync verify

Ask the control plane to verify a MANUAL managed artifact sync

### Synopsis

Ask the control plane to verify the destination artifacts of a private MANUAL sync now instead of waiting for periodic reconciliation. Per-artifact progress is persisted; the sync stays MANUAL until every expected artifact is present, then transitions to READY.

```
omnistrate-ctl managed-artifact sync verify [flags]
```

### Examples

```
omnistrate-ctl managed-artifact sync verify --id spabs-123
```

### Options

```
  -h, --help        help for verify
      --id string   Private MANUAL sync ID (spabs-*) (required)
```

### Options inherited from parent commands

```
  -o, --output string   Output format (text|table|json) (default "table")
  -v, --version         Print the version number of omnistrate-ctl
```

### SEE ALSO

* [omnistrate-ctl managed-artifact sync](omnistrate-ctl_managed-artifact_sync.md)	 - Inspect private artifact syncs and public ECR publications

