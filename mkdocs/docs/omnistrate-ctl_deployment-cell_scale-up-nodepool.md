## omnistrate-ctl deployment-cell scale-up-nodepool

Scale up a nodepool to the default maximum size

### Synopsis

Scale up a nodepool to the default maximum size of 450 nodes.

This restores the nodepool to its default capacity after being scaled down.
Nodes will be provisioned as needed by the autoscaler.

```
omnistrate-ctl deployment-cell scale-up-nodepool [flags]
```

### Options

```
  -h, --help              help for scale-up-nodepool
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

