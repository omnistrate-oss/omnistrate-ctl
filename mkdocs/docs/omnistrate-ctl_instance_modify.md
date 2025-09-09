## omnistrate-ctl instance modify

Modify an instance deployment for your service

### Synopsis

This command helps you modify the instance for your service.

```
omnistrate-ctl instance modify [instance-id] [flags]
```

### Examples

```
# Modify an instance deployment
omctl instance modify instance-abcd1234 --network-type PUBLIC / INTERNAL --param '{"databaseName":"default","password":"a_secure_password","rootPassword":"a_secure_root_password","username":"user"}'

# Modify an instance deployment using a parameter file
omctl instance modify instance-abcd1234 --param-file /path/to/param.json
```

### Options

```
  -h, --help                  help for modify
      --network-type string   Optional network type change for the instance deployment (PUBLIC / INTERNAL)
      --param string          Parameters for the instance deployment
      --param-file string     Json file containing parameters for the instance deployment
```

### Options inherited from parent commands

```
  -o, --output string   Output format (text|table|json) (default "table")
  -v, --version         Print the version number of omnistrate-ctl
```

### SEE ALSO

* [omnistrate-ctl instance](omnistrate-ctl_instance.md)	 - Manage Instance Deployments for your service

