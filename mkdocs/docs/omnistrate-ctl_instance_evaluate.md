## omnistrate-ctl instance evaluate

Evaluate an expression in the context of an instance

### Synopsis

This command helps you evaluate expressions using instance parameters and system variables.

```
omnistrate-ctl instance evaluate [instance-id] [resource-key] [flags]
```

### Examples

```
# Evaluate an expression for an instance
omctl instance evaluate instance-abcd1234 my-resource-key --expression "$var.username + {{ $sys.id }}"

# Evaluate expressions from a JSON file
omctl instance evaluate instance-abcd1234 my-resource-key --expression-file expressions.json
```

### Options

```
  -e, --expression string        Expression string to evaluate
  -f, --expression-file string   Path to JSON file containing expressions mapped to expressionMap field
  -h, --help                     help for evaluate
```

### Options inherited from parent commands

```
  -o, --output string   Output format (text|table|json) (default "table")
  -v, --version         Print the version number of omnistrate-ctl
```

### SEE ALSO

* [omnistrate-ctl instance](omnistrate-ctl_instance.md)	 - Manage Instance Deployments for your service

