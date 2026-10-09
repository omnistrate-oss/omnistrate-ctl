## omnistrate-ctl managed-artifact sync list

List managed artifact synchronization records

### Synopsis

List private ECR synchronization records or public ECR publications.

Omitting --registry-type preserves private ECR behavior. --destination-account-id
filters either registry type by its destination AWS account. When --target-id is
also supplied, both filters must match. Public publications may have no execution
target yet. These filters require the public ECR managed-artifact API update.
Returned records must match the requested registry and destination account; an
empty result alone cannot establish backend support. When paging, keep all
filters unchanged, including --registry-type and --destination-account-id.

```
omnistrate-ctl managed-artifact sync list [flags]
```

### Examples

```
omnistrate-ctl managed-artifact sync list --status failed
omnistrate-ctl managed-artifact sync list --registry-type public_ecr --destination-account-id 123456789012 -o json
omnistrate-ctl managed-artifact sync list --target-id hc-123 -o json
```

### Options

```
      --bundle-version string           Filter by bundle version, for example r0000020
      --destination-account-id string   Filter either ECR registry type by 12-digit destination AWS account ID
  -h, --help                            help for list
      --limit int                       Maximum number of syncs to return (1-100) (default 20)
      --next-page-token string          Opaque token returned by the previous page
      --registry-type string            Registry type: private_ecr or public_ecr (omitted defaults to private ECR)
      --status string                   Filter by status: pending, in_progress, ready, failed, or skipped
      --target-id string                Filter by execution provisioner ID; must also match any destination account filter
      --updated-after string            Filter syncs updated at or after this RFC3339 timestamp
      --updated-before string           Filter syncs updated before this RFC3339 timestamp
```

### Options inherited from parent commands

```
  -o, --output string   Output format (text|table|json) (default "table")
  -v, --version         Print the version number of omnistrate-ctl
```

### SEE ALSO

* [omnistrate-ctl managed-artifact sync](omnistrate-ctl_managed-artifact_sync.md)	 - Inspect private artifact syncs and public ECR publications

