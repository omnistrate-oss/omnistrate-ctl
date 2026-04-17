## omnistrate-ctl account customer describe

Describe a customer BYOA account onboarding instance

### Synopsis

This command describes a customer BYOA account onboarding instance and its backing account config.

```text
omnistrate-ctl account customer describe [customer-account-instance-id] [flags]
```

### Examples

```text
# Describe a customer BYOA account onboarding instance
omnistrate-ctl account customer describe instance-abcd1234

# Get full raw details including the backing account config
omnistrate-ctl account customer describe instance-abcd1234 -o json
```

### Options

```text
  -h, --help            help for describe
  -o, --output string   Output format (text|table|json). (default "table")
```

### Options inherited from parent commands

```text
  -v, --version   Print the version number of omnistrate-ctl
```

### SEE ALSO

- [omnistrate-ctl account customer](../omnistrate-ctl_account_customer/) - Manage customer BYOA account onboarding
