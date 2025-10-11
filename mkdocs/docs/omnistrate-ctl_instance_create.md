## omnistrate-ctl instance create

Create an instance deployment

### Synopsis

This command helps you create an instance deployment for your service.

```
omnistrate-ctl instance create --service=[service] --environment=[environment] --plan=[plan] --version=[version] --resource=[resource] --cloud-provider=[aws|gcp] --region=[region] [--param=param] [--param-file=file-path] [flags]
```

### Examples

```
# Create an instance deployment
omctl instance create --service=mysql --environment=dev --plan=mysql --version=latest --resource=mySQL --cloud-provider=aws --region=ca-central-1 --param '{"databaseName":"default","password":"a_secure_password","rootPassword":"a_secure_root_password","username":"user"}'

# Create an instance deployment with parameters from a file
omctl instance create --service=mysql --environment=dev --plan=mysql --version=latest --resource=mySQL --cloud-provider=aws --region=ca-central-1 --param-file /path/to/params.json

# Create an instance deployment and wait for completion with progress tracking
omctl instance create --service=mysql --environment=dev --plan=mysql --version=latest --resource=mySQL --cloud-provider=aws --region=ca-central-1 --param-file /path/to/params.json --wait
```

### Options

```
      --cloud-provider string    Cloud provider (aws|gcp)
      --environment string       Environment name
  -h, --help                     help for create
      --param string             Parameters for the instance deployment
      --param-file string        Json file containing parameters for the instance deployment
      --plan string              Service plan name
      --region string            Region code (e.g. us-east-2, us-central1)
      --resource string          Resource name
      --service string           Service name
      --subscription-id string   Subscription ID to use for the instance deployment. If not provided, instance deployment will be created in your own subscription.
      --version string           Service plan version (latest|preferred|1.0 etc.) (default "preferred")
      --wait                     Wait for deployment to complete and show progress
```

### Options inherited from parent commands

```
  -o, --output string   Output format (text|table|json) (default "table")
```

### SEE ALSO

* [omnistrate-ctl instance](omnistrate-ctl_instance.md)	 - Manage Instance Deployments for your service

