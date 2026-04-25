## omnistrate-ctl account customer create

Create a customer BYOA account onboarding instance

### Synopsis

This command onboards a customer cloud account into the injected BYOA account-config resource for a specific service plan.

```
omnistrate-ctl account customer create --service=[service] --environment=[environment] --plan=[plan] [provider flags] [flags]
```

### Examples

```
# Onboard a Nebius BYOA account into a service plan
	omnistrate-ctl account customer create \
	  --service=postgres \
	  --environment=prod \
	  --plan=customer-hosted \
	  --customer-email=customer@example.com \
	  --nebius-tenant-id=tenant-xxxx \
	  --nebius-bindings-file=./nebius-bindings.yaml
```

### Options

```
      --allow-create-new-cloud-native-network   Allow the platform to create new cloud-native networks (VPCs) in this account on demand
      --aws-account-id string                   AWS account ID
      --azure-subscription-id string            Azure subscription ID
      --azure-tenant-id string                  Azure tenant ID
      --customer-email string                   Customer email to onboard on behalf of in production environments
      --environment string                      Environment name or ID
      --gcp-project-id string                   GCP project ID
      --gcp-project-number string               GCP project number
  -h, --help                                    help for create
      --nebius-bindings-file string             Path to a YAML file describing Nebius bindings
      --nebius-tenant-id string                 Nebius tenant ID
      --plan string                             Service plan name or ID
      --private-link                            Enable AWS PrivateLink connectivity for services deployed in this account
      --service string                          Service name or ID
      --skip-wait                               Skip waiting for the account to become READY
      --subscription-id string                  Subscription ID to use for the onboarding instance
```

### Options inherited from parent commands

```
  -o, --output string   Output format (text|table|json) (default "table")
  -v, --version         Print the version number of omnistrate-ctl
```

### SEE ALSO

* [omnistrate-ctl account customer](omnistrate-ctl_account_customer.md)	 - Manage customer BYOA account onboarding

