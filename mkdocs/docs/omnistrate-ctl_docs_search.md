## omnistrate-ctl docs search

Search through Omnistrate documentation

### Synopsis

This command helps you search through Omnistrate documentation content for specific terms or topics.

```
omnistrate-ctl docs search [query] [flags]
```

### Examples

```
# Search documentation for a specific term
omnistrate-ctl docs search "kubernetes" --output json

# Search documentation with multiple terms with JSON output
omnistrate-ctl docs search "service plan deployment" --output json

# Limit the number of results returned
omnistrate-ctl docs search "service plan deployment" --limit 5 --output json

```

### Options

```
  -h, --help            help for search
  -l, --limit int       Maximum number of results to return (default 10)
  -o, --output string   Output format (table|json) (default "table")
```

### Options inherited from parent commands

```
  -v, --version   Print the version number of omnistrate-ctl
```

### SEE ALSO

* [omnistrate-ctl docs](omnistrate-ctl_docs.md)	 - Search and access documentation

