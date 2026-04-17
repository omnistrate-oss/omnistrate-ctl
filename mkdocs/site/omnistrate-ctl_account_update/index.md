## omnistrate-ctl account update

Update a Cloud Provider Account

### Synopsis

This command helps you update mutable fields on an existing cloud provider account.

```text
omnistrate-ctl account update [account-name or account-id] [flags]
```

### Examples

```text
# Update account name and description
omnistrate-ctl account update [account-name or account-id] --name=[new-name] --description=[new-description]

# Replace Nebius bindings on an existing Nebius account
omnistrate-ctl account update [account-name or account-id] --nebius-bindings-file=[bindings-file]
```

### Options

```text
      --description string            Updated account description
  -h, --help                          help for update
      --name string                   Updated account name
      --nebius-bindings-file string   Path to a YAML file describing the full replacement Nebius bindings
      --skip-wait                     Skip waiting for account to become READY after replacing Nebius bindings
```

### Options inherited from parent commands

```text
  -o, --output string   Output format (text|table|json) (default "table")
  -v, --version         Print the version number of omnistrate-ctl
```

### SEE ALSO

- [omnistrate-ctl account](../omnistrate-ctl_account/) - Manage your Cloud Provider Accounts
