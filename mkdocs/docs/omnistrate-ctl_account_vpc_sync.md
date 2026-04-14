## omnistrate-ctl account vpc sync

Discover and sync VPCs from the cloud provider

### Synopsis

Triggers VPC discovery for a BYOA cloud provider account. Discovered VPCs will appear with AVAILABLE status and can then be imported.

```
omnistrate-ctl account vpc sync [account-id] [flags]
```

### Examples

```
# Sync VPCs for an account
omnistrate-ctl account vpc sync [account-id]
```

### Options

```
  -h, --help   help for sync
```

### Options inherited from parent commands

```
  -o, --output string   Output format (text|table|json) (default "table")
  -v, --version         Print the version number of omnistrate-ctl
```

### SEE ALSO

* [omnistrate-ctl account vpc](omnistrate-ctl_account_vpc.md)	 - Manage VPCs for a Cloud Provider Account

