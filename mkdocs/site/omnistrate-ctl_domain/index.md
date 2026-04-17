## omnistrate-ctl domain

Manage Customer Domains for your service

### Synopsis

This command helps you manage the domains for your service. These domains are used to access your service in the cloud. You can set up custom domains for each environment type, such as 'dev', 'prod', 'qa', 'canary', 'staging', 'private'.

```text
omnistrate-ctl domain [operation] [flags]
```

### Options

```text
  -h, --help   help for domain
```

### Options inherited from parent commands

```text
  -o, --output string   Output format (text|table|json) (default "table")
  -v, --version         Print the version number of omnistrate-ctl
```

### SEE ALSO

- [omnistrate-ctl](../omnistrate-ctl/) - Manage your Omnistrate SaaS from the command line
- [omnistrate-ctl domain create](../omnistrate-ctl_domain_create/) - Create a Custom Domain
- [omnistrate-ctl domain delete](../omnistrate-ctl_domain_delete/) - Delete a Custom Domain
- [omnistrate-ctl domain list](../omnistrate-ctl_domain_list/) - List SaaS Portal Custom Domains
