## omnistrate-ctl custom-network delete

Deletes a custom network

### Synopsis

This command helps you delete an existing custom network.

```
omnistrate-ctl custom-network delete [custom-network-name] [flags]
```

### Examples

```
# Delete a custom network by ID
omnistrate-ctl custom-network delete --custom-network-id [custom-network-id]
```

### Options

```
      --custom-network-id string   ID of the custom network
  -h, --help                       help for delete
```

### Options inherited from parent commands

```
  -o, --output string   Output format (text|table|json) (default "table")
  -v, --version         Print the version number of omnistrate-ctl
```

### SEE ALSO

* [omnistrate-ctl custom-network](omnistrate-ctl_custom-network.md)	 - List and describe custom networks of your customers

