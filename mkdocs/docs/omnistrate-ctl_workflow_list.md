## omnistrate-ctl workflow list

List workflows for a service environment

### Synopsis

List all workflows for a specific service and environment.

```
omnistrate-ctl workflow list [flags]
```

### Options

```
      --end-date string          Filter workflows created before this date (RFC3339 format)
  -e, --environment-id string    Environment ID (required)
  -h, --help                     help for list
  -i, --instance-id string       Filter by instance ID (optional)
      --next-page-token string   Token for next page of results
      --page-size int            Number of results per page
  -s, --service-id string        Service ID (required)
      --start-date string        Filter workflows created after this date (RFC3339 format)
```

### Options inherited from parent commands

```
  -o, --output string   Output format (text|table|json) (default "table")
  -v, --version         Print the version number of omnistrate-ctl
```

### SEE ALSO

- [omnistrate-ctl workflow](omnistrate-ctl_workflow.md) - Manage service workflows
