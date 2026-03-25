## omnistrate-ctl workflow events

Get workflow execution status and events

### Synopsis

Get workflow execution status showing resources, steps, and their status.

By default, shows a summary with:
- Resources involved in the workflow
- Workflow steps for each resource
- Status of each step (success, failed, running, etc.)

Use --detail to see full event details for each step. Duplicate events are automatically deduplicated to reduce output size.
Use --max-events to limit the number of unique events shown per event type (default: 3, use 0 for unlimited).

```
omnistrate-ctl workflow events [workflow-id] [flags]
```

### Examples

```
  # Show workflow summary with step statuses
  omnistrate-ctl workflow events <workflow-id> -s <service-id> -e <env-id>

  # Show detailed events for all steps (with deduplication, max 3 per event type)
  omnistrate-ctl workflow events <workflow-id> -s <service-id> -e <env-id> --detail

  # Show detailed events with up to 5 events per event type
  omnistrate-ctl workflow events <workflow-id> -s <service-id> -e <env-id> --detail --max-events 5

  # Show all events without limiting by type
  omnistrate-ctl workflow events <workflow-id> -s <service-id> -e <env-id> --detail --max-events 0

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
      --max-events int          Maximum number of events to show per event type within each step (0 = unlimited) (default 3)
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

