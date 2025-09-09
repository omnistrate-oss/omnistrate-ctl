## omnistrate-ctl deployment-cell debug

Debug deployment cell resources and retrieve custom helm execution logs

### Synopsis

Debug deployment cell resources with custom helm execution logs and save them to a specified output directory.

```
omnistrate-ctl deployment-cell debug [flags]
```

### Examples

```
  omnistrate-ctl deployment-cell debug --id <deployment-cell-id> --output-dir ./debug-output
```

### Options

```
  -h, --help                help for debug
  -i, --id string           Deployment cell ID (required)
  -d, --output-dir string   Output directory to save debug logs (default "./debug-output")
```

### Options inherited from parent commands

```
  -o, --output string   Output format (text|table|json) (default "table")
  -v, --version         Print the version number of omnistrate-ctl
```

### SEE ALSO

- [omnistrate-ctl deployment-cell](omnistrate-ctl_deployment-cell.md) - Manage Deployment Cells
