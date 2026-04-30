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
omnistrate-ctl login

# Login with email and password
omnistrate-ctl login --email email --password password

# Login with environment variables
  export OMNISTRATE_USER_NAME=YOUR_EMAIL
  export OMNISTRATE_PASSWORD=YOUR_PASSWORD
  ./omnistrate-ctl-darwin-arm64 login --email "$OMNISTRATE_USER_NAME" --password "$OMNISTRATE_PASSWORD"

# Login with email and password from stdin. Save the password in a file and use cat to read it
  cat ~/omnistrate_pass.txt | omnistrate-ctl login --email email --password-stdin

# Login with email and password from stdin. Save the password in an environment variable and use echo to read it
  echo $OMNISTRATE_PASSWORD | omnistrate-ctl login --email email --password-stdin

# Login with an org-bounded API key (insecure; prefer --api-key-stdin)
  omnistrate-ctl login --api-key om_…

# Login with an API key from stdin
  cat ~/omnistrate_apikey.txt | omnistrate-ctl login --api-key-stdin
  echo $OMNISTRATE_API_KEY | omnistrate-ctl login --api-key-stdin

# Login with GitHub SSO
  omnistrate-ctl login --gh

# Login with Google SSO
  omnistrate-ctl login --google

# Login with Microsoft Entra SSO
  omnistrate-ctl login --entra
```

### Options

```
      --api-key string    Org-bounded API key plaintext (om_…)
      --api-key-stdin     Reads the API key from stdin
      --email string      email
      --entra             Login with Microsoft Entra
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

