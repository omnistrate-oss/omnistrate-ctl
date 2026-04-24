## omnistrate-ctl account cloud-native-network bulk-import

Import multiple cloud-native networks at once

### Synopsis

Imports multiple discovered cloud-native networks in a single operation, changing their status from AVAILABLE to READY.

```
omnistrate-ctl account cloud-native-network bulk-import [account-id] --network-ids=[network-id1,network-id2,...] [flags]
```

### Examples

```
# Bulk import multiple cloud-native networks
omnistrate-ctl account cloud-native-network bulk-import [account-id] --network-ids=cnn-abc123,cnn-def456
```

### Options

```
  -h, --help                 help for bulk-import
      --network-ids string   Comma-separated list of cloud-native network IDs to import (required)
```

### Options inherited from parent commands

```
  -o, --output string   Output format (text|table|json) (default "table")
  -v, --version         Print the version number of omnistrate-ctl
```

### SEE ALSO

* [omnistrate-ctl account cloud-native-network](omnistrate-ctl_account_cloud-native-network.md)	 - Manage cloud-native networks (VPCs) for a BYOA Cloud Provider Account

