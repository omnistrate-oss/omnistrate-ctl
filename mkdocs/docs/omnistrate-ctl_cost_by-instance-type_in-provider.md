## omnistrate-ctl cost by-instance-type in-provider

List instance type costs in a specific cloud provider

### Synopsis

Get cost breakdown by instance type for a specific cloud provider.

```
omnistrate-ctl cost by-instance-type in-provider <provider-id> [flags]
```

### Options

```
      --end-date string      End date for cost analysis (RFC3339 format, e.g., 2024-01-31T23:59:59Z)
  -e, --environment string   Environment type (valid: dev, qa, staging, canary, prod, private)
  -f, --frequency string     Frequency of cost data (daily, weekly, monthly) (default "daily")
  -h, --help                 help for in-provider
      --start-date string    Start date for cost analysis (RFC3339 format, e.g., 2024-01-01T00:00:00Z)
```

### Options inherited from parent commands

```
  -o, --output string   Output format (text|table|json) (default "table")
  -v, --version         Print the version number of omnistrate-ctl
```

### SEE ALSO

* [omnistrate-ctl cost by-instance-type](omnistrate-ctl_cost_by-instance-type.md)	 - Get cost breakdown by instance type

