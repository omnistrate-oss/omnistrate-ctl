## omnistrate-ctl domain create

Create a Custom Domain

### Synopsis

This command helps you create a Custom Domain.

```
omnistrate-ctl domain create [flags]
```

### Examples

```
# Create a custom domain for dev environment
omctl domain create dev --domain=abc.dev --environment-type=dev

# Create a custom domain for prod environment
omctl domain create abc.cloud --domain=abc.cloud --environment-type=prod
```

### Options

```
      --domain string             Custom domain
      --environment-type string   Type of environment. Valid options include: 'dev', 'prod', 'qa', 'canary', 'staging', 'private'
  -h, --help                      help for create
```

### Options inherited from parent commands

```
  -o, --output string   Output format (text|table|json) (default "table")
  -v, --version         Print the version number of omnistrate-ctl
```

### SEE ALSO

- [omnistrate-ctl domain](omnistrate-ctl_domain.md) - Manage Customer Domains for your service
