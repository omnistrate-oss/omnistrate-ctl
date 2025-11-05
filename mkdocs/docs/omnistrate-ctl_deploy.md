## omnistrate-ctl deploy

Deploy a service using a spec file

### Synopsis

Deploy a service using a spec file. This command builds the service in DEV, creates/checks PROD environment, promotes to PROD, marks as preferred, subscribes, and automatically creates/upgrades instances.

```
omnistrate-ctl deploy [spec-file] [flags]
```

### Examples

```

# Deploy a service using a spec file (automatically creates/upgrades instances)
omctl deploy spec.yaml

# Deploy a service with a custom product name
omctl deploy spec.yaml --product-name "My Service"

# Build service from an existing compose spec in the repository
omctl deploy --file omnistrate-compose.yaml

# Build service with a custom service name
omctl deploy --product-name my-custom-service

# Build service with service specification for Helm, Operator or Kustomize in prod environment
omctl deploy --file spec.yaml --product-name "My Service" --environment prod --environment-type prod

# Skip building and pushing Docker image
omctl deploy --skip-docker-build

# Create an deploy deployment, cloud provider and region
omctl deploy --cloud-provider=aws --region=ca-central-1 --param '{"databaseName":"default","password":"a_secure_password","rootPassword":"a_secure_root_password","username":"user"}'

# Create an deploy deployment with parameters from a file, cloud provider and region
omctl deploy --cloud-provider=aws --region=ca-central-1 --param-file /path/to/params.json

# Create an deploy with instance id
omctl deploy --instance-id <instance-id>

# Create an deploy with resource-id
omctl deploy --resource-id <resource-id>

# Run in dry-run mode (build image locally but don't push or create service)
omctl deploy --dry-run

# Build for multiple platforms
omctl deploy --platforms linux/amd64 --platforms linux/arm64

```

### Options

```
      --cloud-provider string     Cloud provider (aws|gcp|azure)
      --deployment-type string    Type of deployment. Valid values: hosted, byoa (default "hosted")
      --dry-run                   Perform validation checks without actually deploying
  -e, --environment string        Name of the environment to build the service in (default: Prod) (default "Prod")
  -t, --environment-type string   Type of environment. Valid options include: 'dev', 'prod', 'qa', 'canary', 'staging', 'private' (default: prod) (default "prod")
  -f, --file string               Path to the docker compose file (defaults to compose.yaml)
      --github-username string    GitHub username to use if GitHub API fails to retrieve it automatically
  -h, --help                      help for deploy
      --instance-id string        Specify the instance ID to use when multiple deployments exist.
      --param string              Parameters for the instance deployment
      --param-file string         Json file containing parameters for the instance deployment
      --platforms stringArray     Specify the platforms to build for. Use the format: --platforms linux/amd64 --platforms linux/arm64. Default is linux/amd64. (default [linux/amd64])
      --product-name string       Specify a custom service name. If not provided, directory name will be used.
      --region string             Region code (e.g. us-east-2, us-central1)
      --resource-id string        Specify the resource ID to use when multiple resources exist.
      --skip-docker-build         Skip building and pushing the Docker image
```

### Options inherited from parent commands

```
  -o, --output string   Output format (text|table|json) (default "table")
  -v, --version         Print the version number of omnistrate-ctl
```

### SEE ALSO

* [omnistrate-ctl](omnistrate-ctl.md)	 - Manage your Omnistrate SaaS from the command line

