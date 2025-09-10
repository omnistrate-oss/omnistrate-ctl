## omnistrate-ctl upgrade list

List upgrade paths

### Synopsis

List upgrade paths for a service and product tier with filtering options.

```
omnistrate-ctl upgrade list [flags]
```

### Options

```
  -h, --help                     help for list
      --next-page-token string   Token for next page of results
      --page-size int            Number of results per page
  -p, --product-tier-id string   Product tier ID (required)
  -s, --service-id string        Service ID (required)
      --source-version string    Source product tier version to filter by
      --status string            Status of upgrade path to filter by
      --target-version string    Target product tier version to filter by
      --type string              Type of upgrade path to filter by
```

### Options inherited from parent commands

```
  -o, --output string   Output format (text|table|json) (default "table")
  -v, --version         Print the version number of omnistrate-ctl
```

### SEE ALSO

- [omnistrate-ctl upgrade](omnistrate-ctl_upgrade.md) - Upgrade Instance Deployments to a newer or older version
