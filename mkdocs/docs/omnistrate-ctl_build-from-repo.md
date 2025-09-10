## omnistrate-ctl build-from-repo

Build Service from Git Repository

### Synopsis

This command helps to build service from git repository. Run this command from the root of the repository. Make sure you have the Dockerfile in the repository and have the Docker daemon running on your machine. By default, the service name will be the repository name, but you can specify a custom service name with the --product-name flag.

You can also skip specific stages of the build process using the --skip-* flags. For example, you can skip building the Docker image with --skip-docker-build, skip creating the service with --skip-service-build, skip environment promotion with --skip-environment-promotion, or skip SaaS portal initialization with --skip-saas-portal-init.

For testing purposes, use the --dry-run flag to only build the Docker image locally without pushing, skip service creation, and generate a local spec file with a '-dry-run' suffix. Note that --dry-run cannot be used together with any of the --skip-* flags as they are mutually exclusive.

```
omnistrate-ctl build-from-repo [flags]
```

### Examples

```
# Build service from git repository
omctl build-from-repo

# Build service from git repository with environment variables, deployment type and cloud provider account details
omctl build-from-repo --env-var POSTGRES_PASSWORD=default --deployment-type byoa --aws-account-id 442426883376

# Build service from an existing compose spec in the repository
omctl build-from-repo --file compose.yaml

# Build service with a custom service name
omctl build-from-repo --product-name my-custom-service

# Skip building and pushing Docker image
omctl build-from-repo --skip-docker-build

# Skip multiple stages
omctl build-from-repo --skip-docker-build --skip-environment-promotion

# Run in dry-run mode (build image locally but don't push or create service)
omctl build-from-repo --dry-run

# Build for multiple platforms
omctl build-from-repo --platforms linux/amd64 --platforms linux/arm64

# Build with release description
omctl build-from-repo --release-description "v1.0.0-alpha"

# Build using github token from environment variable (GH_PAT)
set GH_PAT=ghp_xxxxxxxx
omctl build-from-repo
"
```

### Options

```
      --aws-account-id string        AWS account ID. Must be used with --deployment-type
      --deployment-type string       Set the deployment type. Options: 'hosted' or 'byoa' (Bring Your Own Account). Only effective when no compose spec exists in the repo.
      --dry-run                      Run in dry-run mode: only build the Docker image locally without pushing, skip service creation, and write the generated spec to a local file with '-dry-run' suffix. Cannot be used with any --skip-* flags.
      --env-var stringArray          Specify environment variables required for running the image. Effective only when the compose.yaml is absent. Use the format: --env-var key1=var1 --env-var key2=var2. Only effective when no compose spec exists in the repo.
  -f, --file string                  Specify the compose file to read and write to (default "compose.yaml")
      --gcp-project-id string        GCP project ID. Must be used with --gcp-project-number and --deployment-type
      --gcp-project-number string    GCP project number. Must be used with --gcp-project-id and --deployment-type
  -h, --help                         help for build-from-repo
  -o, --output string                Output format. Only text is supported (default "text")
      --platforms stringArray        Specify the platforms to build for. Use the format: --platforms linux/amd64 --platforms linux/arm64. Default is linux/amd64. (default [linux/amd64])
      --product-name string          Specify a custom service name. If not provided, the repository name will be used.
      --release-description string   Provide a description for the release version
      --reset-pat                    Reset the GitHub Personal Access Token (PAT) for the current user.
      --skip-docker-build            Skip building and pushing the Docker image
      --skip-environment-promotion   Skip creating and promoting to the production environment
      --skip-saas-portal-init        Skip initializing the SaaS Portal
      --skip-service-build           Skip building the service from the compose spec
```

### Options inherited from parent commands

```
  -v, --version   Print the version number of omnistrate-ctl
```

### SEE ALSO

* [omnistrate-ctl](omnistrate-ctl.md)	 - Manage your Omnistrate SaaS from the command line

