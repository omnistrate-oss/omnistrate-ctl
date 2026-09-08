## omnistrate-ctl instance create

Create an instance deployment

### Synopsis

This command helps you create an instance deployment for your service.

```
omnistrate-ctl instance create --service=[service] --environment=[environment] --plan=[plan] --version=[version] --resource=[resource] [--cloud-provider=aws|gcp|azure|nebius] [--region=region] [--param=param] [--param-file=file-path] [--instance-id=id] [--customer-email=email] [--customer-account-id=account-instance-id] [--cloud-provider-native-network-id=network-id] [--network-type=PUBLIC|INTERNAL] [--onprem-platform=platform] [--tags key=value,key2=value2] [--breakpoints id-or-key[:event[|event...]],...] [flags]
```

### Examples

```
# Create an instance deployment
omnistrate-ctl instance create --service=mysql --environment=dev --plan=mysql --version=latest --resource=mySQL --cloud-provider=aws --region=ca-central-1 --param '{"databaseName":"default","password":"a_secure_password","rootPassword":"a_secure_root_password","username":"user"}'

# Create an instance deployment with parameters from a file
omnistrate-ctl instance create --service=mysql --environment=dev --plan=mysql --version=latest --resource=mySQL --cloud-provider=aws --region=ca-central-1 --param-file /path/to/params.json

# Create an instance deployment with custom tags
omnistrate-ctl instance create --service=mysql --environment=dev --plan=mysql --version=latest --resource=mySQL --cloud-provider=aws --region=ca-central-1 --param-file /path/to/params.json --tags environment=dev,owner=team

# Create an instance deployment with an internal network type
omnistrate-ctl instance create --service=mysql --environment=dev --plan=mysql --version=latest --resource=mySQL --cloud-provider=aws --region=ca-central-1 --param-file /path/to/params.json --network-type INTERNAL

# Create an instance deployment and wait for completion with progress tracking
omnistrate-ctl instance create --service=mysql --environment=dev --plan=mysql --version=latest --resource=mySQL --cloud-provider=aws --region=ca-central-1 --param-file /path/to/params.json --wait

# Create an instance deployment with workflow breakpoints
omnistrate-ctl instance create --service=mysql --environment=dev --plan=mysql --version=latest --resource=mySQL --cloud-provider=aws --region=ca-central-1 --param-file /path/to/params.json --breakpoints writer,reader

# Create an instance deployment with resource event workflow breakpoints
omnistrate-ctl instance create --service=mysql --environment=dev --plan=mysql --version=latest --resource=mySQL --cloud-provider=aws --region=ca-central-1 --param-file /path/to/params.json --breakpoints 'terraform:StartTerraformPlan|CompleteTerraformPlan,helm:StartHelmInstall|CompleteHelmInstall'

# Create a BYOA instance deployment using a customer account onboarding instance
omnistrate-ctl instance create --service=Nebius --environment=dev --plan='Nebius BYOA Compute Variants' --resource=NebiusRedis --cloud-provider=nebius --region=eu-north1 --customer-account-id instance-cg1tthkj0

# Create a BYOA instance deployment using a customer account onboarding instance with imported network
omnistrate-ctl instance create --service=MyService --environment=dev --plan='AWS BYOA' --resource=myResource --cloud-provider=aws --region=us-east-2 --customer-account-id instance-cg1tthkj0 --cloud-provider-native-network-id vpc-0123456789abcdef0

# Create an air-gapped/on-prem installer-backed instance. Do not pass --cloud-provider or --region with --onprem-platform.
omnistrate-ctl instance create --service=MyService --environment=dev --plan='Airgap' --resource=myResource --onprem-platform=Generic --param-file /path/to/params.json

# Create an instance deployment on behalf of an end customer, resolved from their email
omnistrate-ctl instance create --service=mysql --environment=prod --plan=mysql --resource=mySQL --cloud-provider=aws --region=us-east-2 --param-file /path/to/params.json --customer-email customer@example.com
```

### Options

```
      --breakpoints string                        Workflow breakpoint resource IDs or resource keys, optionally scoped to events as id-or-key:event or id-or-key:event|event
      --cloud-provider string                     Cloud provider (aws|gcp|azure|nebius). Required unless --onprem-platform is provided; do not use with --onprem-platform.
      --cloud-provider-native-network-id string   Cloud provider native network ID to inject as cloud_provider_native_network_id in instance deployment parameters
      --customer-account-id string                Customer BYOA account onboarding instance ID to inject as the cloud account. Use 'omnistrate-ctl account customer list' or 'omnistrate-ctl account customer describe <instance-id>' to find it.
      --customer-email string                     Customer email to create the instance deployment on behalf of. Resolves the customer's subscription for this service plan, creating one if they have none. Cannot be combined with --subscription-id.
      --environment string                        Environment name
  -h, --help                                      help for create
      --instance-id string                        ID of a previously deleted instance to restore
      --network-type string                       Optional network type for the instance deployment (PUBLIC / INTERNAL)
      --onprem-platform string                    On-prem platform for installer-backed deployments (for example EKS, GKE, AKS, OpenShift, Generic)
      --param string                              Parameters for the instance deployment
      --param-file string                         Json file containing parameters for the instance deployment
      --plan string                               Service plan name
      --region string                             Region code (e.g. us-east-2, us-central1). Required unless --onprem-platform is provided; do not use with --onprem-platform.
      --resource string                           Resource name
      --service string                            Service name
      --subscription-id string                    Subscription ID to use for the instance deployment. If not provided, instance deployment will be created in your own subscription.
      --tags string                               Custom tags to add to the instance deployment (format: key=value,key2=value2). Escape commas inside values with \,
      --version string                            Service plan version (latest|preferred|1.0 etc.). With 'preferred', no version is sent and the platform picks the preferred version automatically. (default "preferred")
      --wait                                      Wait for deployment to complete and show progress
```

### Options inherited from parent commands

```
  -o, --output string   Output format (text|table|json) (default "table")
```

### SEE ALSO

* [omnistrate-ctl instance](omnistrate-ctl_instance.md)	 - Manage Instance Deployments for your service

