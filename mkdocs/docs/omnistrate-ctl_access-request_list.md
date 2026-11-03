## omnistrate-ctl access-request list

List access requests

### Synopsis

List access requests currently exposed by the backend to the authenticated service-provider organization. Revoked-request visibility is controlled by backend retention policy.

```
omnistrate-ctl access-request list [flags]
```

### Options

```
  -h, --help             help for list
      --status strings   Filter backend-visible requests by lifecycle status (comma-separated or repeated)
```

### Options inherited from parent commands

```
  -o, --output string   Output format (text|table|json) (default "table")
  -v, --version         Print the version number of omnistrate-ctl
```

### SEE ALSO

* [omnistrate-ctl access-request](omnistrate-ctl_access-request.md)	 - Manage deployment-cell access requests

