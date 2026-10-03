## omnistrate-ctl subscription update

Update a subscription

### Synopsis

Update a subscription for a service environment.

```
omnistrate-ctl subscription update <subscription-id> [flags]
```

### Options

```
      --allowed-deployment-locations string   Subscription deployment location restriction as a JSON array. Set to [] to inherit product tier deployment locations
  -e, --environment-id string                 Environment ID (required)
  -h, --help                                  help for update
  -s, --service-id string                     Service ID (required)
```

### Options inherited from parent commands

```
  -o, --output string   Output format (text|table|json) (default "table")
  -v, --version         Print the version number of omnistrate-ctl
```

### SEE ALSO

* [omnistrate-ctl subscription](omnistrate-ctl_subscription.md)	 - Manage Customer Subscriptions for your service

