## omnistrate-ctl custom-network list

List custom networks

### Synopsis

This command helps you list existing custom networks.

```
omnistrate-ctl custom-network list [flags]
```

### Examples

```
# List all custom networks
omctl custom-network list

# List custom networks for a specific cloud provider and region
omctl custom-network list --filter="cloud_provider:aws,region:us-east-1"
```

### Options

```
  -f, --filter stringArray   Filter to apply to the list of custom networks. E.g.: key1:value1,key2:value2, which filters custom networks where key1 equals value1 and key2 equals value2. Allow use of multiple filters to form the logical OR operation. Supported keys: custom_network_id,custom_network_name,cloud_provider,region,cidr,owning_org_id,owning_org_name,aws_account_id,cloud_provider_native_network_id,gcp_project_id,gcp_project_number,host_cluster_id. Check the examples for more details.
  -h, --help                 help for list
```

### Options inherited from parent commands

```
  -o, --output string   Output format (text|table|json) (default "table")
  -v, --version         Print the version number of omnistrate-ctl
```

### SEE ALSO

- [omnistrate-ctl custom-network](omnistrate-ctl_custom-network.md) - List and describe custom networks of your customers
