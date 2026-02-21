## omnistrate-ctl instance create-snapshot

Create a snapshot for an instance

### Synopsis

This command helps you create an on-demand snapshot of your instance. Optionally specify a target region for the snapshot.

```
omnistrate-ctl instance create-snapshot [instance-id] [flags]
```

### Examples

```
# Create a snapshot for an instance
omnistrate-ctl instance create-snapshot instance-abcd1234

# Create a snapshot in a specific region
omnistrate-ctl instance create-snapshot instance-abcd1234 --target-region us-east1

# Create a snapshot with JSON output
omnistrate-ctl instance create-snapshot instance-abcd1234 --output json
```

### Options

```
  -h, --help                   help for create-snapshot
      --target-region string   The target region to create the snapshot in (defaults to the instance region)
```

### Options inherited from parent commands

```
  -o, --output string   Output format (text|table|json) (default "table")
  -v, --version         Print the version number of omnistrate-ctl
```

### SEE ALSO

* [omnistrate-ctl instance](omnistrate-ctl_instance.md)	 - Manage Instance Deployments for your service

