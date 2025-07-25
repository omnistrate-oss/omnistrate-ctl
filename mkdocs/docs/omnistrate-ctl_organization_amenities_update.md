## omnistrate-ctl organization amenities update

Update organization amenities configuration template for target environment

### Synopsis

Update the amenities configuration template for a selected target environment.

You specify which environment the update applies to. Updating the environment 
overrides the previous settings for that context.

This action is not versioned—there is only one active configuration per 
environment within the org at any time.

Organization ID is automatically determined from your credentials.

Examples:
  # Update configuration for production environment
  omnistrate-ctl organization amenities update -e production

  # Update with configuration from file
  omnistrate-ctl organization amenities update -e staging -f config.yaml

  # Interactive update
  omnistrate-ctl organization amenities update -e development --interactive

```
omnistrate-ctl organization amenities update [flags]
```

### Options

```
  -f, --config-file string   Path to configuration YAML file (optional)
  -e, --environment string   Target environment (production, staging, development)
  -h, --help                 help for update
      --interactive          Use interactive mode to update amenities configuration
      --merge                Merge with existing configuration instead of replacing
```

### Options inherited from parent commands

```
  -o, --output string   Output format (text|table|json) (default "table")
  -v, --version         Print the version number of omnistrate-ctl
```

### SEE ALSO

* [omnistrate-ctl organization amenities](omnistrate-ctl_organization_amenities.md)	 - Manage organization amenities configuration templates

