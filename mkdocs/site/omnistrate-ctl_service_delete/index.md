## omnistrate-ctl service delete

Delete a service

### Synopsis

This command helps you delete a service using its name or ID.

```text
omnistrate-ctl service delete [service-name] [flags]
```

### Examples

```text
# Delete service with name
omnistrate-ctl service delete [service-name]

# Delete service with ID
omnistrate-ctl service delete --id=[service-ID]
```

### Options

```text
  -h, --help        help for delete
      --id string   Service ID
```

### Options inherited from parent commands

```text
  -o, --output string   Output format (text|table|json) (default "table")
  -v, --version         Print the version number of omnistrate-ctl
```

### SEE ALSO

- [omnistrate-ctl service](../omnistrate-ctl_service/) - Manage Services for your account
