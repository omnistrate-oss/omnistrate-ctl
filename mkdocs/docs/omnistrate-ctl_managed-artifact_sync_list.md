## omnistrate-ctl managed-artifact sync list

List managed artifact synchronization records

```
omnistrate-ctl managed-artifact sync list [flags]
```

### Examples

```
omnistrate-ctl managed-artifact sync list --status failed
omnistrate-ctl managed-artifact sync list --target-id hc-123 -o json
```

### Options

```
      --bundle-version string    Filter by bundle version, for example r0000020
  -h, --help                     help for list
      --limit int                Maximum number of syncs to return (1-100) (default 20)
      --next-page-token string   Opaque token returned by the previous page
      --status string            Filter by status: pending, in_progress, ready, failed, or skipped
      --target-id string         Filter by provisioner target ID
      --updated-after string     Filter syncs updated at or after this RFC3339 timestamp
      --updated-before string    Filter syncs updated before this RFC3339 timestamp
```

### Options inherited from parent commands

```
  -o, --output string   Output format (text|table|json) (default "table")
  -v, --version         Print the version number of omnistrate-ctl
```

### SEE ALSO

* [omnistrate-ctl managed-artifact sync](omnistrate-ctl_managed-artifact_sync.md)	 - Inspect provisioner managed artifact synchronization

