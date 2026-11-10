## omnistrate-ctl deployment-cell describe

Describe a deployment cell

### Synopsis

Describe a deployment cell by ID or key, including its current and effective managed artifact versions.

```
omnistrate-ctl deployment-cell describe [deployment-cell-id-or-key] [flags]
```

### Examples

```
# Describe a deployment cell
omnistrate-ctl deployment-cell describe hc-12345678

# Describe an adopted deployment cell by key
omnistrate-ctl deployment-cell describe my-adopted-cell

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

