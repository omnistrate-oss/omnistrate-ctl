## omnistrate-ctl docs system-parameters

Get the JSON schema for system parameters

### Synopsis

This command returns the JSON schema for system parameters from the Omnistrate API.

```
omnistrate-ctl docs system-parameters [flags]
```

### Examples

```
# Get the JSON schema for system parameters
omnistrate-ctl docs system-parameters

# Get the JSON schema for system parameters with JSON output
omnistrate-ctl docs system-parameters --output json

```

### Options

```
  -h, --help            help for system-parameters
  -o, --output string   Output format (json|markdown|table|text). json is the default because a JSON schema cannot be represented as a table (default "json")
```

### Options inherited from parent commands

```
  -v, --version   Print the version number of omnistrate-ctl
```

### SEE ALSO

* [omnistrate-ctl docs](omnistrate-ctl_docs.md)	 - Search and access documentation

