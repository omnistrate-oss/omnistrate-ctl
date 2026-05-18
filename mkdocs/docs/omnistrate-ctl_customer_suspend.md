## omnistrate-ctl customer suspend

Suspend a customer portal user

### Synopsis

This command suspends a customer portal user.

```
omnistrate-ctl customer suspend [flags]
```

### Examples

```
# Suspend a customer portal user
omnistrate-ctl customer suspend --user-id user-123
```

### Options

```
  -h, --help             help for suspend
      --user-id string   Customer user ID
```

### Options inherited from parent commands

```
  -o, --output string   Output format (text|table|json) (default "table")
  -v, --version         Print the version number of omnistrate-ctl
```

### SEE ALSO

* [omnistrate-ctl customer](omnistrate-ctl_customer.md)	 - Manage customer portal users

