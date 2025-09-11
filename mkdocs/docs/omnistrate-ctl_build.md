## omnistrate-ctl build

Build Services from image, compose spec or service plan spec

### Synopsis

Build command can be used to build a service from image, docker compose, and service plan spec. 
It has two main modes of operation:
  - Create a new service plan
  - Update an existing service plan

Below info served as service plan identifiers:
  - service name (--product-name, required)
  - environment name (--environment, optional, default: Dev)
  - environment type (--environment-type, optional, default: dev)
  - service plan name (the name field of x-omnistrate-service-plan tag in compose spec file, required)
If the identifiers match an existing service plan, it will update that plan. Otherwise, it'll create a new service plan. 

This command has an interactive mode. In this mode, you can choose to promote the service plan to production by interacting with the prompts.

```
omnistrate-ctl build [--file=file] [--spec-type=spec-type] [--product-name=service-name] [--description=service-description] [--service-logo-url=service-logo-url] [--environment=environment-name] [--environment-type=environment-type] [--release] [--release-as-preferred] [--release-description=release-description][--interactive] [--image=image-url] [--image-registry-auth-username=username] [--image-registry-auth-password=password] [--env-var="key=var"] [flags]
```

### Examples

```

# Build service in dev environment
omctl build --product-name "My Service"

# Build service with compose spec in dev environment
omctl build --file compose.yaml --product-name "My Service"

# Build service with compose spec in prod environment
omctl build --file compose.yaml --product-name "My Service" --environment prod --environment-type prod

# Build service with compose spec and release the service with a release description
omctl build --file compose.yaml --product-name "My Service" --release --release-description "v1.0.0-alpha"

# Build service with compose spec and release the service as preferred with a release description
omctl build --file compose.yaml --product-name "My Service" --release-as-preferred --release-description "v1.0.0-alpha"

# Build service with compose spec interactively
omctl build --file compose.yaml --product-name "My Service" --interactive

# Build service with compose spec with service description and service logo
omctl build --file compose.yaml --product-name "My Service" --description "My Service Description" --service-logo-url "https://example.com/logo.png"

# Build service with service specification for Helm, Operator or Kustomize in dev environment
omctl build --spec-type ServicePlanSpec --file spec.yaml --product-name "My Service"

# Build service with service specification for Helm, Operator or Kustomize in prod environment
omctl build --spec-type ServicePlanSpec --file spec.yaml --product-name "My Service" --environment prod --environment-type prod

# Build service with service specification for Helm, Operator or Kustomize as preferred
omctl build --spec-type ServicePlanSpec --file spec.yaml --product-name "My Service" --release-as-preferred --release-description "v1.0.0-alpha"

# Build service from image in dev environment
omctl build --image docker.io/mysql:5.7 --product-name MySQL --env-var "MYSQL_ROOT_PASSWORD=password" --env-var "MYSQL_DATABASE=mydb"

# Build service with private image in dev environment
omctl build --image docker.io/namespace/my-image:v1.2 --product-name "My Service" --image-registry-auth-username username --image-registry-auth-password password --env-var KEY1:VALUE1 --env-var KEY2:VALUE2

```

### Options

```
      --description string                    A short description for the whole service. A service can have multiple service plans.
  -d, --dry-run                               Simulate building the service without actually creating resources
      --env-var stringArray                   Used together with --image flag. Provide environment variables in the format --env-var key1=var1 --env-var key2=var2
      --environment string                    Name of the environment to build the service in (default "Dev")
      --environment-type string               Type of environment. Valid options include: 'dev', 'prod', 'qa', 'canary', 'staging', 'private') (default "dev")
  -f, --file string                           Path to the docker compose file
  -h, --help                                  help for build
      --image string                          Provide the complete image repository URL with the image name and tag (e.g., docker.io/namespace/my-image:v1.2)
      --image-registry-auth-password string   Used together with --image flag. Provide the password to authenticate with the image registry if it's a private registry
      --image-registry-auth-username string   Used together with --image flag. Provide the username to authenticate with the image registry if it's a private registry
  -i, --interactive                           Interactive mode
      --product-name string                   Name of the service. A service can have multiple service plans. The build command will build a new or existing service plan inside the specified service.
      --release                               Release the service after building it
      --release-as-preferred                  Release the service as preferred after building it
      --release-description string            Used together with --release or --release-as-preferred flag. Provide a description for the release version
      --service-logo-url string               URL to the service logo
  -s, --spec-type string                      Spec type
```

### Options inherited from parent commands

```
  -o, --output string   Output format (text|table|json) (default "table")
  -v, --version         Print the version number of omnistrate-ctl
```

### SEE ALSO

* [omnistrate-ctl](omnistrate-ctl.md)	 - Manage your Omnistrate SaaS from the command line

