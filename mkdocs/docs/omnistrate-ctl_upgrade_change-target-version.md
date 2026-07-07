## omnistrate-ctl upgrade change-target-version

Change the target version for a scheduled upgrade path

### Synopsis

Change the target product tier version for a scheduled upgrade path before it starts.

```
omnistrate-ctl upgrade change-target-version <upgrade-path-id> [flags]
```

### Examples

```
# Change the target version for a scheduled upgrade path
omnistrate-ctl upgrade change-target-version upgrade-abcd1234 --service-id s-abcd1234 --product-tier-id pt-abcd1234 --target-version 90.0
```

### Options

```
  -h, --help                     help for change-target-version
  -p, --product-tier-id string   Product tier ID (required)
  -s, --service-id string        Service ID (required)
      --target-version string    New target product tier version (required)
```

### Options inherited from parent commands

```
  -o, --output string   Output format (text|table|json) (default "table")
  -v, --version         Print the version number of omnistrate-ctl
```

### SEE ALSO

* [omnistrate-ctl upgrade](omnistrate-ctl_upgrade.md)	 - Upgrade Instance Deployments to a newer or older version

