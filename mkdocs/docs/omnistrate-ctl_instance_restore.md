## omnistrate-ctl instance restore

Create a new instance by restoring from a snapshot

### Synopsis

This command helps you create a new instance by restoring from a snapshot using an existing instance for context.

```
omnistrate-ctl instance restore [instance-id] --snapshot-id <snapshot-id> [--param=param] [--param-file=file-path] --tierversion-override <tier-version> --network-type PUBLIC / INTERNAL [flags]
```

### Examples

```
# Restore to a new instance from a snapshot
omctl instance restore instance-abcd1234 --snapshot-id snapshot-xyz789 --param '{"key": "value"}'

# Restore using parameters from a file
omctl instance restore instance-abcd1234 --snapshot-id snapshot-xyz789 --param-file /path/to/params.json
```

### Options

```
  -h, --help                          help for restore
      --network-type string           Optional network type change for the instance deployment (PUBLIC / INTERNAL)
      --param string                  Parameters override for the instance deployment
      --param-file string             Json file containing parameters override for the instance deployment
      --snapshot-id string            The ID of the snapshot to restore from
      --tierversion-override string   Override the tier version for the restored instance
```

### Options inherited from parent commands

```
  -o, --output string   Output format (text|table|json) (default "table")
  -v, --version         Print the version number of omnistrate-ctl
```

### SEE ALSO

- [omnistrate-ctl instance](omnistrate-ctl_instance.md) - Manage Instance Deployments for your service
