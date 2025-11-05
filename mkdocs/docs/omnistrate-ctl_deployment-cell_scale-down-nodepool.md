## omnistrate-ctl deployment-cell scale-down-nodepool

Scale down a nodepool to minimum size

### Synopsis

Scale down a nodepool by setting max size to 0.

This can be used as a cost saving measure to reduce the nodepool capacity.
When set to 0, the nodepool will remain configured but will have no running nodes.

```
omnistrate-ctl deployment-cell scale-down-nodepool [flags]
```

### Options

```
  -h, --help              help for scale-down-nodepool
  -i, --id string         Deployment cell ID (required)
  -n, --nodepool string   Nodepool name (required)
```

### Options inherited from parent commands

```
  -o, --output string   Output format (text|table|json) (default "table")
  -v, --version         Print the version number of omnistrate-ctl
```

### SEE ALSO

* [omnistrate-ctl deployment-cell](omnistrate-ctl_deployment-cell.md)	 - Manage Deployment Cells

