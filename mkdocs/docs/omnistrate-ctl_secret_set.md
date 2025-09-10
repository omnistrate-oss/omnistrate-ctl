## omnistrate-ctl secret set

Set an environment secret

### Synopsis

This command helps you create or update a secret for a specific environment type.

```
omnistrate-ctl secret set [environment-type] [secret-name] [secret-value] [flags]
```

### Examples

```
# Set a secret for dev environment
omctl secret set dev my-secret my-value

# Set a secret for prod environment
omctl secret set prod db-password secret123
```

### Options

```
  -h, --help   help for set
```

### Options inherited from parent commands

```
  -o, --output string   Output format (text|table|json) (default "table")
  -v, --version         Print the version number of omnistrate-ctl
```

### SEE ALSO

- [omnistrate-ctl secret](omnistrate-ctl_secret.md) - Manage secrets
