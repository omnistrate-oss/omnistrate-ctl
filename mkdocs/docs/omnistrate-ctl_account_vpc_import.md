## omnistrate-ctl account vpc import

Import an AVAILABLE VPC for deployments

### Synopsis

Imports a discovered VPC, changing its status from AVAILABLE to READY so it can be used for service deployments.

```
omnistrate-ctl account vpc import [account-id] --vpc-id=[vpc-id] [flags]
```

### Examples

```
# Import a VPC to make it available for deployments
omnistrate-ctl account vpc import [account-id] --vpc-id=[vpc-id]
```

### Options

```
  -h, --help            help for import
      --vpc-id string   The VPC ID to import (required)
```

### Options inherited from parent commands

```
  -o, --output string   Output format (text|table|json) (default "table")
  -v, --version         Print the version number of omnistrate-ctl
```

### SEE ALSO

* [omnistrate-ctl account vpc](omnistrate-ctl_account_vpc.md)	 - Manage VPCs for a Cloud Provider Account

