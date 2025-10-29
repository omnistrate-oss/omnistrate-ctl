## omnistrate-ctl deployment-cell describe-config-template

Describe deployment cell configuration template

### Synopsis

Describe the current deployment cell configuration template for your organization.

This command shows the current amenities configuration template that is applied to 
new deployment cells in the specified environment and cloud provider.

You can also describe the configuration of a specific deployment cell by providing 
its ID as an argument.

Examples:
  # Describe organization template for all environments and AWS
  omnistrate-ctl deployment-cell describe-config-template --cloud aws

  # Describe organization template for all environments and AWS
  omnistrate-ctl deployment-cell describe-config-template --cloud aws

  # Describe organization template for PROD environment and AWS
  omnistrate-ctl deployment-cell describe-config-template --environment PROD --cloud aws

  # Describe specific deployment cell configuration
  omnistrate-ctl deployment-cell describe-config-template --id hc-12345

  # Get JSON output format
  omnistrate-ctl deployment-cell describe-config-template --cloud aws --output json

  # Generate YAML template to local file
  omnistrate-ctl deployment-cell describe-config-template --cloud aws --output-file template.yaml

  # Generate template for specific deployment cell to file
  omnistrate-ctl deployment-cell describe-config-template --id hc-12345 --output-file deployment-cell-config.yaml

```
omnistrate-ctl deployment-cell describe-config-template [flags]
```

### Options

```
  -c, --cloud string         Cloud provider (e.g., aws, azure, gcp) - required for organization template updates
  -e, --environment string   Environment type (e.g., GLOBAL, PROD, STAGING) - optional for organization template update, defaults to GLOBAL
  -h, --help                 help for describe-config-template
  -i, --id string            Deployment cell ID
  -o, --output string        Output format (yaml, json, table) (default "yaml")
      --output-file string   Output template to file (YAML format)
```

### Options inherited from parent commands

```
  -v, --version   Print the version number of omnistrate-ctl
```

### SEE ALSO

* [omnistrate-ctl deployment-cell](omnistrate-ctl_deployment-cell.md)	 - Manage Deployment Cells

