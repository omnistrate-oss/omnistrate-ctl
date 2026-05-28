## omnistrate-ctl instance operation trigger

Trigger a supported instance custom operation

```
omnistrate-ctl instance operation trigger [instance-id] [operation-id-or-verb] [flags]
```

### Options

```
      --capacity int                   Capacity to add or remove for ADD_CAPACITY and REMOVE_CAPACITY custom operations
      --failed-replica-action string   Failed replica action for FAILOVER custom operations
      --failed-replica-id string       Failed replica ID for FAILOVER custom operations
  -h, --help                           help for trigger
      --param string                   Parameters for the custom operation
      --param-file string              Json file containing parameters for the custom operation
      --resource-id string             Resource ID to pass to the custom operation. Defaults to the instance root resource ID.
  -y, --yes                            Pre-approve destructive system workflow-backed custom operations without prompting for confirmation
```

### Options inherited from parent commands

```
  -o, --output string   Output format (text|table|json) (default "table")
  -v, --version         Print the version number of omnistrate-ctl
```

### SEE ALSO

* [omnistrate-ctl instance operation](omnistrate-ctl_instance_operation.md)	 - List, describe, and trigger instance custom operations

