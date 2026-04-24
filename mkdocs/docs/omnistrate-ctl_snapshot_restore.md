## omnistrate-ctl snapshot restore

Restore an instance from a snapshot

### Synopsis

This command helps you restore an instance from a snapshot. By default, a new instance is created. When --restore-to-source is set, the snapshot is restored to the original source instance, preserving its ID and endpoint.

```
omnistrate-ctl snapshot restore --service-id <service-id> --environment-id <environment-id> --snapshot-id <snapshot-id> [--restore-to-source] [flags]
```

### Examples

```
# Restore to a new instance from a snapshot
omnistrate-ctl snapshot restore --service-id service-abcd --environment-id env-1234 --snapshot-id snapshot-xyz789 --param '{"key": "value"}'

# Restore using parameters from a file
omnistrate-ctl snapshot restore --service-id service-abcd --environment-id env-1234 --snapshot-id snapshot-xyz789 --param-file /path/to/params.json

# Restore to the original source instance (preserving its ID and endpoint)
omnistrate-ctl snapshot restore --service-id service-abcd --environment-id env-1234 --snapshot-id snapshot-xyz789 --restore-to-source
```

### Options

```
      --environment-id string         The ID of the environment (required)
  -h, --help                          help for restore
      --network-type string           Optional network type change for the instance deployment (PUBLIC / INTERNAL)
      --param string                  Parameters override for the instance deployment
      --param-file string             Json file containing parameters override for the instance deployment
      --restore-to-source             Restore to the original source instance, preserving its ID and endpoint
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

