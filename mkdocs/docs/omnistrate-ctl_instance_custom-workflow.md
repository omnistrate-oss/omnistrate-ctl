## omnistrate-ctl instance custom-workflow

List, describe, and trigger instance custom workflows

### Synopsis

List, describe, and trigger custom workflows supported by an instance, including system workflow-backed actions returned by the custom workflow API.

### Examples

```
# List custom workflows supported by an instance
omnistrate-ctl instance custom-workflow list instance-abcd1234

# Describe a custom workflow by workflow ID or verb
omnistrate-ctl instance custom-workflow describe instance-abcd1234 cwt-12345678
omnistrate-ctl instance custom-workflow describe instance-abcd1234 BACKUP

# Trigger a custom workflow
omnistrate-ctl instance custom-workflow trigger instance-abcd1234 cwt-12345678 --param '{"primaryPodName":"postgres-1"}'

# Trigger a system workflow-backed backup
omnistrate-ctl instance custom-workflow trigger instance-abcd1234 BACKUP
```

### Options

```
  -h, --help   help for custom-workflow
```

### Options inherited from parent commands

```
  -o, --output string   Output format (text|table|json) (default "table")
  -v, --version         Print the version number of omnistrate-ctl
```

### SEE ALSO

* [omnistrate-ctl instance](omnistrate-ctl_instance.md)	 - Manage Instance Deployments for your service
* [omnistrate-ctl instance custom-workflow describe](omnistrate-ctl_instance_custom-workflow_describe.md)	 - Describe a supported instance custom workflow
* [omnistrate-ctl instance custom-workflow list](omnistrate-ctl_instance_custom-workflow_list.md)	 - List custom workflows supported by an instance
* [omnistrate-ctl instance custom-workflow trigger](omnistrate-ctl_instance_custom-workflow_trigger.md)	 - Trigger a supported instance custom workflow

