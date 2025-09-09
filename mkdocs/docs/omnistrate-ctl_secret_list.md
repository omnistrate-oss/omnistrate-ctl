## omnistrate-ctl secret list

List environment secrets

### Synopsis

This command helps you list all secrets for a specific environment type.

```
omnistrate-ctl secret list [environment-type] [flags]
```

### Examples

```
# List secrets for dev environment
omctl secret list dev

# List secrets for prod environment with JSON output
omctl secret list prod --output json
```

### Options

```
  -h, --help   help for list
```

### Options inherited from parent commands

```
  -o, --output string   Output format (text|table|json) (default "table")
  -v, --version         Print the version number of omnistrate-ctl
```

### SEE ALSO

* [omnistrate-ctl secret](omnistrate-ctl_secret.md)	 - Manage secrets

