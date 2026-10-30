## omnistrate-ctl access-request approve

Approve a pending access request

### Synopsis

Approve a pending access request with an optional fixed duration from 1 to 720 minutes.

```
omnistrate-ctl access-request approve [access-request-id] [flags]
```

### Examples

```
# Approve using the backend's default duration
omnistrate-ctl access-request approve ar-12345678

# Approve for two hours
omnistrate-ctl access-request approve ar-12345678 --duration-minutes 120
```

### Options

```
      --duration-minutes int   Access duration in minutes (1-720; omitted uses the backend default)
  -h, --help                   help for approve
```

### Options inherited from parent commands

```
  -o, --output string   Output format (text|table|json) (default "table")
  -v, --version         Print the version number of omnistrate-ctl
```

### SEE ALSO

* [omnistrate-ctl access-request](omnistrate-ctl_access-request.md)	 - Manage deployment-cell access requests

