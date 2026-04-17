## omnistrate-ctl instance debug terraform-files

Get Terraform files for instance resources

### Synopsis

Get Terraform files for instance resources. Use --resource-id or --resource-key to filter by specific resource.

```text
omnistrate-ctl instance debug terraform-files [instance-id] [flags]
```

### Examples

```text
  omnistrate-ctl instance debug terraform-files <instance-id>
  omnistrate-ctl instance debug terraform-files <instance-id> --resource-key my-resource
  omnistrate-ctl instance debug terraform-files <instance-id> --resource-id abc123
```

### Options

```text
  -h, --help                   help for terraform-files
      --resource-id string     Filter by resource ID
      --resource-key string    Filter by resource key
      --resource-name string   Filter by resource name
```

### Options inherited from parent commands

```text
  -o, --output string   Output format (text|table|json) (default "table")
  -v, --version         Print the version number of omnistrate-ctl
```

### SEE ALSO

- [omnistrate-ctl instance debug](../omnistrate-ctl_instance_debug/) - Visualize the instance plan DAG
