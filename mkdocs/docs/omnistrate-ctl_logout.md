## omnistrate-ctl logout

Logout and revoke refresh token

### Synopsis

The logout command revokes the stored refresh token on the server
and removes local credentials. This ensures the token cannot be replayed
from another machine.

Use --skip-revoke to only remove local credentials without server-side
revocation (legacy behavior).

```
omnistrate-ctl logout [flags]
```

### Examples

```
omnistrate-ctl logout
```

### Options

```
  -h, --help          help for logout
      --skip-revoke   Skip server-side token revocation; only remove local credentials
```

### Options inherited from parent commands

```
  -o, --output string   Output format (text|table|json) (default "table")
  -v, --version         Print the version number of omnistrate-ctl
```

### SEE ALSO

* [omnistrate-ctl](omnistrate-ctl.md)	 - Manage your Omnistrate SaaS from the command line

