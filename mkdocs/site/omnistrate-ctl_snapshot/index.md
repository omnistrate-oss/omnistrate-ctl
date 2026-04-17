## omnistrate-ctl snapshot

Manage instance snapshots and backups

### Synopsis

This command helps you manage snapshots for your service instances, including creating, copying, listing, describing, deleting, and restoring snapshots.

```text
omnistrate-ctl snapshot [operation] [flags]
```

### Options

```text
  -h, --help   help for snapshot
```

### Options inherited from parent commands

```text
  -o, --output string   Output format (text|table|json) (default "table")
  -v, --version         Print the version number of omnistrate-ctl
```

### SEE ALSO

- [omnistrate-ctl](../omnistrate-ctl/) - Manage your Omnistrate SaaS from the command line
- [omnistrate-ctl snapshot delete](../omnistrate-ctl_snapshot_delete/) - Delete an instance snapshot
- [omnistrate-ctl snapshot list](../omnistrate-ctl_snapshot_list/) - List all snapshots for a service environment
- [omnistrate-ctl snapshot restore](../omnistrate-ctl_snapshot_restore/) - Create a new instance by restoring from a snapshot
