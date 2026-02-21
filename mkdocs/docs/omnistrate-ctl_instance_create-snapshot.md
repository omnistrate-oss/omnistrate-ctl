## omnistrate-ctl instance create-snapshot

Create a snapshot for an instance

### Synopsis

This command helps you create an on-demand snapshot of your instance.

By default it creates a snapshot from the current instance state. Optionally specify
a target region for the snapshot.

When --source-snapshot-id is provided, the snapshot is created by copying the specified
source snapshot to the target region. In this mode --target-region is required.

```
omnistrate-ctl instance create-snapshot [instance-id] [flags]
```

### Examples

```
# Create a snapshot from the current instance state
omnistrate-ctl instance create-snapshot instance-abcd1234

# Create a snapshot in a specific region
omnistrate-ctl instance create-snapshot instance-abcd1234 --target-region us-east1

# Create a snapshot from another snapshot (copies to a target region)
omnistrate-ctl instance create-snapshot instance-abcd1234 --source-snapshot-id instance-ss-wxyz6789 --target-region us-east1

# Create a snapshot with JSON output
omnistrate-ctl instance create-snapshot instance-abcd1234 --output json
```

### Options

```
  -h, --help                        help for create-snapshot
      --source-snapshot-id string   Source snapshot ID to create the new snapshot from (uses the copy API; requires --target-region)
      --target-region string        The target region to create the snapshot in (defaults to the instance region)
```

### Options inherited from parent commands

```
  -o, --output string   Output format (text|table|json) (default "table")
  -v, --version         Print the version number of omnistrate-ctl
```

### SEE ALSO

* [omnistrate-ctl instance](omnistrate-ctl_instance.md)	 - Manage Instance Deployments for your service

