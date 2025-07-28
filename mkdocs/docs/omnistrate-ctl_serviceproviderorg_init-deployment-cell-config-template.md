## omnistrate-ctl serviceproviderorg init-deployment-cell-config-template

Initialize deployment cell configuration template for service provider organization

### Synopsis

Initialize service provider organization-level deployment cell configuration template through an interactive process.

This command starts an interactive process to define the default organization-level 
configuration template for deployment cells. This step is purely at the service provider org level; 
no reference to any specific service is needed.

The configuration will be stored as a template that can be applied to different 
environments (production, staging, development) and used to synchronize deployment cells.

Organization ID is automatically determined from your credentials.

Examples:
  # Initialize deployment cell configuration template interactively
  omnistrate-ctl serviceproviderorg init-deployment-cell-config-template

  # Initialize from YAML file
  omnistrate-ctl serviceproviderorg init-deployment-cell-config-template -f sample-amenities.yaml

```
omnistrate-ctl serviceproviderorg init-deployment-cell-config-template [flags]
```

### Options

```
  -f, --config-file string   Path to configuration YAML file (optional)
  -h, --help                 help for init-deployment-cell-config-template
      --interactive          Use interactive mode to configure amenities (default true)
```

### Options inherited from parent commands

```
  -o, --output string   Output format (text|table|json) (default "table")
  -v, --version         Print the version number of omnistrate-ctl
```

### SEE ALSO

* [omnistrate-ctl serviceproviderorg](omnistrate-ctl_serviceproviderorg.md)	 - Manage service provider organization configuration

