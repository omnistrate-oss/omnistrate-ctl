## omnistrate-ctl managed-artifact policy describe

Describe a managed artifact release policy

```
omnistrate-ctl managed-artifact policy describe [flags]
```

### Examples

```
omnistrate-ctl managed-artifact policy describe --environment-type prod --cloud-provider aws
```

### Options

```
      --cloud-provider string     Cloud provider (aws, azure, gcp, nebius, oci, byoc-onprem, or all) (required)
      --environment-type string   Environment type (dev, qa, staging, canary, prod, private, or global) (required)
  -h, --help                      help for describe
```

### Options inherited from parent commands

```
  -o, --output string   Output format (text|table|json) (default "table")
  -v, --version         Print the version number of omnistrate-ctl
```

### SEE ALSO

* [omnistrate-ctl managed-artifact policy](omnistrate-ctl_managed-artifact_policy.md)	 - Manage Base Amenities release policy

