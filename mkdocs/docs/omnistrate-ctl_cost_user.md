## omnistrate-ctl cost user

Get cost breakdown by user

### Synopsis

Get the total cost of operating your fleet for different users.

```
omnistrate-ctl cost user [flags]
```

### Options

```
      --end-date string           End date for cost analysis (RFC3339 format) (required)
  -e, --environment-type string   Environment type (required)
      --exclude-users string      User IDs to exclude (comma-separated)
  -h, --help                      help for user
      --include-users string      User IDs to include (comma-separated)
      --start-date string         Start date for cost analysis (RFC3339 format) (required)
      --top-n-instances int       Limit results to top N instances by cost
      --top-n-users int           Limit results to top N users by cost
```

### Options inherited from parent commands

```
  -o, --output string   Output format (text|table|json) (default "table")
  -v, --version         Print the version number of omnistrate-ctl
```

### SEE ALSO

* [omnistrate-ctl cost](omnistrate-ctl_cost.md)	 - Manage cost analytics for your services

