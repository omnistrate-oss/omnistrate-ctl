## omnistrate-ctl customer verify

Send a customer portal user verification email

### Synopsis

This command sends a verification email to a customer portal user.

```
omnistrate-ctl customer verify [flags]
```

### Examples

```
# Send a verification email to a customer portal user
omnistrate-ctl customer verify --user-id user-123
```

### Options

```
  -h, --help             help for verify
      --user-id string   Customer user ID
```

### Options inherited from parent commands

```
  -o, --output string   Output format (text|table|json) (default "table")
  -v, --version         Print the version number of omnistrate-ctl
```

### SEE ALSO

* [omnistrate-ctl customer](omnistrate-ctl_customer.md)	 - Manage customer portal users

