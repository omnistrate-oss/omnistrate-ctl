## omnistrate-ctl docs system-parameters

Get the JSON schema for system parameters

### Synopsis

This command returns the JSON schema for system parameters from the Omnistrate API.

```text
omnistrate-ctl docs system-parameters [flags]
```

### Examples

```text
# Get the JSON schema for system parameters
omnistrate-ctl docs system-parameters

# Get the JSON schema for system parameters with JSON output
omnistrate-ctl docs system-parameters --output json
```

### Options

```text
  -h, --help            help for system-parameters
  -o, --output string   Output format (table|json) (default "table")
```

### Options inherited from parent commands

```text
  -v, --version   Print the version number of omnistrate-ctl
```

### SEE ALSO

- [omnistrate-ctl docs](../omnistrate-ctl_docs/) - Search and access documentation
