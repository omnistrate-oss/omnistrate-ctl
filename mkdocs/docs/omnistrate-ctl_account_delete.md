## omnistrate-ctl account delete

Delete a Cloud Provider Account

### Synopsis

This command helps you delete a Cloud Provider Account from your account list.

```
omnistrate-ctl account delete [account-name] [flags]
```

### Examples

```
# Delete account with name
omctl account delete [account-name]

# Delete account with ID
omctl account delete --id=[account-ID]
```

### Options

```
  -h, --help        help for delete
      --id string   Account ID
```

### Options inherited from parent commands

```
  -o, --output string   Output format (text|table|json) (default "table")
  -v, --version         Print the version number of omnistrate-ctl
```

### SEE ALSO

- [omnistrate-ctl account](omnistrate-ctl_account.md) - Manage your Cloud Provider Accounts
