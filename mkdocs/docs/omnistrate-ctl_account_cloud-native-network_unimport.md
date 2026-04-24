## omnistrate-ctl account cloud-native-network unimport

Unimport a cloud-native network (revert to AVAILABLE)

### Synopsis

Reverts a previously imported cloud-native network from READY back to AVAILABLE status, removing it from the deployment target pool.

```
omnistrate-ctl account cloud-native-network unimport [account-id] --network-id=[network-id] [flags]
```

### Examples

```
# Unimport a cloud-native network (revert to AVAILABLE)
omnistrate-ctl account cloud-native-network unimport [account-id] --network-id=[network-id]
```

### Options

```
  -h, --help                help for unimport
      --network-id string   The cloud-native network ID to unimport (required)
```

### Options inherited from parent commands

```
  -o, --output string   Output format (text|table|json) (default "table")
  -v, --version         Print the version number of omnistrate-ctl
```

### SEE ALSO

* [omnistrate-ctl account cloud-native-network](omnistrate-ctl_account_cloud-native-network.md)	 - Manage cloud-native networks (VPCs) for a BYOA Cloud Provider Account

