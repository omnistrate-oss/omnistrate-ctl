## omnistrate-ctl account customer install-kit

Download the BYOC On-Premise install kit

### Synopsis

This command downloads the BYOC On-Premise install kit for a customer onboarding instance.

```
omnistrate-ctl account customer install-kit [customer-account-instance-id] [flags]
```

### Examples

```
# Download the BYOC On-Premise install kit for a customer onboarding instance
omnistrate-ctl account customer install-kit instance-abcd1234

# Download the install kit to a specific path
omnistrate-ctl account customer install-kit instance-abcd1234 --output-path /tmp/byoc-onprem-install-kit.tar
```

### Options

```
  -h, --help                 help for install-kit
  -p, --output-path string   Output path for the install kit file
```

### Options inherited from parent commands

```
  -o, --output string   Output format (text|table|json) (default "table")
  -v, --version         Print the version number of omnistrate-ctl
```

### SEE ALSO

* [omnistrate-ctl account customer](omnistrate-ctl_account_customer.md)	 - Manage customer account onboarding

