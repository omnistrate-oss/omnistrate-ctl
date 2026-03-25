## omnistrate-ctl docs compose-spec

Compose spec documentation

### Synopsis

This command returns information about the Omnistrate Docker Compose specification. If no tag is provided, it lists all supported tags. If a tag is provided, it returns the information about the tag.

```
omnistrate-ctl docs compose-spec [tag] [flags]
```

### Examples

```
# List all H3 headers in the compose spec documentation with JSON output
omnistrate-ctl docs compose-spec --output json

# Search for a specific tag with JSON output
omnistrate-ctl docs compose-spec "networks" --output json

# Search for specific custom tags with JSON output
omnistrate-ctl docs compose-spec "x-omnistrate-compute" --output json

```

### Options

```
  -h, --help               help for compose-spec
      --json-schema-only   Return only the JSON schema for the specified tag
  -o, --output string      Output format (table|json) (default "table")
```

### Options inherited from parent commands

```
  -v, --version   Print the version number of omnistrate-ctl
```

### SEE ALSO

* [omnistrate-ctl docs](omnistrate-ctl_docs.md)	 - Search and access documentation

