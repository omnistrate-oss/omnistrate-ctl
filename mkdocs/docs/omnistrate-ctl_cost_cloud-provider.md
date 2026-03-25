## omnistrate-ctl cost cloud-provider

Get cost breakdown by cloud provider

### Synopsis

Get the total cost of operating your fleet across different cloud providers.

```
omnistrate-ctl cost cloud-provider [flags]
```

### Options

```
      --end-date string            End date for cost analysis (RFC3339 format) (required)
  -e, --environment-type string    Environment type (valid: dev, qa, staging, canary, prod, private) (required)
      --exclude-providers string   Cloud provider IDs to exclude (comma-separated)
  -f, --frequency string           Frequency of cost data (daily, weekly, monthly) (default "daily")
  -h, --help                       help for cloud-provider
      --include-providers string   Cloud provider IDs to include (comma-separated)
      --start-date string          Start date for cost analysis (RFC3339 format) (required)
```

### Options inherited from parent commands

```
  -o, --output string   Output format (text|table|json) (default "table")
  -v, --version         Print the version number of omnistrate-ctl
```

### SEE ALSO

* [omnistrate-ctl cost](omnistrate-ctl_cost.md)	 - Manage cost analytics for your services

