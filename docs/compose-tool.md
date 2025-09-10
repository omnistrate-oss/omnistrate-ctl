# Docker Compose Generator Tool for Omnistrate MCP Server

## Overview

The `compose` command is a new tool added to the Omnistrate CTL that allows you to generate custom Docker Compose specifications that follow the Omnistrate specification. This tool is particularly useful for creating properly formatted compose files with Omnistrate-specific extensions (`x-omnistrate-*`).

## Usage

### Basic Command Structure

```bash
omnistrate-ctl compose generate [flags]
```

### Required Flags

- `--service-plan-name` (`-n`): Name of the service plan (required)

### Service Configuration (Choose One)

**Multiple Services Mode (Recommended):**
- `--services`: Services configuration as JSON string containing an array of service definitions

**Legacy Single Service Mode:**
- `--service-name` (`-s`): Name of the main service (required for legacy mode)  
- `--image` (`-i`): Docker image for the service (required for legacy mode)

### Optional Flags

#### Basic Docker Configuration
- `--ports`: Port mappings (e.g., '8080:8080')
- `--environment`: Environment variables (e.g., 'KEY=value')
- `--volumes`: Volume mounts (e.g., './data:/app/data')

#### Omnistrate-Specific Configuration
- `--root-volume-size`: Root volume size in GB (default: 20)
- `--replica-count`: Number of replicas (default: 1)
- `--replica-count-api-param`: API parameter name for replica count
- `--enable-multi-zone`: Enable multi-zone deployment
- `--enable-endpoint-per-replica`: Enable endpoint per replica
- `--mode-internal`: Set service as internal mode

#### Cloud Provider Configuration
- `--cloud-providers`: Supported cloud providers (aws, gcp, azure)
- `--instance-type-api-param`: API parameter name for instance type

#### Advanced Configuration
- `--api-params`: API parameters as JSON string
- `--integrations`: Omnistrate integrations (omnistrateLogging, omnistrateMetrics)

#### Output Configuration
- `--output-file` (`-f`): Output file path (default: stdout)
- `--compose-version`: Docker Compose version (default: "3.9")

## Examples

### Multiple Services Mode (Recommended)

#### Full-Stack Web Application

```bash
omnistrate-ctl generate compose \
  --service-plan-name "Full-Stack Web App" \
  --services '[
    {
      "name": "frontend",
      "image": "nginx:latest",
      "ports": ["80:80", "443:443"],
      "depends_on": ["backend"],
      "replicaCount": 2,
      "enableMultiZone": true,
      "instanceTypes": [
        {"cloudProvider": "aws"},
        {"cloudProvider": "gcp"}
      ]
    },
    {
      "name": "backend",
      "image": "myapp/api:v1.2.0",
      "ports": ["8080:8080"],
      "environment": ["NODE_ENV=production", "PORT=8080"],
      "depends_on": ["database", "redis"],
      "replicaCount": 3,
      "enableMultiZone": true,
      "enableEndpointPerReplica": true,
      "apiParams": [
        {
          "key": "apiKey",
          "type": "string",
          "description": "API authentication key",
          "export": true
        }
      ]
    },
    {
      "name": "database",
      "image": "postgres:15",
      "environment": ["POSTGRES_DB=myapp", "POSTGRES_USER=admin"],
      "volumes": ["./data:/var/lib/postgresql/data"],
      "rootVolumeSizeGi": 100,
      "replicaCount": 1,
      "modeInternal": true,
      "backupConfiguration": {
        "backupPeriodInHours": 24
      }
    },
    {
      "name": "redis",
      "image": "redis:7-alpine",
      "ports": ["6379:6379"],
      "rootVolumeSizeGi": 20,
      "modeInternal": true
    }
  ]' \
  --integrations "omnistrateLogging" \
  --integrations "omnistrateMetrics" \
  --output-file "fullstack-compose.yaml"
```

#### Microservices Architecture

```bash
omnistrate-ctl generate compose \
  --service-plan-name "Microservices Platform" \
  --services '[
    {
      "name": "api-gateway",
      "image": "traefik:v2.10",
      "ports": ["80:80", "8080:8080"],
      "replicaCount": 2,
      "enableMultiZone": true
    },
    {
      "name": "user-service",
      "image": "myorg/user-service:latest",
      "ports": ["3001:3001"],
      "environment": ["SERVICE_PORT=3001"],
      "depends_on": ["user-db"],
      "replicaCount": 2,
      "enableEndpointPerReplica": true
    },
    {
      "name": "order-service",
      "image": "myorg/order-service:latest",
      "ports": ["3002:3002"],
      "environment": ["SERVICE_PORT=3002"],
      "depends_on": ["order-db"],
      "replicaCount": 3,
      "autoscaling": {
        "minReplicas": 2,
        "maxReplicas": 10
      }
    },
    {
      "name": "user-db",
      "image": "postgres:15",
      "environment": ["POSTGRES_DB=users"],
      "rootVolumeSizeGi": 50,
      "modeInternal": true
    },
    {
      "name": "order-db",
      "image": "postgres:15",
      "environment": ["POSTGRES_DB=orders"],
      "rootVolumeSizeGi": 100,
      "modeInternal": true
    }
  ]'
```

### Legacy Single Service Mode

#### Basic PostgreSQL Service

```bash
omnistrate-ctl generate compose \
  --service-plan-name "PostgreSQL Database" \
  --service-name "postgres" \
  --image "postgres:15" \
  --ports "5432:5432" \
  --environment "POSTGRES_USER=admin" \
  --environment "POSTGRES_PASSWORD=secret" \
  --volumes "./data:/var/lib/postgresql/data"
```

#### Multi-Cloud Redis with Integrations

```bash
omnistrate-ctl generate compose \
  --service-plan-name "Redis Cache" \
  --service-name "redis" \
  --image "redis:7" \
  --ports "6379:6379" \
  --cloud-providers "aws" \
  --cloud-providers "gcp" \
  --integrations "omnistrateLogging" \
  --integrations "omnistrateMetrics" \
  --enable-multi-zone \
  --output-file "redis-compose.yaml"
```

#### Advanced MySQL Cluster with API Parameters

```bash
omnistrate-ctl generate compose \
  --service-plan-name "MySQL Cluster" \
  --service-name "mysql" \
  --image "mysql:8.0" \
  --ports "3306:3306" \
  --environment "MYSQL_ROOT_PASSWORD=rootpass" \
  --cloud-providers "aws" \
  --cloud-providers "gcp" \
  --enable-multi-zone \
  --enable-endpoint-per-replica \
  --replica-count-api-param "numReplicas" \
  --instance-type-api-param "instanceType" \
  --api-params '[{"key":"rootPassword","type":"string","description":"MySQL root password","export":true}]' \
  --integrations "omnistrateLogging" \
  --integrations "omnistrateMetrics"
```

## Generated Output Structure

The tool generates a Docker Compose file with the following structure:

```yaml
version: "3.9"
services:
  <service-name>:
    image: <docker-image>
    ports:
      - <port-mappings>
    environment:
      - <environment-variables>
    volumes:
      - <volume-mounts>
    x-omnistrate-compute:
      rootVolumeSizeGi: <size>
      replicaCount: <count>
      instanceTypes:
        - cloudProvider: <provider>
    x-omnistrate-capabilities:
      enableMultiZone: <boolean>
      enableEndpointPerReplica: <boolean>
    x-omnistrate-api-params:
      - key: <param-key>
        description: <param-description>
        type: <param-type>
        export: <boolean>
    x-omnistrate-mode-internal: <boolean>
x-omnistrate-service-plan:
  name: <service-plan-name>
x-omnistrate-integrations:
  - <integration-name>
```

## Supported Omnistrate Extensions

The tool supports all major Omnistrate-specific extensions:

- `x-omnistrate-service-plan`: Service plan configuration
- `x-omnistrate-compute`: Compute resource specifications
- `x-omnistrate-capabilities`: Service capabilities (multi-zone, endpoints, etc.)
- `x-omnistrate-api-params`: API parameter definitions
- `x-omnistrate-integrations`: Omnistrate platform integrations
- `x-omnistrate-mode-internal`: Internal/external service mode
- `x-omnistrate-actionhooks`: Action hooks (future enhancement)
- `x-omnistrate-storage`: Storage configurations (future enhancement)

## Integration with MCP Server

This tool is automatically available through the MCP server and can be used by MCP clients to generate Omnistrate-compliant Docker Compose specifications programmatically.

## Future Enhancements

Planned enhancements include:
- Support for action hooks configuration
- Advanced storage configuration options
- Load balancer configuration
- Image registry authentication
- Template-based generation
- Validation against Omnistrate schema
