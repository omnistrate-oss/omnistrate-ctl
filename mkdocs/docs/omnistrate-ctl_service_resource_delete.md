## omnistrate-ctl service resource delete

Delete a service resource

### Synopsis

This command helps you delete a resource from a service.

```
omnistrate-ctl service resource delete --service-id [service-ID] --id [resource-ID] [flags]
```

### Examples

```
# Delete a resource with service and resource IDs
omnistrate-ctl service resource delete --service-id=[service-ID] --id=[resource-ID]

# Delete a resource using the service ID as a path segment
omnistrate-ctl service [service-ID] resource delete --id=[resource-ID]

# Dry run resource deletion
omnistrate-ctl service [service-ID] resource delete --id=[resource-ID] --dry-run
```

### Options

```
      --dry-run             Validate resource deletion without deleting it
  -h, --help                help for delete
      --id string           Resource ID
      --service-id string   Service ID
```

### Options inherited from parent commands

```
  -o, --output string   Output format (text|table|json) (default "table")
  -v, --version         Print the version number of omnistrate-ctl
```

### SEE ALSO

* [omnistrate-ctl service resource](omnistrate-ctl_service_resource.md)	 - Manage service resources

