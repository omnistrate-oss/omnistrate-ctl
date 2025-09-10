## omnistrate-ctl

Manage your Omnistrate SaaS from the command line

### Synopsis

Omnistrate CTL is a powerful command-line tool designed to simplify the creation, deployment, and
management of your Omnistrate SaaS.

Key Features:

- Build Services: Create services from images or compose specs.
- Manage Service Plans: Efficiently handle your service plans.
- Deploy Instances: Deploy and manage instances with ease.

Resources:

- CTL Manual: https://ctl.omnistrate.cloud/omnistrate-ctl/
- Quick Start Guide: https://docs.omnistrate.com/getting-started/getting-started-with-ctl/

Use the CTL commands below to begin managing your services effectively.

```
omnistrate-ctl [flags]
```

### Options

```
  -h, --help            help for omnistrate-ctl
  -o, --output string   Output format (text|table|json) (default "table")
  -v, --version         Print the version number of omnistrate-ctl
```

### SEE ALSO

- [omnistrate-ctl account](omnistrate-ctl_account.md) - Manage your Cloud Provider Accounts
- [omnistrate-ctl alarms](omnistrate-ctl_alarms.md) - Manage alarms and notification channels
- [omnistrate-ctl audit](omnistrate-ctl_audit.md) - Audit events and logging management
- [omnistrate-ctl build](omnistrate-ctl_build.md) - Build Services from image, compose spec or service plan spec
- [omnistrate-ctl build-from-repo](omnistrate-ctl_build-from-repo.md) - Build Service from Git Repository
- [omnistrate-ctl cost](omnistrate-ctl_cost.md) - Manage cost analytics for your services
- [omnistrate-ctl custom-network](omnistrate-ctl_custom-network.md) - List and describe custom networks of your customers
- [omnistrate-ctl deployment-cell](omnistrate-ctl_deployment-cell.md) - Manage Deployment Cells
- [omnistrate-ctl domain](omnistrate-ctl_domain.md) - Manage Customer Domains for your service
- [omnistrate-ctl environment](omnistrate-ctl_environment.md) - Manage Service Environments for your service
- [omnistrate-ctl helm](omnistrate-ctl_helm.md) - Manage Helm Charts for your service
- [omnistrate-ctl inspect](omnistrate-ctl_inspect.md) - Interactive TUI to inspect Kubernetes resources
- [omnistrate-ctl instance](omnistrate-ctl_instance.md) - Manage Instance Deployments for your service
- [omnistrate-ctl login](omnistrate-ctl_login.md) - Log in to the Omnistrate platform
- [omnistrate-ctl logout](omnistrate-ctl_logout.md) - Logout
- [omnistrate-ctl mcp](omnistrate-ctl_mcp.md) - MCP server management
- [omnistrate-ctl operations](omnistrate-ctl_operations.md) - Operations and health monitoring commands
- [omnistrate-ctl secret](omnistrate-ctl_secret.md) - Manage secrets
- [omnistrate-ctl service](omnistrate-ctl_service.md) - Manage Services for your account
- [omnistrate-ctl service-plan](omnistrate-ctl_service-plan.md) - Manage Service Plans for your service
- [omnistrate-ctl services-orchestration](omnistrate-ctl_services-orchestration.md) - Manage Services Orchestration Deployments across services
- [omnistrate-ctl subscription](omnistrate-ctl_subscription.md) - Manage Customer Subscriptions for your service
- [omnistrate-ctl upgrade](omnistrate-ctl_upgrade.md) - Upgrade Instance Deployments to a newer or older version
- [omnistrate-ctl workflow](omnistrate-ctl_workflow.md) - Manage service workflows
