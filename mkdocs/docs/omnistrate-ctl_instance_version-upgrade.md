## omnistrate-ctl instance version-upgrade

Issue a version upgrade for a deployment instance

### Synopsis

This command helps you issue a version upgrade for a deployment instance with the specified upgrade configuration override.

```
omnistrate-ctl instance version-upgrade [instance-id] [flags]
```

### Examples

```
# Issue a version upgrade for an instance with upgrade configuration override to the latest tier version
omnistrate-ctl instance version-upgrade instance-abcd1234 --upgrade-configuration-override /path/to/config.yaml

# Issue a version upgrade to a specific target tier version
omnistrate-ctl instance version-upgrade instance-abcd1234 --upgrade-configuration-override /path/to/config.yaml --target-tier-version 3.0

# Issue a version upgrade and wait for completion with progress tracking
omnistrate-ctl instance version-upgrade instance-abcd1234 --upgrade-configuration-override /path/to/config.yaml --target-tier-version 3.0 --wait

# [HELM ONLY] Use generate-configuration with a target tier version to generate a default deployment instance configuration file based on the current helm values as well as the proposed helm values for the target tier version
omnistrate-ctl instance version-upgrade instance-abcd1234 --existing-configuration existing-config.yaml --proposed-configuration proposed-config.yaml --generate-configuration --target-tier-version 3.0

# Example upgrade configuration override YAML file:
# resource-key-1:
#   helmChartValues:
#     key1: value1
#     key2: value2
# resource-key-2:
#   helmChartValues:
#     database:
#       host: new-host
#       port: 5432
```

### Options

```
      --existing-configuration string           Path to write the existing configuration to (optional, used with --generate-configuration)
      --generate-configuration                  Generate a default configuration file based on current helm values and proposed helm values for the target tier version.This will not perform an upgrade, but will generate a configuration file that can be used for the upgrade.
  -h, --help                                    help for version-upgrade
      --proposed-configuration string           Path to write the proposed configuration to (optional, used with --generate-configuration)
      --target-tier-version string              Target tier version for the version upgrade
      --upgrade-configuration-override string   YAML file containing upgrade configuration override
      --wait                                    Wait for upgrade to complete and show progress
```

### Options inherited from parent commands

```
  -o, --output string   Output format (text|table|json) (default "table")
  -v, --version         Print the version number of omnistrate-ctl
```

### SEE ALSO

* [omnistrate-ctl instance](omnistrate-ctl_instance.md)	 - Manage Instance Deployments for your service

