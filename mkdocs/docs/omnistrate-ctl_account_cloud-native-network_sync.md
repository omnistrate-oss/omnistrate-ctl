## omnistrate-ctl account cloud-native-network sync

Discover and sync cloud-native networks from the cloud provider

### Synopsis

Triggers cloud-native network discovery for a BYOA cloud provider account. Discovered networks will appear with AVAILABLE status and can then be imported.

```
omnistrate-ctl account cloud-native-network sync [account-id] [flags]
```

### Examples

```
# Sync cloud-native networks for an account
omnistrate-ctl account cloud-native-network sync [account-id]
```

### Options

```
  -h, --help              help for sync
      --regions strings   Cloud regions to discover networks in (comma-separated). Defaults to all regions from the service plan.
```

### Options inherited from parent commands

```
  -o, --output string   Output format (text|table|json) (default "table")
  -v, --version         Print the version number of omnistrate-ctl
```

### SEE ALSO

* [omnistrate-ctl account cloud-native-network](omnistrate-ctl_account_cloud-native-network.md)	 - Manage cloud-native networks (VPCs) for a BYOA Cloud Provider Account

