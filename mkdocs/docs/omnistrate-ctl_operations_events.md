## omnistrate-ctl operations events

List operational events

### Synopsis

List operational events for services, environments, and instances with filtering options.

```
omnistrate-ctl operations events [flags]
```

### Options

```
      --end-date string                 End date of events (RFC3339 format)
  -e, --environment-type string         Environment type to filter by
      --event-types strings             Event types to filter by (comma-separated)
  -h, --help                            help for events
  -i, --instance-id string              Instance ID to list events for
      --next-page-token string          Token for next page of results
      --page-size int                   Number of results per page
      --product-tier-id string          Product tier ID to filter by
      --service-environment-id string   Service environment ID to list events for
  -s, --service-id string               Service ID to list events for
      --start-date string               Start date of events (RFC3339 format)
```

### Options inherited from parent commands

```
  -o, --output string   Output format (text|table|json) (default "table")
  -v, --version         Print the version number of omnistrate-ctl
```

### SEE ALSO

* [omnistrate-ctl operations](omnistrate-ctl_operations.md)	 - Operations and health monitoring commands

