## omnistrate-ctl upgrade

Upgrade Instance Deployments to a newer or older version

### Synopsis

This command helps you upgrade Instance Deployments to a newer or older version.

```
omnistrate-ctl upgrade --version=[version] [flags]
```

### Examples

```
# Upgrade instances to a specific version
omctl upgrade [instance1] [instance2] --version=2.0

# Upgrade instances to the latest version
omctl upgrade [instance1] [instance2] --version=latest

 # Upgrade instances to the preferred version
omctl upgrade [instance1] [instance2] --version=preferred

# Upgrade instances to a specific version with version name
omctl upgrade [instance1] [instance2] --version-name=v0.1.1

# Upgrade instance to a specific version with a schedule date in the future
omctl upgrade [instance-id] --version=1.0 --scheduled-date="2023-12-01T00:00:00Z"

# Upgrade instance with limited concurrent upgrades
omctl upgrade [instance-id] --version=2.0 --max-concurrent-upgrades=5
```

### Options

```
  -h, --help                          help for upgrade
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

* [omnistrate-ctl](omnistrate-ctl.md)	 - Manage your Omnistrate SaaS from the command line
* [omnistrate-ctl upgrade cancel](omnistrate-ctl_upgrade_cancel.md)	 - Cancel an uncompleted upgrade
* [omnistrate-ctl upgrade describe](omnistrate-ctl_upgrade_describe.md)	 - Describe an upgrade path
* [omnistrate-ctl upgrade list](omnistrate-ctl_upgrade_list.md)	 - List upgrade paths
* [omnistrate-ctl upgrade notify-customer](omnistrate-ctl_upgrade_notify-customer.md)	 - Enable customer notifications for a scheduled upgrade
* [omnistrate-ctl upgrade pause](omnistrate-ctl_upgrade_pause.md)	 - Pause an ongoing upgrade
* [omnistrate-ctl upgrade resume](omnistrate-ctl_upgrade_resume.md)	 - Resume a paused upgrade
* [omnistrate-ctl upgrade skip-instances](omnistrate-ctl_upgrade_skip-instances.md)	 - Skip specific instances from an upgrade path
* [omnistrate-ctl upgrade status](omnistrate-ctl_upgrade_status.md)	 - Get Upgrade status

