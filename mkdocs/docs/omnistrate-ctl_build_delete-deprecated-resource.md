## omnistrate-ctl build delete-deprecated-resource

Delete a deprecated resource

### Synopsis

Delete an already deprecated resource from a service. This command does not deprecate resources; it only deletes resources that are ready for deletion.

```
omnistrate-ctl build delete-deprecated-resource --service-id [service-id] --resource-id [resource-id] [flags]
```

### Examples

```
# Validate that a deprecated resource can be deleted
omnistrate-ctl build delete-deprecated-resource --service-id s-123 --resource-id r-123 --dry-run

# Delete a deprecated resource
omnistrate-ctl build delete-deprecated-resource --service-id s-123 --resource-id r-123 --yes
```

### Options

```
      --dry-run              Validate the deprecated resource deletion without deleting it
  -h, --help                 help for delete-deprecated-resource
      --resource-id string   Deprecated resource ID to delete
      --service-id string    Service ID that owns the deprecated resource
  -y, --yes                  Pre-approve deleting the deprecated resource without prompting for confirmation
```

### Options inherited from parent commands

```
  -o, --output string   Output format (text|table|json) (default "table")
  -v, --version         Print the version number of omnistrate-ctl
```

### SEE ALSO

* [omnistrate-ctl build](omnistrate-ctl_build.md)	 - Build Services from image, compose spec or service plan spec

