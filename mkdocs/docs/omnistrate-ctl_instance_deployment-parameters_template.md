## omnistrate-ctl instance deployment-parameters template

Generate a JSON parameter template for instance create

### Synopsis

This command generates a JSON parameter template for the given resource that can be filled in and passed to 'instance create --param-file'. Parameters with defaults are pre-populated; others get typed placeholders.

```
omnistrate-ctl instance deployment-parameters template --service=[service] --environment=[environment] --plan=[plan] --version=[version] --resource=[resource] [flags]
```

### Examples

```
  omnistrate-ctl instance deployment-parameters template --service=mysql --environment=dev --plan=mysql --version=latest --resource=mySQL > params.json
  omnistrate-ctl instance create --service=mysql --environment=dev --plan=mysql --resource=mySQL --cloud-provider=aws --region=us-east-2 --param-file params.json
```

### Options

```
      --environment string   Environment name or ID
  -h, --help                 help for template
      --plan string          Service plan name
      --resource string      Resource name
      --service string       Service name
      --version string       Service plan version (latest|preferred|1.0 etc.) (default "preferred")
```

### Options inherited from parent commands

```
  -o, --output string   Output format (text|table|json) (default "table")
```

### SEE ALSO

* [omnistrate-ctl instance deployment-parameters](omnistrate-ctl_instance_deployment-parameters.md)	 - List API parameters configurable for instance deployment

