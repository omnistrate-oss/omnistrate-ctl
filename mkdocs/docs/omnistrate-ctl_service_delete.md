## omnistrate-ctl service delete

Delete a service

### Synopsis

This command helps you delete a service using its name or ID.

```
omnistrate-ctl service delete [service-name] [flags]
```

### Examples

```
# Delete service with name
omnistrate-ctl service delete [service-name]

# Delete service with ID
omnistrate-ctl service delete --id=[service-ID]
```

### Options

```
  -h, --help        help for delete
      --id string   Service ID
```

### Options inherited from parent commands

```
  -o, --output string   Output format (text|table|json) (default "table")
  -v, --version         Print the version number of omnistrate-ctl
```

### SEE ALSO

* [omnistrate-ctl service](omnistrate-ctl_service.md)	 - Manage Services for your account

