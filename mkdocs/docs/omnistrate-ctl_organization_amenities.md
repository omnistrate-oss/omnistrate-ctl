## omnistrate-ctl organization amenities

Manage organization amenities configuration templates

### Synopsis

Manage organization-level amenities configuration templates.

This command helps you:
- Initialize organization-level amenities configuration templates
- Update amenities configuration templates for target environments

These templates define the organization's amenities policies that can be applied
to deployment cells through drift detection and synchronization operations.

Available operations:
  init        Initialize organization-level amenities configuration template
  update      Update organization amenities configuration template for target environment

```
omnistrate-ctl organization amenities [operation] [flags]
```

### Options

```
  -h, --help   help for amenities
```

### Options inherited from parent commands

```
  -o, --output string   Output format (text|table|json) (default "table")
  -v, --version         Print the version number of omnistrate-ctl
```

### SEE ALSO

* [omnistrate-ctl organization](omnistrate-ctl_organization.md)	 - Manage organization-level configurations
* [omnistrate-ctl organization amenities init](omnistrate-ctl_organization_amenities_init.md)	 - Initialize organization-level amenities configuration template
* [omnistrate-ctl organization amenities update](omnistrate-ctl_organization_amenities_update.md)	 - Update organization amenities configuration template for target environment

