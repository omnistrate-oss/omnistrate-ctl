## omnistrate-ctl domain

Manage Customer Domains for your service

### Synopsis

This command helps you manage the domains for your service.
These domains are used to access your service in the cloud. You can set up custom domains for each environment type, such as 'dev', 'prod', 'qa', 'canary', 'staging', 'private'.

```
omnistrate-ctl domain [operation] [flags]
```

### Options

```
  -h, --help   help for domain
```

### Options inherited from parent commands

```
  -o, --output string   Output format (text|table|json) (default "table")
  -v, --version         Print the version number of omnistrate-ctl
```

### SEE ALSO

- [omnistrate-ctl](omnistrate-ctl.md) - Manage your Omnistrate SaaS from the command line
- [omnistrate-ctl domain create](omnistrate-ctl_domain_create.md) - Create a Custom Domain
- [omnistrate-ctl domain delete](omnistrate-ctl_domain_delete.md) - Delete a Custom Domain
- [omnistrate-ctl domain list](omnistrate-ctl_domain_list.md) - List SaaS Portal Custom Domains
