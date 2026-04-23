## omnistrate-ctl instance restore

Restore an instance from a snapshot

### Synopsis

This command helps you restore an instance from a snapshot. By default, a new instance is created. When --restore-to-source is set, the snapshot is restored to the original source instance, preserving its ID and endpoint.

```
omnistrate-ctl instance restore [instance-id] --snapshot-id <snapshot-id> [--param=param] [--param-file=file-path] [--restore-to-source] [--tierversion-override <tier-version>] [--network-type PUBLIC / INTERNAL] [flags]
```

### Examples

```
# Restore to a new instance from a snapshot
omnistrate-ctl instance restore instance-abcd1234 --snapshot-id snapshot-xyz789 --param '{"key": "value"}'

# Restore using parameters from a file
omnistrate-ctl instance restore instance-abcd1234 --snapshot-id snapshot-xyz789 --param-file /path/to/params.json

# Restore to the original source instance (preserving its ID and endpoint)
omnistrate-ctl instance restore instance-abcd1234 --snapshot-id snapshot-xyz789 --restore-to-source
```

### Options

```
  -h, --help                          help for restore
      --network-type string           Optional network type change for the instance deployment (PUBLIC / INTERNAL)
      --param string                  Parameters override for the instance deployment
      --param-file string             Json file containing parameters override for the instance deployment
      --restore-to-source             Restore to the original source instance, preserving its ID and endpoint
      --snapshot-id string            The ID of the snapshot to restore from
      --tierversion-override string   Override the tier version for the restored instance
```

### Options inherited from parent commands

```
  -o, --output string   Output format (text|table|json) (default "table")
  -v, --version         Print the version number of omnistrate-ctl
```

### SEE ALSO

* [omnistrate-ctl instance](omnistrate-ctl_instance.md)	 - Manage Instance Deployments for your service

