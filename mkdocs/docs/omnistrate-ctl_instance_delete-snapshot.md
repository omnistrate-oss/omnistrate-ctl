## omnistrate-ctl instance delete-snapshot

Delete an instance snapshot

### Synopsis

This command helps you delete a specific snapshot for your instance.

```
omnistrate-ctl instance delete-snapshot [instance-id] --snapshot-id <snapshot-id> [flags]
```

### Examples

```
# Delete a specific snapshot
omnistrate-ctl instance delete-snapshot instance-abcd1234 --snapshot-id snapshot-xyz789
```

### Options

```
  -h, --help                 help for delete-snapshot
      --snapshot-id string   The ID of the snapshot to delete (required)
```

### Options inherited from parent commands

```
  -o, --output string   Output format (text|table|json) (default "table")
  -v, --version         Print the version number of omnistrate-ctl
```

### SEE ALSO

* [omnistrate-ctl instance](omnistrate-ctl_instance.md)	 - Manage Instance Deployments for your service

