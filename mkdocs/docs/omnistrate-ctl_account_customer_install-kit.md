## omnistrate-ctl account customer install-kit

Re-download the BYOC On-Premise install kit

### Synopsis

This command re-downloads the BYOC On-Premise install kit for an existing customer onboarding instance. New BYOC On-Premise customer onboarding instances download the generated install kit during account customer create.

```
omnistrate-ctl account customer install-kit [customer-account-instance-id] [flags]
```

### Examples

```
# Re-download the BYOC On-Premise install kit for a customer onboarding instance
omnistrate-ctl account customer install-kit instance-abcd1234

# Re-download the install kit to a specific path
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

* [omnistrate-ctl account customer](omnistrate-ctl_account_customer.md)	 - Manage customer BYOA account onboarding

