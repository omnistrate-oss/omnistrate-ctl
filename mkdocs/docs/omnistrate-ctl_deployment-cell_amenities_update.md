## omnistrate-ctl deployment-cell amenities update

Update organization amenities configuration for target environment

### Synopsis

Update the amenities configuration template for a selected target environment.

You specify which environment the update applies to. Updating the environment 
overwrites the previous settings for that context.

This action is not versioned—there is only one active configuration per 
environment within the org at any time.

Examples:
  # Update configuration for production environment
  omnistrate-ctl deployment-cell amenities update -g org-123 -e production

  # Update with configuration from file
  omnistrate-ctl deployment-cell amenities update -g org-123 -e staging -f config.json

  # Interactive update
  omnistrate-ctl deployment-cell amenities update -g org-123 -e development --interactive

```
omnistrate-ctl deployment-cell amenities update [flags]
```

### Options

```
  -f, --config-file string       Path to configuration JSON file (optional)
  -e, --environment string       Target environment (production, staging, development)
  -h, --help                     help for update
      --interactive              Use interactive mode to update amenities configuration
      --merge                    Merge with existing configuration instead of replacing
  -g, --organization-id string   Organization ID (required)
```

### Options inherited from parent commands

```
  -o, --output string   Output format (text|table|json) (default "table")
  -v, --version         Print the version number of omnistrate-ctl
```

### SEE ALSO

* [omnistrate-ctl deployment-cell amenities](omnistrate-ctl_deployment-cell_amenities.md)	 - Manage deployment cell amenities configuration

