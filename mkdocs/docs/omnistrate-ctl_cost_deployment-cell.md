## omnistrate-ctl cost deployment-cell

Get cost breakdown by deployment cell

### Synopsis

Get the total cost of operating your fleet across different deployment cells.

```
omnistrate-ctl cost deployment-cell [flags]
```

### Options

```
      --end-date string            End date for cost analysis (RFC3339 format) (required)
  -e, --environment-type string    Environment type (valid: dev, qa, staging, canary, prod, private) (required)
      --exclude-cells string       Deployment cell IDs to exclude (comma-separated)
      --exclude-instances string   Instance IDs to exclude (comma-separated)
      --exclude-providers string   Cloud provider IDs to exclude (comma-separated)
  -f, --frequency string           Frequency of cost data (daily, weekly, monthly) (default "daily")
  -h, --help                       help for deployment-cell
      --include-cells string       Deployment cell IDs to include (comma-separated)
      --include-instances string   Instance IDs to include (comma-separated)
      --include-providers string   Cloud provider IDs to include (comma-separated)
      --start-date string          Start date for cost analysis (RFC3339 format) (required)
      --top-n-instances int        Limit results to top N instances by cost
```

### Options inherited from parent commands

```
  -o, --output string   Output format (text|table|json) (default "table")
  -v, --version         Print the version number of omnistrate-ctl
```

### SEE ALSO

* [omnistrate-ctl cost](omnistrate-ctl_cost.md)	 - Manage cost analytics for your services

