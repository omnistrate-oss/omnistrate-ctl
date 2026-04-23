## omnistrate-ctl account vpc bulk-import

Import multiple VPCs at once

### Synopsis

Imports multiple discovered VPCs in a single operation, changing their status from AVAILABLE to READY.

```
omnistrate-ctl account vpc bulk-import [account-id] --vpc-ids=[vpc-id1,vpc-id2,...] [flags]
```

### Examples

```
# Bulk import multiple VPCs
omnistrate-ctl account vpc bulk-import [account-id] --vpc-ids=vpc-abc123,vpc-def456
```

### Options

```
  -h, --help             help for bulk-import
      --vpc-ids string   Comma-separated list of VPC IDs to import (required)
```

### Options inherited from parent commands

```
  -o, --output string   Output format (text|table|json) (default "table")
  -v, --version         Print the version number of omnistrate-ctl
```

### SEE ALSO

* [omnistrate-ctl account vpc](omnistrate-ctl_account_vpc.md)	 - Manage VPCs for a Cloud Provider Account

