## omnistrate-ctl customer list

List customer portal users

### Synopsis

This command lists customer portal users.

```
omnistrate-ctl customer list [flags]
```

### Examples

```
# List customer portal users
omnistrate-ctl customer list

# List customer portal users as JSON
omnistrate-ctl customer list --output json
```

### Options

```
      --exclude-stats            Exclude user statistics from the response
  -h, --help                     help for list
      --next-page-token string   Token for the next page of results
      --page-size int            Number of users to return per page
```

### Options inherited from parent commands

```
  -o, --output string   Output format (text|table|json) (default "table")
  -v, --version         Print the version number of omnistrate-ctl
```

### SEE ALSO

* [omnistrate-ctl customer](omnistrate-ctl_customer.md)	 - Manage customer portal users

