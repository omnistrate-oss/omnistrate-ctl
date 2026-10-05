## omnistrate-ctl managed-artifact sync describe

Describe a managed artifact synchronization record

### Synopsis

Describe a private synchronization or public publication. Use --output json for source references, digests, completion timestamps, and full execution-target metadata.

```
omnistrate-ctl managed-artifact sync describe [flags]
```

### Examples

```
omnistrate-ctl managed-artifact sync describe --id spabs-123
omnistrate-ctl managed-artifact sync describe --id sppap-123 -o json
```

### Options

```
  -h, --help        help for describe
      --id string   Private sync (spabs-*) or public publication (sppap-*) ID (required)
```

### Options inherited from parent commands

```
  -o, --output string   Output format (text|table|json) (default "table")
  -v, --version         Print the version number of omnistrate-ctl
```

### SEE ALSO

* [omnistrate-ctl managed-artifact sync](omnistrate-ctl_managed-artifact_sync.md)	 - Inspect private artifact syncs and public ECR publications

