## omnistrate-ctl access-request config set

Update JIT access approval configuration and display the resulting configuration.

### Synopsis

```
omnistrate-ctl access-request config set [flags]
```

### Examples

```
# Require manual approval using provider-user emails
omnistrate-ctl access-request config set --approval-mode MANUAL --approver-email approver@example.com

# Restore automatic approval
omnistrate-ctl access-request config set --approval-mode AUTO
```

### Options

```
      --approval-mode string      Approval mode (AUTO or MANUAL)
      --approver-email strings    Provider-user email allowed to approve, deny, or revoke (repeatable; required for MANUAL)
      --environment-type string   Environment type to configure (GLOBAL or PROD; defaults from organization configuration)
  -h, --help                      help for set
```

### Options inherited from parent commands

```
  -o, --output string   Output format (text|table|json) (default "table")
  -v, --version         Print the version number of omnistrate-ctl
```

### SEE ALSO

* [omnistrate-ctl access-request config](omnistrate-ctl_access-request_config.md) - View or update JIT access approval configuration
