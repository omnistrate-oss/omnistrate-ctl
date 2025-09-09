## omnistrate-ctl workflow terminate

Terminate a running workflow

### Synopsis

Terminate a running workflow. This will stop the workflow execution.

```
omnistrate-ctl workflow terminate [workflow-id] [flags]
```

### Options

```
  -y, --confirm                 Skip confirmation prompt
  -e, --environment-id string   Environment ID (required)
  -h, --help                    help for terminate
  -s, --service-id string       Service ID (required)
```

### Options inherited from parent commands

```
  -o, --output string   Output format (text|table|json) (default "table")
  -v, --version         Print the version number of omnistrate-ctl
```

### SEE ALSO

- [omnistrate-ctl workflow](omnistrate-ctl_workflow.md) - Manage service workflows
