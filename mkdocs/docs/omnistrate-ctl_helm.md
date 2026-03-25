## omnistrate-ctl helm

Manage Helm Charts for your service

### Synopsis

This command helps you manage the templates for your helm charts. 
Omnistrate automatically installs this charts and maintains the deployment of the release in every cloud / region / account your service is active in.

```
omnistrate-ctl helm [operation] [flags]
```

### Options

```
  -h, --help   help for helm
```

### Options inherited from parent commands

```
  -o, --output string   Output format (text|table|json) (default "table")
  -v, --version         Print the version number of omnistrate-ctl
```

### SEE ALSO

* [omnistrate-ctl](omnistrate-ctl.md)	 - Manage your Omnistrate SaaS from the command line
* [omnistrate-ctl helm delete](omnistrate-ctl_helm_delete.md)	 - Delete a Helm package for your service
* [omnistrate-ctl helm describe](omnistrate-ctl_helm_describe.md)	 - Describe a Helm Chart for your service
* [omnistrate-ctl helm list](omnistrate-ctl_helm_list.md)	 - List all Helm packages that are saved
* [omnistrate-ctl helm list-installations](omnistrate-ctl_helm_list-installations.md)	 - List all Helm Packages and the Kubernetes clusters that they are installed on
* [omnistrate-ctl helm save](omnistrate-ctl_helm_save.md)	 - Save a Helm Chart for your service

