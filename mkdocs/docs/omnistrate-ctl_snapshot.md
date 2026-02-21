## omnistrate-ctl snapshot

Manage instance snapshots and backups

### Synopsis

This command helps you manage snapshots for your service instances, including creating, copying, listing, describing, deleting, and restoring snapshots.

```
omnistrate-ctl snapshot [operation] [flags]
```

### Options

```
  -h, --help   help for snapshot
```

### Options inherited from parent commands

```
  -o, --output string   Output format (text|table|json) (default "table")
  -v, --version         Print the version number of omnistrate-ctl
```

### SEE ALSO

* [omnistrate-ctl](omnistrate-ctl.md)	 - Manage your Omnistrate SaaS from the command line
* [omnistrate-ctl snapshot copy](omnistrate-ctl_snapshot_copy.md)	 - Copy an instance snapshot to another region
* [omnistrate-ctl snapshot delete](omnistrate-ctl_snapshot_delete.md)	 - Delete an instance snapshot
* [omnistrate-ctl snapshot describe](omnistrate-ctl_snapshot_describe.md)	 - Describe a specific instance snapshot
* [omnistrate-ctl snapshot list](omnistrate-ctl_snapshot_list.md)	 - List all snapshots for an instance
* [omnistrate-ctl snapshot restore](omnistrate-ctl_snapshot_restore.md)	 - Create a new instance by restoring from a snapshot
* [omnistrate-ctl snapshot trigger-backup](omnistrate-ctl_snapshot_trigger-backup.md)	 - Trigger an automatic backup for your instance

