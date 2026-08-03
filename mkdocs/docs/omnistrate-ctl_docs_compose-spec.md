## omnistrate-ctl docs compose-spec

Compose spec documentation

### Synopsis

This command returns information about the Omnistrate Docker Compose specification. If no tag is provided, it lists all supported tags. If a tag is provided, it returns the information about the tag.

```
omnistrate-ctl docs compose-spec [tag] [flags]
```

### Examples

```
# List every tag and extension in the compose spec documentation
omnistrate-ctl docs compose-spec

# Read one tag (full text, examples and markdown tables preserved)
omnistrate-ctl docs compose-spec "x-omnistrate-compute"

# Read a tag as JSON, for scripting
omnistrate-ctl docs compose-spec "networks" --output json

# Get the JSON schema covering a tag, including nested tags
omnistrate-ctl docs compose-spec "x-omnistrate-capabilities.sidecars" --json-schema-only

```

### Options

```
  -h, --help               help for compose-spec
      --json-schema-only   Return only the JSON schema for the specified tag. Nested tags resolve to their root extension; use 'docs json-schema' to list every schema type or request one directly
  -o, --output string      Output format (markdown|json|table|text). markdown preserves the full section text; table truncates it to one line per row (default "markdown")
```

### Options inherited from parent commands

```
  -v, --version   Print the version number of omnistrate-ctl
```

### SEE ALSO

* [omnistrate-ctl docs](omnistrate-ctl_docs.md)	 - Search and access documentation

