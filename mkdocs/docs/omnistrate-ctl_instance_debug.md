## omnistrate-ctl instance debug

Debug instance resources

### Synopsis

Debug instance resources with an interactive TUI showing helm charts, terraform files, and logs. Use --output=json for non-interactive JSON output.

```
omnistrate-ctl instance debug [instance-id] [flags]
```

### Examples

```
  omnistrate-ctl instance debug <instance-id>
  omnistrate-ctl instance debug <instance-id> --output=json
```

### Options

```
  -h, --help            help for debug
  -o, --output string   Output format (interactive|json) (default "interactive")
```

### Options inherited from parent commands

```
  -v, --version   Print the version number of omnistrate-ctl
```

### SEE ALSO

* [omnistrate-ctl instance](omnistrate-ctl_instance.md)	 - Manage Instance Deployments for your service

