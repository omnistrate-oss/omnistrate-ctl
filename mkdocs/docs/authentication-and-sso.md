# Authentication and SSO

`omnistrate-ctl` supports both password-based authentication and SSO sign-in using a browser device flow.

## Authentication methods

- Email + password (`--email` and `--password`)
- Email + password from stdin (`--password-stdin`)
- SSO with Google (`--google`)
- SSO with GitHub (`--gh`)
- SSO with Microsoft Entra (`--entra`)

You can also run `omnistrate-ctl login` with no flags and choose interactively.

## Password authentication

Direct password login:

```sh
omnistrate-ctl login --email your_email@example.com --password your_password
```

Recommended for scripts (avoid putting passwords in shell history):

```sh
echo "$OMNISTRATE_PASSWORD" | omnistrate-ctl login --email your_email@example.com --password-stdin
```

## SSO authentication (Google, GitHub, Microsoft Entra)

Start SSO login with one of:

```sh
omnistrate-ctl login --google
omnistrate-ctl login --gh
omnistrate-ctl login --entra
```

The CLI will:

1. Request a device code from the selected identity provider
2. Open the verification page in your browser
3. Show (and copy) a user code
4. Poll Omnistrate until authentication completes
5. Save the Omnistrate JWT token locally

## Microsoft Entra requirements

By default, CTL uses the Omnistrate Entra client ID:

```text
3a09381f-919b-40d5-ac1e-3ad35297a438
```

You can override it if needed:

```sh
export OMNISTRATE_ENTRA_CLIENT_ID="<your-entra-app-client-id>"
```

Optional tenant override (defaults to `common`):

```sh
export OMNISTRATE_ENTRA_TENANT_ID="<tenant-id-or-domain>"
```

## Token behavior

After successful login, CTL stores your Omnistrate JWT token in local auth config and reuses it for future commands.
If a saved token is invalid/expired, CTL clears it and prompts you to log in again.
