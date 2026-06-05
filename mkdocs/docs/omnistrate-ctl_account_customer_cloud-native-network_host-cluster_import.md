## omnistrate-ctl account customer cloud-native-network host-cluster import

Import a host cluster from a cloud-native network

### Synopsis

Imports a provider host cluster from an imported cloud-native network into Omnistrate.

```
omnistrate-ctl account customer cloud-native-network host-cluster import [account-id] --region=[region] --network-id=[network-id] --host-cluster-name=[name] [flags]
```

### Examples

```
# Import a host cluster from a cloud-native network
omnistrate-ctl account customer cloud-native-network host-cluster import [account-id] --region=[region] --network-id=[network-id] --host-cluster-name=[name]
```

### Options

```
  -h, --help                       help for import
      --host-cluster-name string   The provider host cluster name to import (required)
      --network-id string          The cloud-native network ID that contains the host cluster (required)
      --region string              The cloud provider region of the cloud-native network (required)
```

### Options inherited from parent commands

```
  -o, --output string   Output format (text|table|json) (default "table")
  -v, --version         Print the version number of omnistrate-ctl
```

### SEE ALSO

* [omnistrate-ctl account customer cloud-native-network host-cluster](omnistrate-ctl_account_customer_cloud-native-network_host-cluster.md)	 - Manage imported host clusters for cloud-native networks

