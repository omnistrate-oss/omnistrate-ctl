## omnistrate-ctl subscription create-on-behalf

Create subscription on behalf of customer

### Synopsis

Create a subscription on behalf of a customer for a service environment.

```
omnistrate-ctl subscription create-on-behalf [flags]
```

### Options

```
      --allow-creates-without-payment   Allow creation without payment configured
      --billing-provider string         Billing provider
      --custom-price                    Whether to use custom price
      --custom-price-per-unit string    Custom price per unit (JSON object)
      --customer-user-id string         Customer user ID (required)
  -e, --environment-id string           Environment ID (required)
      --external-payer-id string        External payer ID
  -h, --help                            help for create-on-behalf
      --max-instances int               Maximum number of instances
      --price-effective-date string     Price effective date
      --product-tier-id string          Product tier ID (required)
  -s, --service-id string               Service ID (required)
```

### Options inherited from parent commands

```
  -o, --output string   Output format (text|table|json) (default "table")
  -v, --version         Print the version number of omnistrate-ctl
```

### SEE ALSO

* [omnistrate-ctl subscription](omnistrate-ctl_subscription.md)	 - Manage Customer Subscriptions for your service

