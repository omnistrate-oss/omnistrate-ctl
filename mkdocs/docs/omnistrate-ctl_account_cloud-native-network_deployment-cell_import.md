## omnistrate-ctl account cloud-native-network deployment-cell import

Import a deployment cell from an imported cloud-native network

### Synopsis

Creates or returns an Omnistrate deployment cell record backed by an imported cloud-native network.

```
omnistrate-ctl account cloud-native-network deployment-cell import [account-id] --region=[region] --network-id=[network-id] --name=[name] [flags]
```

### Examples

```
# Import a deployment cell from an imported cloud-native network
omnistrate-ctl account cloud-native-network deployment-cell import ac-x9KpL2mQ7r --region=ap-south-1 --network-id=vpc-0f8a7c6d5e4b3a291 --name=imported-cell-r7x4q2
```

### Options

```
  -h, --help                help for import
      --name string         The cloud provider deployment cell name to import (required)
      --network-id string   The cloud-native network ID to import the deployment cell from (required)
      --region string       The region of the imported cloud-native network (required)
```

### Options inherited from parent commands

```
  -o, --output string   Output format (text|table|json) (default "table")
  -v, --version         Print the version number of omnistrate-ctl
```

### SEE ALSO

* [omnistrate-ctl account cloud-native-network deployment-cell](omnistrate-ctl_account_cloud-native-network_deployment-cell.md)	 - Manage deployment cells backed by imported cloud-native networks

