## omnistrate-ctl instance copy-snapshot

Copy an instance snapshot to another region

### Synopsis

This command helps you copy an instance snapshot to a different region in the same cloud account for redundancy or disaster recovery. Do not support cross-cloud / cross-account snapshot copies at this time.

```
omnistrate-ctl instance copy-snapshot [instance-id] --snapshot-id <snapshot-id> --target-region <region> [flags]
```

### Examples

```
# Copy a snapshot to another region
omctl instance copy-snapshot instance-abcd1234 --snapshot-id instance-ss-wxyz6789 --target-region us-east1

# Copy the latest snapshot to another region
omctl instance copy-snapshot instance-abcd1234 --target-region us-east1

```

### Options

```
  -h, --help                   help for copy-snapshot
      --snapshot-id string     The ID of the snapshot to copy
      --target-region string   The region to copy the snapshot into
```

### Options inherited from parent commands

```
  -o, --output string   Output format (text|table|json) (default "table")
  -v, --version         Print the version number of omnistrate-ctl
```

### SEE ALSO

* [omnistrate-ctl instance](omnistrate-ctl_instance.md)	 - Manage Instance Deployments for your service

