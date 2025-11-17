## omnistrate-ctl cost by-provider list

List costs for all cloud providers

### Synopsis

Get cost breakdown for all cloud providers in your account.

```
omnistrate-ctl cost by-provider list [flags]
```

### Options

```
      --end-date string      End date for cost analysis (RFC3339 format, e.g., 2024-01-31T23:59:59Z)
  -e, --environment string   Environment type (valid: dev, qa, staging, canary, prod, private)
  -f, --frequency string     Frequency of cost data (daily, weekly, monthly) (default "daily")
  -h, --help                 help for list
      --start-date string    Start date for cost analysis (RFC3339 format, e.g., 2024-01-01T00:00:00Z)
```

### Options inherited from parent commands

```
  -o, --output string   Output format (text|table|json) (default "table")
  -v, --version         Print the version number of omnistrate-ctl
```

### SEE ALSO

* [omnistrate-ctl cost by-provider](omnistrate-ctl_cost_by-provider.md)	 - Get cost breakdown by cloud provider

