## omnistrate-ctl customer update

Update a customer portal user

### Synopsis

This command updates customer portal user attributes.

```
omnistrate-ctl customer update [flags]
```

### Examples

```
# Update attributes on a customer portal user
omnistrate-ctl customer update --user-id user-123 --attribute plan=enterprise --attribute region=us-west-2
```

### Options

```
      --attribute stringArray   Customer user attribute in key=value format. Can be repeated or comma-separated
  -h, --help                    help for update
      --user-id string          Customer user ID
```

### Options inherited from parent commands

```
  -o, --output string   Output format (text|table|json) (default "table")
  -v, --version         Print the version number of omnistrate-ctl
```

### SEE ALSO

* [omnistrate-ctl customer](omnistrate-ctl_customer.md)	 - Manage customer portal users

