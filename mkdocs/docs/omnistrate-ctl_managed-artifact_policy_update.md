## omnistrate-ctl managed-artifact policy update

Update a managed artifact release policy

### Synopsis

Update automatic managed artifact release adoption for one Base Amenities environment and cloud provider.

Enabling auto-upgrade clears a stored release pin. Disabling auto-upgrade pins the
release set by --preferred-bundle-version, or the current effective release if the
flag is omitted. Only a release whose private sync is READY can be pinned or
rendered as effective; auto-upgrade follows the newest READY release. The CLI
forwards the requested policy for every supported cloud provider and leaves
authorization and readiness checks to the Omnistrate API.

Requires the root, admin, or editor role.

```
omnistrate-ctl managed-artifact policy update [flags]
```

### Examples

```
omnistrate-ctl managed-artifact policy update --environment-type prod --cloud-provider aws --auto-upgrade=true
omnistrate-ctl managed-artifact policy update --environment-type prod --cloud-provider aws --auto-upgrade=false --preferred-bundle-version r0000020
omnistrate-ctl managed-artifact policy update --environment-type dev --cloud-provider byoc-onprem --auto-upgrade=false --preferred-bundle-version r0000294
```

### Options

```
      --auto-upgrade                      Automatically adopt the newest READY managed artifact release (required)
      --cloud-provider string             Cloud provider (aws, azure, gcp, nebius, oci, or byoc-onprem) (required)
      --environment-type string           Environment type (dev, qa, staging, canary, prod, private, or global) (required)
  -h, --help                              help for update
      --preferred-bundle-version string   READY release to pin when auto-upgrade is disabled
```

### Options inherited from parent commands

```
  -o, --output string   Output format (text|table|json) (default "table")
  -v, --version         Print the version number of omnistrate-ctl
```

### SEE ALSO

* [omnistrate-ctl managed-artifact policy](omnistrate-ctl_managed-artifact_policy.md)	 - Manage Base Amenities release policy

