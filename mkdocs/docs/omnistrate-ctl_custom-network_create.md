## omnistrate-ctl custom-network create

Create a custom network

### Synopsis

This command helps you create a new custom network.

```
omnistrate-ctl custom-network create [flags]
```

### Examples

```
# Create a custom network for specific cloud provider and region 
omnistrate-ctl custom-network create --cloud-provider=[cloud-provider-name] --region=[cloud-provider-region] --cidr=[cidr-block] --name=[friendly-network-name]
```

### Options

```
      --cidr string             Network CIDR block
      --cloud-provider string   Cloud provider name. Valid options include: 'aws', 'azure', 'gcp'
  -h, --help                    help for create
      --name string             Optional friendly name for the custom network
      --region string           Region for the custom network (format is cloud provider specific)
```

### Options inherited from parent commands

```
  -o, --output string   Output format (text|table|json) (default "table")
  -v, --version         Print the version number of omnistrate-ctl
```

### SEE ALSO

* [omnistrate-ctl custom-network](omnistrate-ctl_custom-network.md)	 - List and describe custom networks of your customers

