## omnistrate-ctl snapshot copy

Copy an instance snapshot to another region

### Synopsis

This command helps you copy an instance snapshot to a different region in the same cloud account for redundancy or disaster recovery. Does not support cross-cloud / cross-account snapshot copies at this time.

```
omnistrate-ctl snapshot copy [instance-id] --target-region <region> [flags]
```

### Examples

```
# Copy a snapshot to another region
omnistrate-ctl snapshot copy instance-abcd1234 --snapshot-id instance-ss-wxyz6789 --target-region us-east1

# Copy the latest snapshot to another region
omnistrate-ctl snapshot copy instance-abcd1234 --target-region us-east1

```

### Options

```
  -h, --help                   help for copy
      --snapshot-id string     The ID of the snapshot or backup to copy. If not provided, the latest snapshot will be used.
      --target-region string   The target region to copy the snapshot into (required)
```

### Options inherited from parent commands

```
  -o, --output string   Output format (text|table|json) (default "table")
  -v, --version         Print the version number of omnistrate-ctl
```

### SEE ALSO

* [omnistrate-ctl snapshot](omnistrate-ctl_snapshot.md)	 - Manage instance snapshots and backups

