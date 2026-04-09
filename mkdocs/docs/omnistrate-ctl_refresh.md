## omnistrate-ctl refresh

Refresh the access token using the stored refresh token

### Synopsis

The refresh command exchanges the stored refresh token for a new
JWT access token without requiring the user to re-enter credentials.

This is useful for testing the token refresh flow end-to-end and for
scripting scenarios where a fresh token is needed.

```
omnistrate-ctl refresh [flags]
```

### Examples

```
omnistrate-ctl refresh
```

### Options

```
  -h, --help   help for refresh
```

### Options inherited from parent commands

```
  -o, --output string   Output format (text|table|json) (default "table")
  -v, --version         Print the version number of omnistrate-ctl
```

### SEE ALSO

* [omnistrate-ctl](omnistrate-ctl.md)	 - Manage your Omnistrate SaaS from the command line

