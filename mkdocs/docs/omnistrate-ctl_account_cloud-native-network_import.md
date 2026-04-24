## omnistrate-ctl account cloud-native-network import

Import an AVAILABLE cloud-native network for deployments

### Synopsis

Imports a discovered cloud-native network, changing its status from AVAILABLE to READY so it can be used for service deployments.

```
omnistrate-ctl account cloud-native-network import [account-id] --network-id=[network-id] [flags]
```

### Examples

```
# Import a cloud-native network to make it available for deployments
omnistrate-ctl account cloud-native-network import [account-id] --network-id=[network-id]
```

### Options

```
  -h, --help                help for import
      --network-id string   The cloud-native network ID to import (required)
```

### Options inherited from parent commands

```
  -o, --output string   Output format (text|table|json) (default "table")
  -v, --version         Print the version number of omnistrate-ctl
```

### SEE ALSO

* [omnistrate-ctl account cloud-native-network](omnistrate-ctl_account_cloud-native-network.md)	 - Manage cloud-native networks (VPCs) for a BYOA Cloud Provider Account

