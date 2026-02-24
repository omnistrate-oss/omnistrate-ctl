## omnistrate-ctl instance debug helm-values

Get Helm chart values for instance resources

### Synopsis

Get Helm chart values for instance resources. Use --resource-id or --resource-key to filter by specific resource.

```
omnistrate-ctl instance debug helm-values [instance-id] [flags]
```

### Examples

```
  omnistrate-ctl instance debug helm-values <instance-id>
  omnistrate-ctl instance debug helm-values <instance-id> --resource-key my-resource
  omnistrate-ctl instance debug helm-values <instance-id> --resource-id abc123
```

### Options

```
  -h, --help                  help for helm-values
      --resource-id string    Filter by resource ID
      --resource-key string   Filter by resource key
```

### Options inherited from parent commands

```
  -o, --output string   Output format (text|table|json) (default "table")
  -v, --version         Print the version number of omnistrate-ctl
```

### SEE ALSO

* [omnistrate-ctl instance debug](omnistrate-ctl_instance_debug.md)	 - Visualize the instance plan DAG

