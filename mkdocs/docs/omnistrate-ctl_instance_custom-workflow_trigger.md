## omnistrate-ctl instance custom-workflow trigger

Trigger a supported instance custom workflow

```
omnistrate-ctl instance custom-workflow trigger [instance-id] [workflow-id-or-verb] [flags]
```

### Options

```
      --capacity int                   Capacity to add or remove for ADD_CAPACITY and REMOVE_CAPACITY custom workflows
      --failed-replica-action string   Failed replica action for FAILOVER custom workflows
      --failed-replica-id string       Failed replica ID for FAILOVER custom workflows
  -h, --help                           help for trigger
      --param string                   Parameters for the custom workflow
      --param-file string              JSON file containing object parameters for the custom workflow
      --resource-id string             Resource ID to pass to the custom workflow. Defaults to the instance root resource ID.
  -y, --yes                            Pre-approve destructive system workflow-backed custom workflows without prompting for confirmation
```

### Options inherited from parent commands

```
  -o, --output string   Output format (text|table|json) (default "table")
  -v, --version         Print the version number of omnistrate-ctl
```

### SEE ALSO

* [omnistrate-ctl instance custom-workflow](omnistrate-ctl_instance_custom-workflow.md)	 - List, describe, and trigger instance custom workflows

