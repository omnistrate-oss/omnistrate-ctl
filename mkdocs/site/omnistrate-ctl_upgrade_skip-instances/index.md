## omnistrate-ctl upgrade skip-instances

Skip specific instances from an upgrade path

```text
omnistrate-ctl upgrade skip-instances [upgrade-id] [flags]
```

### Examples

```text
 Skip specific instances from an upgrade path #
omnistrate-ctl upgrade skip-instances [upgrade-id] --resource-ids instance-1,instance-2
```

### Options

```text
  -h, --help                  help for skip-instances
      --resource-ids string   Comma-separated list of instance IDs to skip
```

### Options inherited from parent commands

```text
  -o, --output string   Output format (text|table|json) (default "table")
  -v, --version         Print the version number of omnistrate-ctl
```

### SEE ALSO

- [omnistrate-ctl upgrade](../omnistrate-ctl_upgrade/) - Upgrade Instance Deployments to a newer or older version
