## omnistrate-ctl login

Log in to the Omnistrate platform

### Synopsis

The login command is used to authenticate and log in to the Omnistrate platform.

```
omnistrate-ctl login [flags]
```

### Examples

```
# Select login method with a prompt
omctl login

# Login with email and password
omctl login --email email --password password

# Login with environment variables
  export OMNISTRATE_USER_NAME=YOUR_EMAIL
  export OMNISTRATE_PASSWORD=YOUR_PASSWORD
  ./omnistrate-ctl-darwin-arm64 login --email "$OMNISTRATE_USER_NAME" --password "$OMNISTRATE_PASSWORD"

# Login with email and password from stdin. Save the password in a file and use cat to read it
  cat ~/omnistrate_pass.txt | omnistrate-ctl login --email email --password-stdin

# Login with email and password from stdin. Save the password in an environment variable and use echo to read it
  echo $OMNISTRATE_PASSWORD | omnistrate-ctl login --email email --password-stdin
```

### Options

```
      --email string      email
      --gh                Login with GitHub
      --google            Login with Google
  -h, --help              help for login
      --password string   password
      --password-stdin    Reads the password from stdin
```

### Options inherited from parent commands

```
  -o, --output string   Output format (text|table|json) (default "table")
  -v, --version         Print the version number of omnistrate-ctl
```

### SEE ALSO

* [omnistrate-ctl](omnistrate-ctl.md)	 - Manage your Omnistrate SaaS from the command line

