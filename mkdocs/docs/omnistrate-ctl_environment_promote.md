## omnistrate-ctl environment promote

Promote a environment

### Synopsis

This command helps you promote a environment in your service.

```
omnistrate-ctl environment promote [service-name] [environment-name] [flags]
```

### Examples

```
# Promote environment
omnistrate-ctl environment promote [service-name] [environment-name]

# Promote environment by ID instead of name
omnistrate-ctl environment promote --service-id=[service-id] --environment-id=[environment-id]

# Promote environment from a specific product tier version
omnistrate-ctl environment promote --service-id=[service-id] --environment-id=[environment-id] --product-tier-id=[product-tier-id] --product-tier-version=[product-tier-version]
```

### Options

```
      --environment-id string         Environment ID. Required if environment name is not provided
  -h, --help                          help for promote
  -p, --product-tier-id string        Product tier ID to promote. Required when product tier version is provided
      --product-tier-version string   Product tier version to promote from. Requires --product-tier-id
      --service-id string             Service ID. Required if service name is not provided
```

### Options inherited from parent commands

```
  -o, --output string   Output format (text|table|json) (default "table")
  -v, --version         Print the version number of omnistrate-ctl
```

### SEE ALSO

* [omnistrate-ctl environment](omnistrate-ctl_environment.md)	 - Manage Service Environments for your service

