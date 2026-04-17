## omnistrate-ctl custom-network describe

Describe a custom network

### Synopsis

This command helps you describe an existing custom network.

```text
omnistrate-ctl custom-network describe [flags]
```

### Examples

```text
# Describe a custom network by ID
omnistrate-ctl custom-network describe --custom-network-id [custom-network-id]
```

### Options

```text
      --custom-network-id string   ID of the custom network
  -h, --help                       help for describe
```

### Options inherited from parent commands

```text
  -o, --output string   Output format (text|table|json) (default "table")
  -v, --version         Print the version number of omnistrate-ctl
```

### SEE ALSO

- [omnistrate-ctl custom-network](../omnistrate-ctl_custom-network/) - List and describe custom networks of your customers
