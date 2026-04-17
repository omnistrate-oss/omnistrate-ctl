## omnistrate-ctl account customer delete

Delete a customer BYOA account onboarding instance

### Synopsis

This command deletes the customer BYOA account onboarding instance for a service plan.

```text
omnistrate-ctl account customer delete [customer-account-instance-id] [flags]
```

### Examples

```text
# Delete a customer BYOA account onboarding instance
omnistrate-ctl account customer delete instance-abcd1234

# Delete and print the deleted instance summary
omnistrate-ctl account customer delete instance-abcd1234 -o json
```

### Options

```text
  -h, --help   help for delete
```

### Options inherited from parent commands

```text
  -o, --output string   Output format (text|table|json) (default "table")
  -v, --version         Print the version number of omnistrate-ctl
```

### SEE ALSO

- [omnistrate-ctl account customer](../omnistrate-ctl_account_customer/) - Manage customer BYOA account onboarding
