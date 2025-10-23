## omnistrate-ctl instance debug

Debug instance resources

### Synopsis

Debug instance resources with an interactive TUI showing helm charts, terraform files, and logs. Use --output=json for non-interactive JSON output.

```
omnistrate-ctl instance debug [instance-id] [flags]
```

### Examples

```
  omnistrate-ctl instance debug <instance-id>
  omnistrate-ctl instance debug <instance-id> --output=json
```

### Options

```
  -h, --help                  help for debug
  -o, --output string         Output format (interactive|json) (default "interactive")
      --resource-id string    Filter results by resource ID
      --resource-key string   Filter results by resource key
```

### Options inherited from parent commands

```
  -v, --version   Print the version number of omnistrate-ctl
```

### SEE ALSO

* [omnistrate-ctl instance](omnistrate-ctl_instance.md)	 - Manage Instance Deployments for your service
* [omnistrate-ctl instance debug helm-logs](omnistrate-ctl_instance_debug_helm-logs.md)	 - Get Helm installation logs for instance resources
* [omnistrate-ctl instance debug helm-values](omnistrate-ctl_instance_debug_helm-values.md)	 - Get Helm chart values for instance resources
* [omnistrate-ctl instance debug terraform-files](omnistrate-ctl_instance_debug_terraform-files.md)	 - Get Terraform files for instance resources
* [omnistrate-ctl instance debug terraform-output](omnistrate-ctl_instance_debug_terraform-output.md)	 - Get Terraform logs for instance resources

