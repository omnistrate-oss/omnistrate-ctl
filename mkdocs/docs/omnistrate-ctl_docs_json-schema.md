## omnistrate-ctl docs json-schema

Get the JSON schema for a spec type

### Synopsis

This command returns the JSON schema for a spec type from the Omnistrate API. If no type is provided, it lists the types that can be requested. Use this to validate a compose or plan spec against the authoritative schema.

```
omnistrate-ctl docs json-schema [type] [flags]
```

### Examples

```
# List the schema types that can be requested
omnistrate-ctl docs json-schema

# Get the full Docker Compose spec schema, including every Omnistrate extension
omnistrate-ctl docs json-schema compose --output json

# Get the full Plan spec (ServicePlanSpec) schema
omnistrate-ctl docs json-schema service-plan --output json

# Get the schema for a single compose extension
omnistrate-ctl docs json-schema x-omnistrate-compute --output json

```

### Options

```
  -h, --help            help for json-schema
  -o, --output string   Output format (json|markdown|table|text). json is the default because a JSON schema cannot be represented as a table (default "json")
```

### Options inherited from parent commands

```
  -v, --version   Print the version number of omnistrate-ctl
```

### SEE ALSO

* [omnistrate-ctl docs](omnistrate-ctl_docs.md)	 - Search and access documentation

