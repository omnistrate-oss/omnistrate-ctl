## omnistrate-ctl serviceproviderorg update-amenities

Update service provider organization amenities configuration template for target environment

### Synopsis

Update the service provider organization amenities configuration template for a selected target environment.

You specify which environment the update applies to. Updating the environment 
overrides the previous settings for that context.

This action is not versioned—there is only one active configuration per 
environment within the service provider org at any time.

Organization ID is automatically determined from your credentials.

Examples:
  # Update configuration for production environment
  omnistrate-ctl serviceproviderorg update-amenities -e production

  # Update with configuration from file
  omnistrate-ctl serviceproviderorg update-amenities -e staging -f sample-amenities.yaml

  # Interactive update
  omnistrate-ctl serviceproviderorg update-amenities -e development --interactive

```
omnistrate-ctl serviceproviderorg update-amenities [flags]
```

### Options

```
  -f, --config-file string   Path to configuration YAML file (optional)
  -e, --environment string   Target environment (production, staging, development)
  -h, --help                 help for update-amenities
      --interactive          Use interactive mode to update amenities configuration
      --merge                Merge with existing configuration instead of replacing
```

### Options inherited from parent commands

```
  -o, --output string   Output format (text|table|json) (default "table")
  -v, --version         Print the version number of omnistrate-ctl
```

### SEE ALSO

* [omnistrate-ctl serviceproviderorg](omnistrate-ctl_serviceproviderorg.md)	 - Manage service provider organization configuration

