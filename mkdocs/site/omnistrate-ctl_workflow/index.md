## omnistrate-ctl workflow

Manage service workflows

### Synopsis

This command helps you manage workflows for your services. You can list, describe, get events, resume, retry, and terminate workflows.

```text
omnistrate-ctl workflow [operation] [flags]
```

### Options

```text
  -h, --help   help for workflow
```

### Options inherited from parent commands

```text
  -o, --output string   Output format (text|table|json) (default "table")
  -v, --version         Print the version number of omnistrate-ctl
```

### SEE ALSO

- [omnistrate-ctl](../omnistrate-ctl/) - Manage your Omnistrate SaaS from the command line
- [omnistrate-ctl workflow describe](../omnistrate-ctl_workflow_describe/) - Describe a specific workflow
- [omnistrate-ctl workflow events](../omnistrate-ctl_workflow_events/) - Get workflow execution status and events
- [omnistrate-ctl workflow list](../omnistrate-ctl_workflow_list/) - List workflows for a service environment
- [omnistrate-ctl workflow resume](../omnistrate-ctl_workflow_resume/) - Resume a paused workflow
- [omnistrate-ctl workflow retry](../omnistrate-ctl_workflow_retry/) - Retry a failed workflow
- [omnistrate-ctl workflow summary](../omnistrate-ctl_workflow_summary/) - Get workflow summary for a service environment
- [omnistrate-ctl workflow terminate](../omnistrate-ctl_workflow_terminate/) - Terminate a running workflow
