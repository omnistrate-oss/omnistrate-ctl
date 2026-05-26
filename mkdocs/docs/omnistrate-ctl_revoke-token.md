## omnistrate-ctl revoke-token

Revoke the stored refresh token

### Synopsis

The revoke-token command invalidates the stored refresh token on the
server and removes local credentials. After revocation the token can
never be used to obtain a new access token.

This is stronger than "logout" which only removes local credentials:
revoke-token also tells the server to delete the refresh token so it
cannot be replayed from another machine.

```
omnistrate-ctl revoke-token [flags]
```

### Examples

```
omnistrate-ctl revoke-token
```

### Options

```
  -h, --help   help for revoke-token
```

### Options inherited from parent commands

```
  -o, --output string   Output format (text|table|json) (default "table")
  -v, --version         Print the version number of omnistrate-ctl
```

### SEE ALSO

* [omnistrate-ctl](omnistrate-ctl.md)	 - Manage your Omnistrate SaaS from the command line

