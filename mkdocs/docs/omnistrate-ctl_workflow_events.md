## omnistrate-ctl workflow events

Get workflow execution status and events

### Synopsis

Get workflow execution status showing resources, steps, and their status.

By default, shows a summary with:
- Resources involved in the workflow
- Workflow steps for each resource  
- Status of each step (success, failed, running, etc.)

Use --detail to see full event details for each step.

```
omnistrate-ctl workflow events [workflow-id] [flags]
```

### Examples

```
  # Show workflow summary with step statuses
  omnistrate-ctl workflow events <workflow-id> -s <service-id> -e <env-id>

  # Show detailed events for all steps
  omnistrate-ctl workflow events <workflow-id> -s <service-id> -e <env-id> --detail

  # Filter to specific resource and show details
  omnistrate-ctl workflow events <workflow-id> -s <service-id> -e <env-id> --resource-key mydb --detail

  # Show only specific steps
  omnistrate-ctl workflow events <workflow-id> -s <service-id> -e <env-id> --step-names Bootstrap,Deployment

  # Show events from a specific time period
  omnistrate-ctl workflow events <workflow-id> -s <service-id> -e <env-id> --since 2024-01-15T10:00:00Z
```

### Options

```
      --detail                  Show detailed events for each step (default: show step summary with status only)
  -e, --environment-id string   Environment ID (required)
  -h, --help                    help for events
      --resource-id string      Filter to specific resource by ID
      --resource-key string     Filter to specific resource by key/name
  -s, --service-id string       Service ID (required)
      --since string            Show events after this time (RFC3339 format, e.g. 2024-01-15T10:00:00Z)
      --step-names strings      Filter by step names (e.g., Bootstrap, Compute, Deployment, Network, Storage, Monitoring)
      --until string            Show events before this time (RFC3339 format, e.g. 2024-01-15T11:00:00Z)
```

### Options inherited from parent commands

```
  -o, --output string   Output format (text|table|json) (default "table")
  -v, --version         Print the version number of omnistrate-ctl
```

### SEE ALSO

* [omnistrate-ctl workflow](omnistrate-ctl_workflow.md)	 - Manage service workflows

