## omnistrate-ctl secret list

List environment secrets

### Synopsis

This command helps you list all secrets for a specific environment type.

```text
omnistrate-ctl secret list [environment-type] [flags]
```

### Examples

```text
# List secrets for dev environment
omnistrate-ctl secret list dev

# List secrets for prod environment with JSON output
omnistrate-ctl secret list prod --output json
```

### Options

```text
  -h, --help   help for list
```

### Options inherited from parent commands

```text
  -o, --output string   Output format (text|table|json) (default "table")
  -v, --version         Print the version number of omnistrate-ctl
```

### SEE ALSO

- [omnistrate-ctl secret](../omnistrate-ctl_secret/) - Manage secrets
