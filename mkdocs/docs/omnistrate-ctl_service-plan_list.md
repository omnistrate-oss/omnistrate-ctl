## omnistrate-ctl service-plan list

List Service Plans for your service

### Synopsis

This command helps you list Service Plans for your service.
You can filter for specific service plans by using the filter flag.

```
omnistrate-ctl service-plan list [flags]
```

### Examples

```
# List service plans of the service postgres in the prod and dev environments
omctl service-plan list -f="service_name:postgres,environment:prod" -f="service:postgres,environment:dev"
```

### Options

```
  -f, --filter stringArray   Filter to apply to the list of service plans. E.g.: key1:value1,key2:value2, which filters service plans where key1 equals value1 and key2 equals value2. Allow use of multiple filters to form the logical OR operation. Supported keys: plan_id,plan_name,service_id,service_name,environment,version,release_description,version_set_status. Check the examples for more details.
  -h, --help                 help for list
      --truncate             Truncate long names in the output
```

### Options inherited from parent commands

```
  -o, --output string   Output format (text|table|json) (default "table")
  -v, --version         Print the version number of omnistrate-ctl
```

### SEE ALSO

- [omnistrate-ctl service-plan](omnistrate-ctl_service-plan.md) - Manage Service Plans for your service
