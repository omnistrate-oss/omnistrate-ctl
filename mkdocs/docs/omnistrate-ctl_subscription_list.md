## omnistrate-ctl subscription list

List Customer Subscriptions to your services

### Synopsis

This command helps you list Customer Subscriptions to your services.
You can filter for specific subscriptions by using the filter flag.

```
omnistrate-ctl subscription list [flags]
```

### Examples

```
# List subscriptions of the service postgres and mysql in the prod environment
omctl subscription list -f="service_name:postgres,environment:prod" -f="service:mysql,environment:prod"
```

### Options

```
  -f, --filter stringArray   Filter to apply to the list of subscriptions. E.g.: key1:value1,key2:value2, which filters subscriptions where key1 equals value1 and key2 equals value2. Allow use of multiple filters to form the logical OR operation. Supported keys: subscription_id,service_id,service_name,plan_id,plan_name,environment,subscription_owner_name,subscription_owner_email,status. Check the examples for more details.
  -h, --help                 help for list
      --truncate             Truncate long names in the output
```

### Options inherited from parent commands

```
  -o, --output string   Output format (text|table|json) (default "table")
  -v, --version         Print the version number of omnistrate-ctl
```

### SEE ALSO

* [omnistrate-ctl subscription](omnistrate-ctl_subscription.md)	 - Manage Customer Subscriptions for your service

