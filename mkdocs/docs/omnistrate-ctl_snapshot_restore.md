## omnistrate-ctl snapshot restore

Create a new instance by restoring from a snapshot

### Synopsis

This command helps you create a new instance by restoring from a snapshot.

```
omnistrate-ctl snapshot restore --service-id <service-id> --environment-id <environment-id> --snapshot-id <snapshot-id> [flags]
```

### Examples

```
# Restore to a new instance from a snapshot
omnistrate-ctl snapshot restore --service-id service-abcd --environment-id env-1234 --snapshot-id snapshot-xyz789 --param '{"key": "value"}'

# Restore using parameters from a file
omnistrate-ctl snapshot restore --service-id service-abcd --environment-id env-1234 --snapshot-id snapshot-xyz789 --param-file /path/to/params.json
```

### Options

```
      --environment-id string         The ID of the environment (required)
  -h, --help                          help for restore
      --network-type string           Optional network type change for the instance deployment (PUBLIC / INTERNAL)
      --param string                  Parameters override for the instance deployment
      --param-file string             Json file containing parameters override for the instance deployment
      --service-id string             The ID of the service (required)
      --snapshot-id string            The ID of the snapshot to restore from (required)
      --tierversion-override string   Override the tier version for the restored instance
```

### Options inherited from parent commands

```
  -o, --output string   Output format (text|table|json) (default "table")
  -v, --version         Print the version number of omnistrate-ctl
```

### SEE ALSO

* [omnistrate-ctl snapshot](omnistrate-ctl_snapshot.md)	 - Manage instance snapshots and backups

