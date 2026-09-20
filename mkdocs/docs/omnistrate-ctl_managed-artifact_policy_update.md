## omnistrate-ctl managed-artifact policy update

Update a managed artifact release policy

### Synopsis

Update automatic managed artifact release adoption for one Base Amenities environment and cloud provider.

Enabling auto-upgrade clears a stored release pin. Disabling auto-upgrade without
--preferred-bundle-version freezes the current effective release.

```
omnistrate-ctl managed-artifact policy update [flags]
```

### Examples

```
omnistrate-ctl managed-artifact policy update --environment-type prod --cloud-provider aws --auto-upgrade=true
omnistrate-ctl managed-artifact policy update --environment-type prod --cloud-provider aws --auto-upgrade=false --preferred-bundle-version r0000020
```

### Options

```
      --auto-upgrade                      Automatically adopt the newest published managed artifact release (required) (default true)
      --cloud-provider string             Cloud provider (aws, azure, gcp, nebius, oci, byoc-onprem, or all) (required)
      --environment-type string           Environment type (dev, qa, staging, canary, prod, private, or global) (required)
  -h, --help                              help for update
      --preferred-bundle-version string   Release to pin when auto-upgrade is disabled
```

### Options inherited from parent commands

```
  -o, --output string   Output format (text|table|json) (default "table")
  -v, --version         Print the version number of omnistrate-ctl
```

### SEE ALSO

* [omnistrate-ctl managed-artifact policy](omnistrate-ctl_managed-artifact_policy.md)	 - Manage Base Amenities release policy

