## omnistrate-ctl account customer list

List customer BYOA account onboarding instances

### Synopsis

This command lists customer BYOA account onboarding instances created through account customer create.

```text
omnistrate-ctl account customer list [flags]
```

### Examples

```text
# List all customer BYOA account onboarding instances
omnistrate-ctl account customer list

# Filter by service and plan
omnistrate-ctl account customer list --service Nebius --plan "Nebius BYOA Compute Variants"

# Filter by cloud provider
omnistrate-ctl account customer list --cloud-provider nebius

# Filter by subscription or customer email
omnistrate-ctl account customer list --subscription-id sub-123456
omnistrate-ctl account customer list --service Nebius --plan "Nebius BYOA Compute Variants" --customer-email customer@example.com
```

### Options

```text
      --cloud-provider string    Filter by cloud provider
      --customer-email string    Filter by customer email; requires both --service and --plan
  -h, --help                     help for list
      --plan string              Filter by service plan name or ID
      --service string           Filter by service name or ID
      --subscription-id string   Filter by subscription ID
```

### Options inherited from parent commands

```text
  -o, --output string   Output format (text|table|json) (default "table")
  -v, --version         Print the version number of omnistrate-ctl
```

### SEE ALSO

- [omnistrate-ctl account customer](../omnistrate-ctl_account_customer/) - Manage customer BYOA account onboarding
