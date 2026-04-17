## omnistrate-ctl service describe

Describe a service

### Synopsis

This command helps you describe a service using its name or ID.

```text
omnistrate-ctl service describe [flags]
```

### Examples

```text
# Describe service with name
omnistrate-ctl service describe [service-name]

# Describe service with ID
omnistrate-ctl service describe --id=[service-ID]
```

### Options

```text
  -h, --help            help for describe
      --id string       Service ID
  -o, --output string   Output format. Only json is supported. (default "json")
```

### Options inherited from parent commands

```text
  -v, --version   Print the version number of omnistrate-ctl
```

### SEE ALSO

- [omnistrate-ctl service](../omnistrate-ctl_service/) - Manage Services for your account
