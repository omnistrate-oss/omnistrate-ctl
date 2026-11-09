## omnistrate-ctl deployment-cell describe

Describe a deployment cell

### Synopsis

Describe a deployment cell by ID, including its current managed artifact version.

```
omnistrate-ctl deployment-cell describe [deployment-cell-id] [flags]
```

### Examples

```
# Describe a deployment cell
omnistrate-ctl deployment-cell describe hc-12345678

# Describe a deployment cell with JSON output
omnistrate-ctl deployment-cell describe hc-12345678 --output json
```

### Options

```
  -h, --help   help for describe
```

### Options inherited from parent commands

```
  -o, --output string   Output format (text|table|json) (default "table")
  -v, --version         Print the version number of omnistrate-ctl
```

### SEE ALSO

* [omnistrate-ctl deployment-cell](omnistrate-ctl_deployment-cell.md)	 - Manage Deployment Cells

