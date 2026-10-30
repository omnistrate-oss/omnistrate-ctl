## omnistrate-ctl access-request configure

Configure deployment-cell access approval

### Synopsis

Configure AUTO or MANUAL deployment-cell access approval.

```
omnistrate-ctl access-request configure [flags]
```

### Examples

```
# Require manual approval in PROD
omnistrate-ctl access-request configure --environment-type PROD --approval-mode MANUAL --approver-user-id user-123 --approver-user-id user-456
```

### Options

```
      --approval-mode string       Approval mode (AUTO or MANUAL)
      --approver-user-id strings   Provider user ID allowed to approve, deny, or revoke (repeatable; required for MANUAL)
      --environment-type string    Environment type to configure
  -h, --help                       help for configure
```

### Options inherited from parent commands

```
  -o, --output string   Output format (text|table|json) (default "table")
  -v, --version         Print the version number of omnistrate-ctl
```

### SEE ALSO

* [omnistrate-ctl access-request](omnistrate-ctl_access-request.md)	 - Manage deployment-cell access requests

