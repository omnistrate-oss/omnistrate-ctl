## omnistrate-ctl service list

List services for your account

### Synopsis

This command helps you list services for your account.
You can filter for specific services by using the filter flag.

```
omnistrate-ctl service list [flags]
```

### Examples

```
# List services
omctl service list
```

### Options

```
  -f, --filter stringArray   Filter to apply to the list of services. E.g.: key1:value1,key2:value2, which filters services where key1 equals value1 and key2 equals value2. Allow use of multiple filters to form the logical OR operation. Supported keys: id,name,environments. Check the examples for more details.
  -h, --help                 help for list
      --truncate             Truncate long names in the output
```

### Options inherited from parent commands

```
  -o, --output string   Output format (text|table|json) (default "table")
  -v, --version         Print the version number of omnistrate-ctl
```

### SEE ALSO

* [omnistrate-ctl service](omnistrate-ctl_service.md)	 - Manage Services for your account

