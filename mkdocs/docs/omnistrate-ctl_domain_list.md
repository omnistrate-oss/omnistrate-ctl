## omnistrate-ctl domain list

List SaaS Portal custom domains

### Synopsis

This command helps you list SaaS Portal custom domains.
You can filter for specific domains by using the filter flag.

```
omnistrate-ctl domain list [flags]
```

### Examples

```
# List domains
omnistrate domain list -o=table
```

### Options

```
  -f, --filter stringArray   Filter to apply to the list of domains. E.g.: key1:value1,key2:value2, which filters domains where key1 equals value1 and key2 equals value2. Allow use of multiple filters to form the logical OR operation. Supported keys: environment_type,name,domain,status,cluster_endpoint. Check the examples for more details.
  -h, --help                 help for list
  -o, --output string        Output format (text|table|json) (default "text")
```

### Options inherited from parent commands

```
  -v, --version   Print the version number of omnistrate-ctl
```

### SEE ALSO

* [omnistrate-ctl domain](omnistrate-ctl_domain.md)	 - Manage Customer Domains for your service

