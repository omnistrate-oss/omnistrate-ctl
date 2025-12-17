## omnistrate-ctl account list

List Cloud Provider Accounts

### Synopsis

This command helps you list Cloud Provider Accounts.
You can filter for specific accounts by using the filter flag.

```
omnistrate-ctl account list [flags]
```

### Examples

```
# List accounts
omnistrate-ctl account list
```

### Options

```
  -f, --filter stringArray   Filter to apply to the list of accounts. E.g.: key1:value1,key2:value2, which filters accounts where key1 equals value1 and key2 equals value2. Allow use of multiple filters to form the logical OR operation. Supported keys: id,name,status,cloud_provider,target_account_id. Check the examples for more details.
  -h, --help                 help for list
```

### Options inherited from parent commands

```
  -o, --output string   Output format (text|table|json) (default "table")
  -v, --version         Print the version number of omnistrate-ctl
```

### SEE ALSO

* [omnistrate-ctl account](omnistrate-ctl_account.md)	 - Manage your Cloud Provider Accounts

