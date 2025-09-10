## omnistrate-ctl deployment-cell generate-config-template

Generate deployment cell configuration template

### Synopsis

Generate a deployment cell configuration template with available amenities for a specific cloud provider.

This command creates a YAML template file containing all available amenities (Helm charts)
that can be configured for deployment cells. The template includes both managed amenities
(maintained by Omnistrate) and custom amenities based on the organization's current configuration.

The generated template can be customized and used with the update-config-template command
to configure deployment cell amenities for your organization.

Examples:

# Generate template for AWS cloud provider

omnistrate-ctl deployment-cell generate-config-template --cloud aws --output template-aws.yaml

# Generate template for Azure cloud provider

omnistrate-ctl deployment-cell generate-config-template --cloud azure --output template-azure.yaml

# Generate template and display to stdout

omnistrate-ctl deployment-cell generate-config-template --cloud aws

```
omnistrate-ctl deployment-cell generate-config-template [flags]
```

### Options

```
  -c, --cloud string    Cloud provider to generate template for (aws,azure,gcp).
  -h, --help            help for generate-config-template
  -o, --output string   Output file path for the template (if not specified, outputs to stdout)
```

### Options inherited from parent commands

```
  -v, --version   Print the version number of omnistrate-ctl
```

### SEE ALSO

- [omnistrate-ctl deployment-cell](omnistrate-ctl_deployment-cell.md) - Manage Deployment Cells
