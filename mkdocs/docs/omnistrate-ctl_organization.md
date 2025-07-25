## omnistrate-ctl organization

Manage organization-level configurations

### Synopsis

Manage organization-level configurations including amenities templates.

This command provides access to organization-level management operations including:
- Initialize and update amenities configuration templates
- Manage environment-specific organization settings

These operations affect organization-wide policies and templates that can be
applied to deployment cells within the organization.

```
omnistrate-ctl organization [command] [flags]
```

### Options

```
  -h, --help   help for organization
```

### Options inherited from parent commands

```
  -o, --output string   Output format (text|table|json) (default "table")
  -v, --version         Print the version number of omnistrate-ctl
```

### SEE ALSO

* [omnistrate-ctl](omnistrate-ctl.md)	 - Manage your Omnistrate SaaS from the command line
* [omnistrate-ctl organization amenities](omnistrate-ctl_organization_amenities.md)	 - Manage organization amenities configuration templates

