## omnistrate-ctl account vpc list

List VPCs for a Cloud Provider Account

### Synopsis

Lists all VPCs registered with a BYOA cloud provider account, including their status (AVAILABLE, READY, SYNCING, etc).

```
omnistrate-ctl account vpc list [account-id] [flags]
```

### Examples

```
# List all VPCs for an account
omnistrate-ctl account vpc list [account-id]
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

* [omnistrate-ctl account vpc](omnistrate-ctl_account_vpc.md)	 - Manage VPCs for a Cloud Provider Account

