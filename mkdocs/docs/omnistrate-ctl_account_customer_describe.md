## omnistrate-ctl account customer describe

Describe a customer account onboarding instance

### Synopsis

This command describes a customer-owned cloud account or BYOC On-Premise onboarding instance and its backing account config.

```
omnistrate-ctl account customer describe [customer-account-instance-id] [flags]
```

### Examples

```
# Describe a customer account onboarding instance
omnistrate-ctl account customer describe instance-abcd1234

# Get full raw details including the backing account config
omnistrate-ctl account customer describe instance-abcd1234 -o json
```

### Options

```
  -h, --help            help for describe
  -o, --output string   Output format (text|table|json). (default "table")
```

### Options inherited from parent commands

```
  -v, --version   Print the version number of omnistrate-ctl
```

### SEE ALSO

* [omnistrate-ctl account customer](omnistrate-ctl_account_customer.md)	 - Manage customer account onboarding

