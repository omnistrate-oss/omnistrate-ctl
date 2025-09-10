## omnistrate-ctl service describe

Describe a service

### Synopsis

This command helps you describe a service using its name or ID.

```
omnistrate-ctl service describe [flags]
```

### Examples

```
# Describe service with name
omctl service describe [service-name]

# Describe service with ID
omctl service describe --id=[service-ID]
```

### Options

```
  -h, --help            help for describe
      --id string       Service ID
  -o, --output string   Output format. Only json is supported. (default "json")
```

### Options inherited from parent commands

```
  -v, --version   Print the version number of omnistrate-ctl
```

### SEE ALSO

- [omnistrate-ctl service](omnistrate-ctl_service.md) - Manage Services for your account
