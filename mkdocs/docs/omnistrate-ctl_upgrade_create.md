## omnistrate-ctl upgrade create

Create an upgrade path for one or more instances

### Synopsis

This command creates an upgrade path for one or more Instance Deployments to a newer or older version.

```
omnistrate-ctl upgrade create [instance-id] [instance-id] ... --version=[version] [flags]
```

### Examples

```
# Create upgrade for instances to a specific version
omctl upgrade create [instance1] [instance2] --version=2.0

# Create upgrade for instances to the latest version
omctl upgrade create [instance1] [instance2] --version=latest

# Create upgrade for instances to the preferred version
omctl upgrade create [instance1] [instance2] --version=preferred

# Create upgrade for instances to a specific version with version name
omctl upgrade create [instance1] [instance2] --version-name=v0.1.1

# Create upgrade for instance to a specific version with a schedule date in the future
omctl upgrade create [instance-id] --version=1.0 --scheduled-date="2023-12-01T00:00:00Z"

# Create upgrade for instance with limited concurrent upgrades
omctl upgrade create [instance-id] --version=2.0 --max-concurrent-upgrades=5
```

### Options

```
  -h, --help                          help for create
      --max-concurrent-upgrades int   Maximum number of concurrent upgrades (1-25). If 0 or not specified, uses system default.
      --notify-customer               Enable customer notifications for the upgrade
      --scheduled-date string         Specify the scheduled date for the upgrade.
      --version string                Specify the version number to upgrade to. Use 'latest' to upgrade to the latest version. Use 'preferred' to upgrade to the preferred version. Use either this flag or the --version-name flag to upgrade to a specific version.
      --version-name string           Specify the version name to upgrade to. Use either this flag or the --version flag to upgrade to a specific version.
```

### Options inherited from parent commands

```
  -o, --output string   Output format (text|table|json) (default "table")
```

### SEE ALSO

* [omnistrate-ctl upgrade](omnistrate-ctl_upgrade.md)	 - Upgrade Instance Deployments to a newer or older version

