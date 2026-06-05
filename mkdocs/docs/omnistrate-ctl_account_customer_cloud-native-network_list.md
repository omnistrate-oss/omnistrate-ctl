## omnistrate-ctl account customer cloud-native-network list

List cloud-native networks for a BYOA account

### Synopsis

Lists the cloud-native networks registered under a BYOA account configuration, including import and in-use status.

```
omnistrate-ctl account customer cloud-native-network list [account-id] [flags]
```

### Examples

```
# List cloud-native networks registered under an account
omnistrate-ctl account customer cloud-native-network list [account-id]
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

* [omnistrate-ctl account customer cloud-native-network](omnistrate-ctl_account_customer_cloud-native-network.md)	 - Manage cloud-native networks (VPCs) for a BYOA Cloud Provider Account

