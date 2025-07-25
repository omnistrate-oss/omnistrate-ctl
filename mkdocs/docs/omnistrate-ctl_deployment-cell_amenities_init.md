## omnistrate-ctl deployment-cell amenities init

Initialize organization-level amenities configuration

### Synopsis

Initialize organization-level amenities configuration through an interactive process.

This command starts an interactive process to define the default organization-level 
amenities configuration. This step is purely at the org level; no reference to any 
service is needed.

The configuration will be stored as a template that can be applied to different 
environments (production, staging, development) and used to synchronize deployment cells.

```
omnistrate-ctl deployment-cell amenities init [flags]
```

### Options

```
  -f, --config-file string       Path to configuration JSON file (optional)
  -e, --environment string       Target environment (production, staging, development)
  -h, --help                     help for init
      --interactive              Use interactive mode to configure amenities (default true)
  -g, --organization-id string   Organization ID (required)
```

### Options inherited from parent commands

```
  -o, --output string   Output format (text|table|json) (default "table")
  -v, --version         Print the version number of omnistrate-ctl
```

### SEE ALSO

* [omnistrate-ctl deployment-cell amenities](omnistrate-ctl_deployment-cell_amenities.md)	 - Manage deployment cell amenities configuration

