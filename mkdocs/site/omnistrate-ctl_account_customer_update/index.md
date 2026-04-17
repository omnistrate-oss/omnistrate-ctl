## omnistrate-ctl account customer update

Update a customer BYOA account onboarding instance

### Synopsis

This command updates mutable fields on the backing account config associated with a customer BYOA onboarding instance.

```text
omnistrate-ctl account customer update [customer-account-instance-id] [flags]
```

### Examples

```text
# Update the backing account name and description for a customer BYOA onboarding instance
omnistrate-ctl account customer update instance-abcd1234 --name=my-customer-account --description="customer hosted account"

# Replace Nebius bindings on the backing account
omnistrate-ctl account customer update instance-abcd1234 --nebius-bindings-file=./nebius-bindings.yaml
```

### Options

```text
      --description string            Updated backing account description
  -h, --help                          help for update
      --name string                   Updated backing account name
      --nebius-bindings-file string   Path to a YAML file describing the full replacement Nebius bindings
      --skip-wait                     Skip waiting for the backing account to become READY after replacing Nebius bindings
```

### Options inherited from parent commands

```text
  -o, --output string   Output format (text|table|json) (default "table")
  -v, --version         Print the version number of omnistrate-ctl
```

### SEE ALSO

- [omnistrate-ctl account customer](../omnistrate-ctl_account_customer/) - Manage customer BYOA account onboarding
