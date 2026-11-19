## omnistrate-ctl account update

Update a Cloud Provider Account

### Synopsis

This command helps you update mutable fields on an existing cloud provider account.

```
omnistrate-ctl account update [account-name or account-id] [flags]
```

### Examples

```
# Update account name and description
omnistrate-ctl account update [account-name or account-id] --name=[new-name] --description=[new-description]

# Replace Nebius bindings on an existing Nebius account
omnistrate-ctl account update [account-name or account-id] --nebius-bindings-file=[bindings-file]

# Enable private managed-artifact publication for an AWS non-BYOA account
omnistrate-ctl account update [account-name or account-id] --private-artifact-publication-enabled=true

# Disable private managed-artifact publication (use = with false)
omnistrate-ctl account update [account-name or account-id] --private-artifact-publication-enabled=false
```

### Options

```
      --description string                     Updated account description
  -h, --help                                   help for update
      --name string                            Updated account name
      --nebius-bindings-file string            Path to a YAML file describing the full replacement Nebius bindings
      --private-artifact-publication-enabled   Whether an AWS non-BYOA account may publish managed artifacts to private ECR
      --skip-wait                              Skip waiting for account to become READY after replacing Nebius bindings
      --tags string                            Full replacement set of custom tags for the account in key=value format, separated by commas (e.g. env=prod,team=platform). Escape commas inside values with \,
```

### Options inherited from parent commands

```
  -o, --output string   Output format (text|table|json) (default "table")
  -v, --version         Print the version number of omnistrate-ctl
```

### SEE ALSO

* [omnistrate-ctl account](omnistrate-ctl_account.md)	 - Manage your Cloud Provider Accounts

