## omnistrate-ctl docs validate

Validate a spec file against the authoritative JSON schema

### Synopsis

This command validates a Docker Compose spec or a ServicePlanSpec against the JSON
schema served by the Omnistrate API, and reports every field that violates it.

Use it to check a spec before building. It catches unknown and misplaced fields,
wrong types, and missing required fields without creating anything in your account.

Two limits are worth knowing. The schema does not enumerate allowed values, so a
field with a valid type but an invalid value (a wrong tenancyType or cloudProvider)
passes — confirm values with 'docs compose-spec' or 'docs plan-spec'. And the compose
schema accepts any 'x-' key, so a misspelled extension name passes here; check
spelling against 'docs compose-spec'.

```
omnistrate-ctl docs validate --file <spec.yaml> [flags]
```

### Examples

```
# Validate a compose spec (the spec type is detected from the file)
omnistrate-ctl docs validate --file docker-compose.yaml

# Validate a ServicePlanSpec
omnistrate-ctl docs validate --file spec.yaml

# Force a spec type when detection is ambiguous
omnistrate-ctl docs validate --file spec.yaml --spec-type service-plan

# Machine-readable output, for a pre-commit hook or CI step
omnistrate-ctl docs validate --file spec.yaml --output json

```

### Options

```
  -f, --file string          Path to the spec file to validate (required)
  -h, --help                 help for validate
  -o, --output string        Output format (markdown|json|table|text). markdown preserves the full section text; table truncates it to one line per row (default "markdown")
      --schema-file string   Validate against a schema on disk instead of fetching one, for offline or pinned-schema use
      --spec-type string     Spec type: compose|service-plan (detected from the file when omitted)
```

### Options inherited from parent commands

```
  -v, --version   Print the version number of omnistrate-ctl
```

### SEE ALSO

* [omnistrate-ctl docs](omnistrate-ctl_docs.md)	 - Search and access documentation

