## omnistrate-ctl managed-artifact release list

List published managed artifact releases

```
omnistrate-ctl managed-artifact release list [flags]
```

### Examples

```
omnistrate-ctl managed-artifact release list --limit 20
omnistrate-ctl managed-artifact release list --bundle-version r0000020 -o json
```

### Options

```
      --bundle-version string    Filter by bundle version, for example r0000020
  -h, --help                     help for list
      --limit int                Maximum number of releases to return (1-100) (default 20)
      --next-page-token string   Opaque token returned by the previous page
      --released-after string    Filter releases at or after this RFC3339 timestamp
      --released-before string   Filter releases before this RFC3339 timestamp
```

### Options inherited from parent commands

```
  -o, --output string   Output format (text|table|json) (default "table")
  -v, --version         Print the version number of omnistrate-ctl
```

### SEE ALSO

* [omnistrate-ctl managed-artifact release](omnistrate-ctl_managed-artifact_release.md)	 - Inspect managed artifact releases

