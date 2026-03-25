## omnistrate-ctl instance adopt

Adopt a resource instance

### Synopsis

Adopt a resource instance with the specified parameters and optional resource adoption configuration.

```
omnistrate-ctl instance adopt [flags]
```

### Examples

```
# Adopt a resource instance with basic parameters
omnistrate-ctl instance adopt --service-id my-service --service-plan-id my-plan --host-cluster-id my-cluster --primary-resource-key my-resource

# Adopt a resource instance with YAML configuration file
omnistrate-ctl instance adopt --service-id my-service --service-plan-id my-plan --host-cluster-id my-cluster --primary-resource-key my-resource --config-file adoption-config.yaml

# Example adoption-config.yaml format:
resourceAdoptionConfiguration:
  myRedis:
    helmAdoptionConfiguration:
      chartRepoURL: "https://charts.bitnami.com/bitnami"
      releaseName: "my-redis-instance"
      releaseNamespace: "default"
      username: "admin"
      password: "secretpassword"
      runtimeConfiguration:
        disableHooks: false
        recreate: false
        resetThenReuseValues: false
        resetValues: false
        reuseValues: true
        skipCRDs: false
        timeoutNanos: 300000000000  # 5 minutes in nanoseconds
        upgradeCRDs: true
        wait: true
        waitForJobs: true
  myDatabase:
    helmAdoptionConfiguration:
      chartRepoURL: "https://charts.example.com/postgres"
      releaseName: "my-postgres-instance"
      releaseNamespace: "production"
      runtimeConfiguration:
        disableHooks: false
        recreate: true
        resetThenReuseValues: false
        resetValues: false
        reuseValues: false
        skipCRDs: true
        timeoutNanos: 600000000000  # 10 minutes in nanoseconds
        upgradeCRDs: false
        wait: true
        waitForJobs: false
```

### Options

```
  -f, --config-file string            YAML file containing resource adoption configuration (optional)
  -e, --customer-email string         Customer email for notifications (optional)
  -h, --help                          help for adopt
  -c, --host-cluster-id string        Host cluster ID (required)
  -k, --primary-resource-key string   Primary resource key to adopt (required)
  -s, --service-id string             Service ID (required)
  -p, --service-plan-id string        Service plan ID (required)
  -g, --service-plan-version string   Service plan version (optional)
  -u, --subscription-id string        Subscription ID (optional)
```

### Options inherited from parent commands

```
  -o, --output string   Output format (text|table|json) (default "table")
  -v, --version         Print the version number of omnistrate-ctl
```

### SEE ALSO

* [omnistrate-ctl instance](omnistrate-ctl_instance.md)	 - Manage Instance Deployments for your service

