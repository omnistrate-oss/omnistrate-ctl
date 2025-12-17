## omnistrate-ctl account create

Create a Cloud Provider Account

### Synopsis

This command helps you create a Cloud Provider Account in your account list.

```
omnistrate-ctl account create [account-name] [--aws-account-id=account-id] [--gcp-project-id=project-id] [--gcp-project-number=project-number] [--azure-subscription-id=subscription-id] [--azure-tenant-id=tenant-id] [flags]
```

### Examples

```
# Create aws account
omnistrate-ctl account create [account-name] --aws-account-id=[account-id]

# Create gcp account
omnistrate-ctl account create [account-name] --gcp-project-id=[project-id] --gcp-project-number=[project-number]

# Create azure account
omnistrate-ctl account create [account-name] --azure-subscription-id=[subscription-id] --azure-tenant-id=[tenant-id]
```

### Options

```
      --aws-account-id string          AWS account ID
      --azure-subscription-id string   Azure subscription ID
      --azure-tenant-id string         Azure tenant ID
      --gcp-project-id string          GCP project ID
      --gcp-project-number string      GCP project number
  -h, --help                           help for create
```

### Options inherited from parent commands

```
  -o, --output string   Output format (text|table|json) (default "table")
  -v, --version         Print the version number of omnistrate-ctl
```

### SEE ALSO

* [omnistrate-ctl account](omnistrate-ctl_account.md)	 - Manage your Cloud Provider Accounts

