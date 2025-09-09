## omnistrate-ctl audit list

List audit events

### Synopsis

List audit events with detailed filtering options for services, instances, and time ranges.

```
omnistrate-ctl audit list [flags]
```

### Options

```
      --end-date string              End date for events (RFC3339 format)
  -e, --environment-type string      Environment type to filter by
      --event-source-types strings   Event source types to filter by (comma-separated)
  -h, --help                         help for list
  -i, --instance-id string           Instance ID to filter by
      --next-page-token string       Token for next page of results
      --page-size int                Number of results per page
      --product-tier-id string       Product tier ID to filter by
  -s, --service-id string            Service ID to filter by
      --start-date string            Start date for events (RFC3339 format)
```

### Options inherited from parent commands

```
  -o, --output string   Output format (text|table|json) (default "table")
  -v, --version         Print the version number of omnistrate-ctl
```

### SEE ALSO

* [omnistrate-ctl audit](omnistrate-ctl_audit.md)	 - Audit events and logging management

