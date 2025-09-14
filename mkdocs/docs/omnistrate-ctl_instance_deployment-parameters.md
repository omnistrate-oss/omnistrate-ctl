## omnistrate-ctl instance deployment-parameters

List API parameters configurable for instance deployment

### Synopsis

This command retrieves and displays the configurable API parameters from the service offerings API that can be used during instance deployment.

```
omnistrate-ctl instance deployment-parameters --service=[service] --plan=[plan] --version=[version] --resource=[resource] [flags]
```

### Examples

```
  omnistrate-ctl instance deployment-parameters --service=mysql --plan=mysql --version=latest --resource=mySQL
  omnistrate-ctl instance deployment-parameters --service=mysql --plan=mysql --version=latest --resource=mySQL --output=json
```

### Options

```
  -h, --help              help for deployment-parameters
  -o, --output string     Output format (table|json) (default "table")
      --plan string       Service plan name
      --resource string   Resource name
      --service string    Service name
      --version string    Service plan version (latest|preferred|1.0 etc.) (default "preferred")
```

### SEE ALSO

* [omnistrate-ctl instance](omnistrate-ctl_instance.md)	 - Manage Instance Deployments for your service

