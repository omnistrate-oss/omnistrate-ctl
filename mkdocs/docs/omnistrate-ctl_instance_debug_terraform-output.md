## omnistrate-ctl instance debug terraform-output

Get Terraform logs for instance resources

### Synopsis

Get Terraform logs (apply, plan, etc.) for instance resources. Use --resource-id or --resource-key to filter by specific resource.

```
omnistrate-ctl instance debug terraform-output [instance-id] [flags]
```

### Examples

```
  omnistrate-ctl instance debug terraform-output <instance-id>
  omnistrate-ctl instance debug terraform-output <instance-id> --resource-key my-resource
  omnistrate-ctl instance debug terraform-output <instance-id> --resource-id abc123
```

### Options

```
  -h, --help                   help for terraform-output
      --resource-id string     Filter by resource ID
      --resource-key string    Filter by resource key
      --resource-name string   Filter by resource name
```

### Options inherited from parent commands

```
  -o, --output string   Output format (text|table|json) (default "table")
  -v, --version         Print the version number of omnistrate-ctl
```

### SEE ALSO

* [omnistrate-ctl instance debug](omnistrate-ctl_instance_debug.md)	 - Visualize the instance plan DAG

