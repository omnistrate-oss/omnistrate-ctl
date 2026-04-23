## omnistrate-ctl instance create

Create or restore an instance deployment

### Synopsis

This command helps you create an instance deployment for your service. When --instance-id is provided, it restores a previously deleted instance using the specified ID, preserving the original instance ID and endpoint.

```
omnistrate-ctl instance create --service=[service] --environment=[environment] --plan=[plan] --version=[version] --resource=[resource] --cloud-provider=[aws|gcp|azure|nebius] --region=[region] [--param=param] [--param-file=file-path] [--instance-id=id] [--customer-account-id=account-instance-id] [--tags key=value,key2=value2] [--breakpoints id-or-key,id-or-key] [flags]
```

### Examples

```
# Create an instance deployment
omnistrate-ctl instance create --service=mysql --environment=dev --plan=mysql --version=latest --resource=mySQL --cloud-provider=aws --region=ca-central-1 --param '{"databaseName":"default","password":"a_secure_password","rootPassword":"a_secure_root_password","username":"user"}'

# Create an instance deployment with parameters from a file
omnistrate-ctl instance create --service=mysql --environment=dev --plan=mysql --version=latest --resource=mySQL --cloud-provider=aws --region=ca-central-1 --param-file /path/to/params.json

# Create an instance deployment with custom tags
omnistrate-ctl instance create --service=mysql --environment=dev --plan=mysql --version=latest --resource=mySQL --cloud-provider=aws --region=ca-central-1 --param-file /path/to/params.json --tags environment=dev,owner=team

# Create an instance deployment and wait for completion with progress tracking
omnistrate-ctl instance create --service=mysql --environment=dev --plan=mysql --version=latest --resource=mySQL --cloud-provider=aws --region=ca-central-1 --param-file /path/to/params.json --wait

# Create an instance deployment with workflow breakpoints
omnistrate-ctl instance create --service=mysql --environment=dev --plan=mysql --version=latest --resource=mySQL --cloud-provider=aws --region=ca-central-1 --param-file /path/to/params.json --breakpoints writer,reader

# Create a BYOA instance deployment using a customer account onboarding instance
omnistrate-ctl instance create --service=Nebius --environment=dev --plan='Nebius BYOA Compute Variants' --resource=NebiusRedis --cloud-provider=nebius --region=eu-north1 --customer-account-id instance-cg1tthkj0

# Restore a previously deleted instance
omnistrate-ctl instance create --service=mysql --environment=dev --plan=mysql --version=latest --resource=mySQL --cloud-provider=aws --region=ca-central-1 --instance-id instance-abcd1234
```

### Options

```
      --breakpoints string           Workflow breakpoint resource IDs or resource keys (comma-separated)
      --cloud-provider string        Cloud provider (aws|gcp|azure|nebius)
      --customer-account-id string   Customer BYOA account onboarding instance ID to inject as the cloud account. Use 'omnistrate-ctl account customer list' or 'omnistrate-ctl account customer describe <instance-id>' to find it.
      --environment string           Environment name
  -h, --help                         help for create
      --instance-id string           ID of a previously deleted instance to restore
      --param string                 Parameters for the instance deployment
      --param-file string            Json file containing parameters for the instance deployment
      --plan string                  Service plan name
      --region string                Region code (e.g. us-east-2, us-central1)
      --resource string              Resource name
      --service string               Service name
      --subscription-id string       Subscription ID to use for the instance deployment. If not provided, instance deployment will be created in your own subscription.
      --tags string                  Custom tags to add to the instance deployment (format: key=value,key2=value2)
      --version string               Service plan version (latest|preferred|1.0 etc.) (default "preferred")
      --wait                         Wait for deployment to complete and show progress
```

### Options inherited from parent commands

```
  -o, --output string   Output format (text|table|json) (default "table")
```

### SEE ALSO

* [omnistrate-ctl instance](omnistrate-ctl_instance.md)	 - Manage Instance Deployments for your service

