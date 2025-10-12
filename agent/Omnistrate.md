# Omnistrate Platform Guide

## Overview
Omnistrate is an Agentic Control Plane as-a-Service that enables ISVs to extend their existing Docker Compose definitions and deploy them as PaaS offerings across isolated networks, cloud accounts, or on-premises infrastructure. It provides a unified control plane for managing tenant-isolated deployments at scale.

## Core Value Proposition
- **Compose Extension**: Extend existing Docker Compose with Omnistrate-specific tags to build PaaS distribution
- **Lift-and-Shift**: Deploy existing Compose stacks without rewriting application code
- **Tenant Isolation**: Each customer tenant gets dedicated infrastructure in isolated networks or cloud accounts
- **BYOC Distribution**: Primary focus on deploying in customer's own cloud accounts (BYOC) or on-premises
- **Multi-Cloud Support**: Deploy across AWS, GCP, Azure with consistent tenant isolation
- **Full Automation**: Automated provisioning, tenant onboarding, operations, and recovery across distributed infrastructure
- **10x Cost Effective**: Compared to building multi-tenant control planes in-house

## Key Concepts

### Service Definition
- **Resource**: A service component in your Docker Compose that represents a deployable unit (database, API, worker, etc.). Your PaaS offering is a collection of 1+ Resources defined in the compose file.
- **Compose Spec**: Start with standard Docker Compose (v3.9+), then extend it with Omnistrate-specific `x-` extension tags to enable PaaS distribution capabilities.

### Service Plans/Tiers
- Configure different deployment tiers (small, medium, large) with specific resource allocations
- Each service plan defines compute, storage, and capability configurations
- Primary tenancy model: Dedicated isolated deployments per tenant
- Each tenant deployment runs in separate isolated networks or cloud accounts

### Deployment Models
- **BYOC (Bring Your Own Cloud)**: PRIMARY - Deploy entire compose stack in customer's cloud account with dedicated isolated networks per tenant
- **On-Premises**: Deploy compose stack on customer's infrastructure with tenant isolation
- **Air-Gapped**: Isolated environments for security-sensitive deployments
- **Hybrid**: Combination of on-premises and cloud deployments

## Compose Extension Model

Omnistrate extends standard Docker Compose with custom `x-` extension tags that enable PaaS distribution capabilities:

- **Service Plans**: Define deployment tiers and tenant isolation models
- **Compute Configuration**: Specify instance types, replicas, and resource requirements
- **Capabilities**: Enable autoscaling, backup, multi-zone deployment, network isolation
- **Integrations**: Configure observability, logging, metrics, and customer-specific integrations
- **Variable Interpolation**: Dynamic configuration using API and system parameters

The agent has access to MCP tools for discovering specific tag schemas and configurations.

## Platform Capabilities

### Build & Distribution
- Start with existing Docker Compose definitions
- Extend with Omnistrate tags to enable PaaS capabilities
- Auto-generate self-service customer portals
- Auto-generate APIs and CLIs
- Support multiple deployment interfaces (API, CLI, UI)

### Operations Management
- Fleet health monitoring and automated recovery
- Software inventory tracking
- Automated patching and upgrades
- Continuous deployment support
- Proactive monitoring with intelligent fleet management

### Financial Operations
- Usage metering
- Flexible billing models
- Marketplace integration (AWS, GCP, Azure)
- Billing integration (Stripe Connect)
- Cost visibility and optimization recommendations

### Security & Compliance
- Enterprise-grade security controls
- Compliance framework support
- SSO and RBAC
- Service account policies for cloud service interactions
- Comprehensive audit logging

### Integrations
- **Observability**: Native dashboards, OpenTelemetry (NewRelic, Signoz, Datadog), Cloud-native (CloudWatch, GCP Operations, Azure App Insights)
- **Alarming**: PagerDuty
- **Billing**: Stripe Connect
- **Marketplace**: AWS, GCP, Azure
- **Cloud Insurance**: Available

## Typical Onboarding Flow

1. **Compose Assessment**: Review customer's existing Docker Compose definition
2. **Extend Compose**: Add Omnistrate `x-` extension tags to enable PaaS distribution
3. **Configure Tenant Isolation**: Set up dedicated tenancy with network isolation per tenant
4. **Cloud Account Setup**: Deploy Control Plane agent in customer's cloud account(s) for BYOC model
5. **Build Service**: Use Omnistrate CTL to build service from extended compose spec
6. **Tenant Onboarding**: Each new customer tenant gets isolated deployment in separate network/cloud account
7. **Operate**: Monitor distributed tenant fleet, manage upgrades across isolated environments

## Use Cases (PaaS-Focused)
- **Enterprise BYOC Offerings**: Deploy software in customer cloud accounts with tenant isolation
- **On-Premises Distribution**: Deliver software to customer data centers with automated operations
- **Multi-Cloud PaaS**: Manage isolated tenant deployments across AWS, GCP, Azure from single control plane
- **Internal PaaS Platforms**: Provide isolated environments for internal teams/business units
- **Open-Source PaaS Monetization**: Enable open-source projects to offer managed deployments in customer accounts
- **Regulated Industry Deployments**: Air-gapped and isolated deployments for compliance requirements

## Tools
- **Omnistrate CTL**: CLI tool for building services, managing service plans, and managing deployments
- **Customer Portal**: Auto-generated self-service portal for end customers
- **APIs**: Auto-generated REST APIs for programmatic access

## FDE Approach for Customer Onboarding

### Discovery Phase
- **Review Existing Compose**: Examine customer's current Docker Compose definition
- **Map Service Architecture**: Identify all services (databases, APIs, workers, caches, etc.) and dependencies
- **Understand Tenant Model**: Determine how many customer tenants need isolated deployments
- **Cloud Account Strategy**: Clarify if tenants bring their own cloud accounts (BYOC) or deploy on-premises
- **Network Isolation Requirements**: Understand isolation needs (VPC per tenant, separate cloud accounts, air-gapped)
- **Deployment Targets**: Identify target clouds (AWS, GCP, Azure) and regions

### Compose Extension Phase
- **Extend Compose with Omnistrate Tags**: Add `x-` extension tags to existing compose definition
- **Preserve Existing Structure**: Maintain existing service architecture and dependencies
- **Configure Tenant Isolation**: Set up dedicated tenancy for isolated deployments per tenant
- **Map Services to Resources**: Each compose service becomes an Omnistrate Resource
- **Configure Compute**: Define instance types and resource allocations using extension tags
- **Enable Capabilities**: Add autoscaling, backup, multi-zone deployment, network isolation as needed

### Service Plan Configuration
- **Create Deployment Tiers**: Define service plans for different customer tiers (e.g., small, medium, large instances)
- **Configure BYOC Settings**: Set up cloud account connection parameters for customer accounts
- **Isolation Per Tenant**: Ensure each service plan enforces dedicated infrastructure per tenant
- **Regional Deployment**: Configure multi-region support if tenants are geographically distributed

### Testing & Validation
- **Test Isolated Deployments**: Deploy multiple test tenant instances in separate networks/accounts
- **Validate Network Isolation**: Verify tenants cannot access each other's infrastructure
- **Test BYOC Flow**: Connect to test cloud accounts and validate compose stack deployment
- **Verify Operations**: Test upgrades, patches, backup/restore across isolated tenant environments
- **Performance Validation**: Ensure isolated deployments meet performance requirements

### Tenant Onboarding (Production)
- **Cloud Account Connection**: Connect customer tenant's cloud account or set up on-premises agent
- **Deploy Isolated Infrastructure**: Provision dedicated network and infrastructure for tenant
- **Compose Stack Deployment**: Deploy complete compose stack in tenant's isolated environment
- **Validation**: Verify tenant deployment is functional and isolated
- **Handoff**: Provide customer tenant with access to their dedicated environment

### Multi-Tenant Operations
- **Fleet Management**: Monitor health of all isolated tenant deployments from unified control plane
- **Coordinated Upgrades**: Roll out patches/upgrades across distributed tenant fleet
- **Tenant-Specific Operations**: Handle tenant-specific customizations, scaling, backup/restore
- **Cost Tracking**: Monitor per-tenant resource usage and costs (especially for BYOC)
- **Troubleshooting**: Debug issues in specific tenant environments while maintaining isolation

### Key Success Metrics
- **Deployment Time**: How quickly new tenants can be onboarded with isolated infrastructure
- **Operational Efficiency**: Ability to manage N isolated tenant deployments from single control plane
- **Tenant Isolation**: Zero cross-tenant access or data leakage
- **Upgrade Success Rate**: Successful rollout of updates across distributed tenant fleet
