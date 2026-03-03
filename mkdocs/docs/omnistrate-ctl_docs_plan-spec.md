## omnistrate-ctl docs plan-spec

Plan spec documentation

### Synopsis

This command returns information about the Omnistrate Plan specification. If no tag is provided, it lists all supported tags. If a tag is provided, it returns the information about the tag.

```
omnistrate-ctl docs plan-spec [tag] [flags]
```

### Examples

```
# List all H3 headers in the plan spec documentation with JSON output
omnistrate-ctl docs plan-spec --output json

# Search for a specific tag with JSON output
omnistrate-ctl docs plan-spec "compute" --output json

# Search for specific schema tags with JSON output
omnistrate-ctl docs plan-spec "helm chart configuration" --output json

```

### Options

```
  -h, --help               help for plan-spec
      --json-schema-only   Return only the JSON schema for the specified tag
  -o, --output string      Output format (table|json) (default "table")
```

### Options inherited from parent commands

```
  -v, --version   Print the version number of omnistrate-ctl
```

### SEE ALSO

* [omnistrate-ctl docs](omnistrate-ctl_docs.md)	 - Search and access documentation

