## omnistrate-ctl deployment-cell update-config-template

Update deployment cell configuration template

### Synopsis

Update the deployment cell configuration template for your organization or a specific deployment cell.

This command allows you to:
1. Update the organization-level template that applies to new deployment cells
2. Update configuration for a specific deployment cell
3. Sync a deployment cell with the organization template

When updating the organization template, you must specify the environment and cloud provider.
When updating a specific deployment cell, provide the deployment cell ID as an argument or use the --id flag.

Examples:

  # Update organization template for all environments and AWS
  omnistrate-ctl deployment-cell update-config-template - --cloud aws -f template-aws.yaml

  # Update organization template for all environment and AWS
  omnistrate-ctl deployment-cell update-config-template --environment GLOBAL --cloud aws -f template-aws.yaml

  # Update organization template for PROD environment and AWS
  omnistrate-ctl deployment-cell update-config-template --environment PROD --cloud aws -f template-aws.yaml

  # Update specific deployment cell with configuration file using flag
  omnistrate-ctl deployment-cell update-config-template --id hc-12345 -f deployment-cell-config.yaml

  # Sync deployment cell with organization template
  omnistrate-ctl deployment-cell update-config-template --id hc-12345 --sync-with-template

```
omnistrate-ctl deployment-cell update-config-template [flags]
```

### Options

```
  -c, --cloud string         Cloud provider (e.g., aws, azure, gcp) - required for organization template updates
  -e, --environment string   Environment type (e.g., GLOBAL, PROD, STAGING) - optional for organization template update, defaults to GLOBAL
  -f, --file string          Configuration file path (YAML format)
  -h, --help                 help for update-config-template
  -i, --id string            Deployment cell ID
      --sync-with-template   Sync deployment cell with organization template
```

### Options inherited from parent commands

```
  -o, --output string   Output format (text|table|json) (default "table")
  -v, --version         Print the version number of omnistrate-ctl
```

### SEE ALSO

* [omnistrate-ctl deployment-cell](omnistrate-ctl_deployment-cell.md)	 - Manage Deployment Cells

