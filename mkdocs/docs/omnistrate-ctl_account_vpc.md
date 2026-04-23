## omnistrate-ctl account vpc

Manage VPCs for a Cloud Provider Account

### Synopsis

This command helps you discover, import, and manage VPCs associated with your BYOA cloud provider accounts.

```
omnistrate-ctl account vpc [operation] [flags]
```

### Options

```
  -h, --help   help for vpc
```

### Options inherited from parent commands

```
  -o, --output string   Output format (text|table|json) (default "table")
  -v, --version         Print the version number of omnistrate-ctl
```

### SEE ALSO

* [omnistrate-ctl account](omnistrate-ctl_account.md)	 - Manage your Cloud Provider Accounts
* [omnistrate-ctl account vpc bulk-import](omnistrate-ctl_account_vpc_bulk-import.md)	 - Import multiple VPCs at once
* [omnistrate-ctl account vpc import](omnistrate-ctl_account_vpc_import.md)	 - Import an AVAILABLE VPC for deployments
* [omnistrate-ctl account vpc list](omnistrate-ctl_account_vpc_list.md)	 - List VPCs for a Cloud Provider Account
* [omnistrate-ctl account vpc sync](omnistrate-ctl_account_vpc_sync.md)	 - Discover and sync VPCs from the cloud provider
* [omnistrate-ctl account vpc unimport](omnistrate-ctl_account_vpc_unimport.md)	 - Unimport a VPC (revert to AVAILABLE)

