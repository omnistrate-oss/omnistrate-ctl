## omnistrate-ctl serviceproviderorg update

Update service provider organization configuration template for target environment

### Synopsis

Update the service provider organization configuration template for a selected target environment.

You specify which environment the update applies to and provide a configuration file.
Updating the environment overrides the previous settings for that context.

This action is not versioned—there is only one active configuration per 
environment within the service provider org at any time.

Organization ID is automatically determined from your credentials.

Examples:
  # Update configuration for production environment
  omnistrate-ctl serviceproviderorg update -e PROD -f config-template.yaml

  # Update staging environment with configuration file
  omnistrate-ctl serviceproviderorg update -e STAGING -f config-template.yaml

```
omnistrate-ctl serviceproviderorg update [flags]
```

### Options

```
  -f, --config-file string   Path to configuration YAML file
  -e, --environment string   Target environment (PROD, PRIVATE, CANARY, STAGING, QA, DEV)
  -h, --help                 help for update
```

### Options inherited from parent commands

```
  -o, --output string   Output format (text|table|json) (default "table")
  -v, --version         Print the version number of omnistrate-ctl
```

### SEE ALSO

* [omnistrate-ctl serviceproviderorg](omnistrate-ctl_serviceproviderorg.md)	 - Manage service provider organization configuration

