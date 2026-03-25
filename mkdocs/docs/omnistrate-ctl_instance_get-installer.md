## omnistrate-ctl instance get-installer

Download the installer for an instance

### Synopsis

This command downloads the installer for an instance and saves it locally.

```
omnistrate-ctl instance get-installer [instance-id] [flags]
```

### Examples

```
# Get the installer for an instance
omnistrate-ctl instance get-installer instance-abcd1234

# Get the installer and save to a specific location
omnistrate-ctl instance get-installer instance-abcd1234 --output-path /tmp/installer.tar.gz
```

### Options

```
  -h, --help                 help for get-installer
  -p, --output-path string   Output path for the installer file (default: ./installer-{instance-id}.tar.gz)
```

### Options inherited from parent commands

```
  -o, --output string   Output format (text|table|json) (default "table")
  -v, --version         Print the version number of omnistrate-ctl
```

### SEE ALSO

* [omnistrate-ctl instance](omnistrate-ctl_instance.md)	 - Manage Instance Deployments for your service

