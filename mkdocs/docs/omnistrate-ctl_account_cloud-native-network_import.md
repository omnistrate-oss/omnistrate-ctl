## omnistrate-ctl account cloud-native-network import

Import one or more AVAILABLE cloud-native networks for deployments

### Synopsis

Imports discovered cloud-native networks, changing their status from AVAILABLE to READY so they can be used for service deployments. Use --network-id for a single network or --network-ids for bulk import.

```
omnistrate-ctl account cloud-native-network import [account-id] [flags]
```

### Examples

```
# Import a single cloud-native network
omnistrate-ctl account cloud-native-network import [account-id] --network-id=[network-id]

# Import multiple cloud-native networks at once
omnistrate-ctl account cloud-native-network import [account-id] --network-ids=vpc-abc123,vpc-def456
```

### Options

```
  -h, --help                 help for import
      --network-id string    The cloud-native network ID to import
      --network-ids string   Comma-separated list of cloud-native network IDs to import
```

### Options inherited from parent commands

```
  -o, --output string   Output format (text|table|json) (default "table")
  -v, --version         Print the version number of omnistrate-ctl
```

### SEE ALSO

* [omnistrate-ctl account cloud-native-network](omnistrate-ctl_account_cloud-native-network.md)	 - Manage cloud-native networks (VPCs) for a BYOA Cloud Provider Account

