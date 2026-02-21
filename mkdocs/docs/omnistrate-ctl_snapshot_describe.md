## omnistrate-ctl snapshot describe

Describe a specific instance snapshot

### Synopsis

This command helps you get detailed information about a specific snapshot for your instance.

```
omnistrate-ctl snapshot describe [instance-id] [snapshot-id] [flags]
```

### Examples

```
# Describe a specific snapshot
omnistrate-ctl snapshot describe instance-abcd1234 snapshot-xyz789
```

### Options

```
  -h, --help   help for describe
```

### Options inherited from parent commands

```
  -o, --output string   Output format (text|table|json) (default "table")
  -v, --version         Print the version number of omnistrate-ctl
```

### SEE ALSO

* [omnistrate-ctl snapshot](omnistrate-ctl_snapshot.md)	 - Manage instance snapshots and backups

