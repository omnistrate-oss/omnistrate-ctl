## omnistrate-ctl docs plan-spec

Plan spec documentation

### Synopsis

This command returns information about the Omnistrate Plan specification. If no tag is provided, it lists all supported tags. If a tag is provided, it returns the information about the tag.

```
omnistrate-ctl docs plan-spec [tag] [flags]
```

### Examples

```
# List every section of the plan spec documentation
omnistrate-ctl docs plan-spec

# Read one section (full text, markdown tables preserved)
omnistrate-ctl docs plan-spec "Root schema"

# Read a section as JSON, for scripting
omnistrate-ctl docs plan-spec "compute" --output json

# Get the JSON schema that defines a section
omnistrate-ctl docs plan-spec "helm chart configuration" --json-schema-only

```

### Options

```
  -h, --help               help for plan-spec
      --json-schema-only   Return only the JSON schema covering the specified tag. Plan spec tags are definitions within the ServicePlanSpec schema, so this returns that schema; use 'docs json-schema' to request a schema type directly
  -o, --output string      Output format (markdown|json|table|text). markdown preserves the full section text; table truncates it to one line per row (default "markdown")
```

### Options inherited from parent commands

```
  -v, --version   Print the version number of omnistrate-ctl
```

### SEE ALSO

* [omnistrate-ctl docs](omnistrate-ctl_docs.md)	 - Search and access documentation

