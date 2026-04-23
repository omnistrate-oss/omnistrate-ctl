## omnistrate-ctl account vpc unimport

Unimport a VPC (revert to AVAILABLE)

### Synopsis

Reverts a previously imported VPC from READY back to AVAILABLE status, removing it from the deployment target pool.

```
omnistrate-ctl account vpc unimport [account-id] --vpc-id=[vpc-id] [flags]
```

### Examples

```
# Unimport a VPC (revert to AVAILABLE)
omnistrate-ctl account vpc unimport [account-id] --vpc-id=[vpc-id]
```

### Options

```
  -h, --help            help for unimport
      --vpc-id string   The VPC ID to unimport (required)
```

### Options inherited from parent commands

```
  -o, --output string   Output format (text|table|json) (default "table")
  -v, --version         Print the version number of omnistrate-ctl
```

### SEE ALSO

* [omnistrate-ctl account vpc](omnistrate-ctl_account_vpc.md)	 - Manage VPCs for a Cloud Provider Account

