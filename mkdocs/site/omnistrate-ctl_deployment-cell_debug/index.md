## omnistrate-ctl deployment-cell debug

Debug deployment cell resources and retrieve custom helm execution logs

### Synopsis

Debug deployment cell resources with custom helm execution logs and save them to a specified output directory.

```text
omnistrate-ctl deployment-cell debug [flags]
```

### Examples

```text
  omnistrate-ctl deployment-cell debug --id <deployment-cell-id> --output-dir ./debug-output
```

### Options

```text
  -h, --help                help for debug
  -i, --id string           Deployment cell ID (required)
  -d, --output-dir string   Output directory to save debug logs (default "./debug-output")
```

### Options inherited from parent commands

```text
  -o, --output string   Output format (text|table|json) (default "table")
  -v, --version         Print the version number of omnistrate-ctl
```

### SEE ALSO

- [omnistrate-ctl deployment-cell](../omnistrate-ctl_deployment-cell/) - Manage Deployment Cells
