## omnistrate-ctl account cloud-native-network import

Import cloud-native networks for deployments

### Synopsis

Marks discovered cloud-native networks as READY so they can be used as deployment targets.

```
omnistrate-ctl account cloud-native-network import [account-id] --region=[region] --network-id=[network-id] [flags]
```

### Examples

```
# Import a cloud-native network for deployments
omnistrate-ctl account cloud-native-network import [account-id] --region=[region] --network-id=[network-id]

# Import multiple cloud-native networks in the same region
omnistrate-ctl account cloud-native-network import [account-id] --region=[region] --network-id=[network-id-1] --network-id=[network-id-2]
```

### Options

```
  -h, --help                 help for import
      --network-id strings   Cloud-native network ID to import (repeatable, required)
      --region string        The cloud provider region of the cloud-native network to import (required)
```

### Options inherited from parent commands

```
  -o, --output string   Output format (text|table|json) (default "table")
  -v, --version         Print the version number of omnistrate-ctl
```

### SEE ALSO

* [omnistrate-ctl account cloud-native-network](omnistrate-ctl_account_cloud-native-network.md)	 - Manage cloud-native networks (VPCs) for a BYOA Cloud Provider Account

