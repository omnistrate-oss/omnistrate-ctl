## omnistrate-ctl managed-artifact sync verify

Ask the control plane to verify a MANUAL managed artifact sync

### Synopsis

Ask the control plane to verify the destination artifacts of a private MANUAL sync now instead of waiting for periodic reconciliation. Per-artifact progress is persisted. A complete verification transitions the sync to READY. An incomplete verification remains MANUAL unless the applied account setting explicitly enables automatic publication, in which case the sync transitions to PENDING and normal automatic reconciliation resumes.

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

