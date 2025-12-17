## omnistrate-ctl secret get

Get an environment secret

### Synopsis

This command helps you get a specific secret for an environment type.

```
omnistrate-ctl secret get [environment-type] [secret-name] [flags]
```

### Examples

```
# Get a secret in dev environment
omnistrate-ctl secret get dev my-secret

# Get a secret with JSON output
omnistrate-ctl secret get prod db-password --output json
```

### Options

```
  -h, --help   help for get
```

### Options inherited from parent commands

```
  -o, --output string   Output format (text|table|json) (default "table")
  -v, --version         Print the version number of omnistrate-ctl
```

### SEE ALSO

* [omnistrate-ctl secret](omnistrate-ctl_secret.md)	 - Manage secrets

