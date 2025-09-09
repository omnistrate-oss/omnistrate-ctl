## omnistrate-ctl account describe

Describe a Cloud Provider Account

### Synopsis

This command helps you get details of a cloud provider account.

```
omnistrate-ctl account describe [account-name] [flags]
```

### Examples

```
# Describe account with name
omctl account describe [account-name]

# Describe account with ID
omctl account describe --id=[account-id]
```

### Options

```
  -h, --help            help for describe
      --id string       Account ID
  -o, --output string   Output format. Only json is supported. (default "json")
```

### Options inherited from parent commands

```
  -v, --version   Print the version number of omnistrate-ctl
```

### SEE ALSO

* [omnistrate-ctl account](omnistrate-ctl_account.md)	 - Manage your Cloud Provider Accounts

