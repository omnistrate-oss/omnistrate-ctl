## omnistrate-ctl custom-network update

Update a custom network

### Synopsis

This command helps you update an existing custom network.

```
omnistrate-ctl custom-network update [flags]
```

### Examples

```
# Update a custom network by ID
omctl custom-network update --custom-network-id [custom-network-id] --name [new-custom-network-name]
```

### Options

```
      --custom-network-id string   ID of the custom network
  -h, --help                       help for update
      --name string                New name of the custom network
```

### Options inherited from parent commands

```
  -o, --output string   Output format (text|table|json) (default "table")
  -v, --version         Print the version number of omnistrate-ctl
```

### SEE ALSO

* [omnistrate-ctl custom-network](omnistrate-ctl_custom-network.md)	 - List and describe custom networks of your customers

