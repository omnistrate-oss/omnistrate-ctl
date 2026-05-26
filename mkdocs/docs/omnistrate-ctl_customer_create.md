## omnistrate-ctl customer create

Create a customer portal user

### Synopsis

This command creates a customer portal user.

```
omnistrate-ctl customer create [flags]
```

### Examples

```
# Create a customer portal user
omnistrate-ctl customer create --email user@example.com --name "Jane Doe" --password "$PASSWORD" --legal-company-name "Example Inc"

# Create a customer portal user and auto-verify the account
omnistrate-ctl customer create --email user@example.com --name "Jane Doe" --password "$PASSWORD" --legal-company-name "Example Inc" --auto-verify

# Create a customer portal user with the password read from stdin
echo "$PASSWORD" | omnistrate-ctl customer create --email user@example.com --name "Jane Doe" --password-stdin --legal-company-name "Example Inc"
```

### Options

```
      --attribute stringArray       Customer user attribute in key=value format. Can be repeated or comma-separated
      --auto-verify                 Enable automatic verification for the customer user
      --company-url string          Customer company URL
      --email string                Customer user email
  -h, --help                        help for create
      --legal-company-name string   Customer legal company name
      --name string                 Customer user name
      --password string             Customer user password
      --password-stdin              Reads the customer user password from stdin
```

### Options inherited from parent commands

```
  -o, --output string   Output format (text|table|json) (default "table")
  -v, --version         Print the version number of omnistrate-ctl
```

### SEE ALSO

* [omnistrate-ctl customer](omnistrate-ctl_customer.md)	 - Manage customer portal users

