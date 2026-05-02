## omnistrate-ctl account cloud-native-network list

List cloud-native networks for a BYOA Cloud Provider Account

### Synopsis

Lists all cloud-native networks registered with a BYOA cloud provider account, including their status (AVAILABLE, READY, SYNCING, etc).

```
omnistrate-ctl account cloud-native-network list [account-id] [flags]
```

### Examples

```
# List all cloud-native networks for an account
omnistrate-ctl account cloud-native-network list [account-id]
```

### Options

```
  -h, --help   help for list
```

### Options inherited from parent commands

```
  -o, --output string   Output format (text|table|json) (default "table")
  -v, --version         Print the version number of omnistrate-ctl
```

### SEE ALSO

* [omnistrate-ctl account cloud-native-network](omnistrate-ctl_account_cloud-native-network.md)	 - Manage cloud-native networks (VPCs) for a BYOA Cloud Provider Account

