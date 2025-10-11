## omnistrate-ctl service-plan list-versions

List Versions of a specific Service Plan

### Synopsis

This command helps you list Versions of a specific Service Plan.
You can filter for specific service plan versions by using the filter flag.

```
omnistrate-ctl service-plan list-versions [service-name] [plan-name] [flags]
```

### Examples

```
# List service plan versions of the service postgres in the prod and dev environments
omctl service-plan list-versions postgres postgres -f="service_name:postgres,environment:prod" -f="service:postgres,environment:dev"
```

### Options

```
      --environment string   Environment name. Use this flag with service name and plan name to describe the version in a specific environment
  -f, --filter stringArray   Filter to apply to the list of service plan versions. E.g.: key1:value1,key2:value2, which filters service plans where key1 equals value1 and key2 equals value2. Allow use of multiple filters to form the logical OR operation. Supported keys: plan_id,plan_name,service_id,service_name,environment,version,release_description,version_set_status,is_new_service_plan_version_created. Check the examples for more details.
  -h, --help                 help for list-versions
      --limit int            List only the latest N service plan versions (default -1)
      --plan-id string       Environment ID. Required if plan name is not provided
      --service-id string    Service ID. Required if service name is not provided
      --truncate             Truncate long names in the output
```

### Options inherited from parent commands

```
  -o, --output string   Output format (text|table|json) (default "table")
  -v, --version         Print the version number of omnistrate-ctl
```

### SEE ALSO

* [omnistrate-ctl service-plan](omnistrate-ctl_service-plan.md)	 - Manage Service Plans for your service

