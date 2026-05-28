## omnistrate-ctl instance operation

List, describe, and trigger instance custom operations

### Synopsis

List, describe, and trigger custom operations supported by an instance, including system workflow-backed actions returned by the custom operations API.

### Examples

```
# List custom operations supported by an instance
omnistrate-ctl instance operation list instance-abcd1234

# Describe a custom operation by operation ID or verb
omnistrate-ctl instance operation describe instance-abcd1234 cwt-12345678
omnistrate-ctl instance operation describe instance-abcd1234 BACKUP

# Trigger a custom operation
omnistrate-ctl instance operation trigger instance-abcd1234 cwt-12345678 --param '{"primaryPodName":"postgres-1"}'

# Trigger a system workflow-backed backup
omnistrate-ctl instance operation trigger instance-abcd1234 BACKUP
```

### Options

```
  -h, --help   help for operation
```

### Options inherited from parent commands

```
  -o, --output string   Output format (text|table|json) (default "table")
  -v, --version         Print the version number of omnistrate-ctl
```

### SEE ALSO

* [omnistrate-ctl instance](omnistrate-ctl_instance.md)	 - Manage Instance Deployments for your service
* [omnistrate-ctl instance operation describe](omnistrate-ctl_instance_operation_describe.md)	 - Describe a supported instance custom operation
* [omnistrate-ctl instance operation list](omnistrate-ctl_instance_operation_list.md)	 - List custom operations supported by an instance
* [omnistrate-ctl instance operation trigger](omnistrate-ctl_instance_operation_trigger.md)	 - Trigger a supported instance custom operation

