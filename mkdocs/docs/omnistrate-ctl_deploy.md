## omnistrate-ctl deploy

Build or update a service and deploy or upgrade an instance

### Synopsis

Deploy command is the unified entry point to build (or update) a service and then
deploy or upgrade an instance of that service.

It automatically handles:

  - Building from repository when no spec file is found

  - Building from an Omnistrate spec (such as omnistrate-compose.yaml)

  - Creating or updating the service version

  - Determining deployment type (Hosted or BYOC)

  - Selecting cloud and region

  - Selecting or onboarding cloud accounts for BYOC deployments

  - Collecting instance parameters

  - Launching a new instance or upgrading an existing instance

Main modes of operation:

  - Build from repository and deploy
      Triggered when no spec file is provided and no supported spec is found in
      the current directory. The command detects a Dockerfile, builds an image,
      creates the service, generates the Omnistrate spec, and deploys an instance.

  - Build from Omnistrate spec and deploy
      Triggered when a supported spec (with x-omnistrate metadata) is provided or
      discovered. The command creates or updates the service version and deploys.

  - Upgrade an existing instance
      If --instance-id is provided, deploy builds the service version and upgrades
      the specified instance directly.

Instance selection and deployment:

  - If instances already exist in the target environment, the command can prompt
    to upgrade an existing instance or create a new one.

  - If no instances exist, a new one will be created automatically.

  - When creating a new instance, deploy determines the cloud, region, resource
    (if applicable), BYOA account (if applicable), and any required parameters.

Dry run:

  - With --dry-run, deploy performs full validation and build steps but stops
    before launching or upgrading an instance.

```
omnistrate-ctl deploy [--file=file] [--product-name=service-name] [--dry-run] [--deployment-type=deployment-type] [--spec-type=spec-type] [--cloud-provider=cloud] [--region=region] [--env-type=type] [--env-name=name] [--skip-docker-build] [--platforms=platforms] [--param key=value] [--param-file=file] [--instance-id=id] [--resource-id=id] [--github-user-name=username] [flags]
```

### Examples

```

# Build and deploy using the default spec in the current directory
# Looks for omnistrate-compose.yaml, if no spec file is found, deploy falls back to build-from-repo.
omctl deploy

# Deploy using a specific Omnistrate spec
omctl deploy --file omnistrate-compose.yaml

# Build and deploy with a specific product name
omctl deploy --product-name "My Service"

# Build and deploy to a specific cloud and region
omctl deploy --cloud-provider aws --region us-east-1

# Build and deploy using BYOA (Bring Your Own Account)
omctl deploy --deployment-type byoa

# Build and deploy with instance parameters supplied inline
omctl deploy --param '{"disk_size":"20Gi", "username":"test", "password":"Test@123"}'

# Build and deploy with parameters loaded from a file
omctl deploy --param-file params.json

# Build and upgrade an existing instance
omctl deploy --instance-id inst-12345

# Build from repository but skip Docker build (use pre-built image) and then deploy
omctl deploy --skip-docker-build --product-name "My Service"

# Multi-arch build from repo and deploy
omctl deploy --platforms "linux/amd64,linux/arm64"

```

### Options

```
      --cloud-provider string     Cloud provider (aws|gcp|azure)
      --deployment-type string    Type of deployment. Valid values: hosted, byoa (default "hosted" i.e. deployments are hosted in the service provider account) (default "hosted")
      --dry-run                   Perform validation checks without actually building or deploying
  -e, --environment string        Name of the environment to build the service in (default: Prod) (default "Prod")
  -t, --environment-type string   Type of environment. Valid options: dev, prod, qa, canary, staging, private (default: prod) (default "prod")
  -f, --file string               Path to the Omnistrate spec or compose file (defaults to omnistrate-compose.yaml)
      --github-username string    GitHub username to use if GitHub API fails to retrieve it automatically
  -h, --help                      help for deploy
      --instance-id string        Specify the instance ID to use when multiple deployments exist.
      --param string              JSON parameters for the instance deployment
      --param-file string         JSON file containing parameters for the instance deployment
      --platforms stringArray     Specify the platforms to build for. Example: --platforms linux/amd64 --platforms linux/arm64 (default [linux/amd64])
      --product-name string       Specify a custom service name. If not provided, the directory name will be used.
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

