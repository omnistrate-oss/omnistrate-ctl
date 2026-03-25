## omnistrate-ctl instance debug

Visualize the instance plan DAG

### Synopsis

Visualize the plan DAG for an instance based on its product tier version. Use --output=json for non-interactive output.

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
  -h, --help            help for debug
  -o, --output string   Output format (interactive|json) (default "interactive")
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

