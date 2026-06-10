## omnistrate-ctl account customer cloud-native-network sync

Sync cloud-native networks from the cloud provider

### Synopsis

Discovers or re-validates cloud-native networks for a BYOA account configuration.

```
omnistrate-ctl account customer cloud-native-network sync [account-id] [flags]
```

### Examples

```
# Sync all cloud-native networks for an account
omnistrate-ctl account customer cloud-native-network sync [account-id]

# Sync all networks in specific regions
omnistrate-ctl account customer cloud-native-network sync [account-id] --region=us-east-1 --region=us-west-2

# Sync specific networks
omnistrate-ctl account customer cloud-native-network sync [account-id] --region=us-east-1 --network-id=vpc-abc123

# Sync networks and include host clusters in discovery
omnistrate-ctl account customer cloud-native-network sync [account-id] --region=us-east-1 --include-host-clusters
```

### Options

```
  -h, --help                    help for sync
      --include-host-clusters   Include host clusters when refreshing targeted cloud-native networks
      --network strings         Specific network to sync in region:network-id format (repeatable)
      --network-id strings      Cloud-native network ID to sync in the specified region (repeatable)
      --region strings          Cloud region to discover (repeatable)
```

### Options inherited from parent commands

```
  -o, --output string   Output format (text|table|json) (default "table")
  -v, --version         Print the version number of omnistrate-ctl
```

### SEE ALSO

* [omnistrate-ctl account customer cloud-native-network](omnistrate-ctl_account_customer_cloud-native-network.md)	 - Manage cloud-native networks (VPCs) for a BYOA Cloud Provider Account

