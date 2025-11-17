## omnistrate-ctl cost

Manage cost analytics for your services

### Synopsis

This command helps you analyze costs for your services across different dimensions.
You can get cost breakdowns by cloud provider, deployment cell, region, user, instance type, or individual instance.

Available aggregations:
  by-provider       Cost breakdown by cloud provider
  by-cell           Cost breakdown by deployment cell
  by-region         Cost breakdown by region
  by-user           Cost breakdown by user
  by-instance-type  Cost breakdown by instance type (e.g., m5.large)
  by-instance       Cost breakdown by individual instance

Legacy commands (deprecated):
  cloud-provider    Use 'by-provider' instead
  deployment-cell   Use 'by-cell' instead
  region            Use 'by-region' instead
  user              Use 'by-user' instead

```
omnistrate-ctl cost [operation] [flags]
```

### Options

```
  -h, --help   help for cost
```

### Options inherited from parent commands

```
  -o, --output string   Output format (text|table|json) (default "table")
  -v, --version         Print the version number of omnistrate-ctl
```

### SEE ALSO

* [omnistrate-ctl](omnistrate-ctl.md)	 - Manage your Omnistrate SaaS from the command line
* [omnistrate-ctl cost by-cell](omnistrate-ctl_cost_by-cell.md)	 - Get cost breakdown by deployment cell
* [omnistrate-ctl cost by-instance](omnistrate-ctl_cost_by-instance.md)	 - Get cost breakdown by individual instance
* [omnistrate-ctl cost by-instance-type](omnistrate-ctl_cost_by-instance-type.md)	 - Get cost breakdown by instance type
* [omnistrate-ctl cost by-provider](omnistrate-ctl_cost_by-provider.md)	 - Get cost breakdown by cloud provider
* [omnistrate-ctl cost by-region](omnistrate-ctl_cost_by-region.md)	 - Get cost breakdown by region
* [omnistrate-ctl cost by-user](omnistrate-ctl_cost_by-user.md)	 - Get cost breakdown by user
* [omnistrate-ctl cost cloud-provider](omnistrate-ctl_cost_cloud-provider.md)	 - Get cost breakdown by cloud provider
* [omnistrate-ctl cost deployment-cell](omnistrate-ctl_cost_deployment-cell.md)	 - Get cost breakdown by deployment cell
* [omnistrate-ctl cost region](omnistrate-ctl_cost_region.md)	 - Get cost breakdown by region
* [omnistrate-ctl cost user](omnistrate-ctl_cost_user.md)	 - Get cost breakdown by user

