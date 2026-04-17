## omnistrate-ctl secret delete

Delete an environment secret

### Synopsis

This command helps you delete a secret from a specific environment type.

```text
omnistrate-ctl secret delete [environment-type] [secret-name] [flags]
```

### Examples

```text
# Delete a secret from dev environment
omnistrate-ctl environment secret delete dev my-secret

# Delete a secret from prod environment
omnistrate-ctl environment secret delete prod db-password
```

### Options

```text
  -h, --help   help for delete
```

### Options inherited from parent commands

```text
  -o, --output string   Output format (text|table|json) (default "table")
  -v, --version         Print the version number of omnistrate-ctl
```

### SEE ALSO

- [omnistrate-ctl secret](../omnistrate-ctl_secret/) - Manage secrets
