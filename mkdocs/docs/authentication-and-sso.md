# Authentication and SSO

`omnistrate-ctl` supports password-based authentication, org-bounded API key login, and SSO sign-in using a browser device flow.

## Authentication methods

- Email + password (`--email` and `--password`)
- Email + password from stdin (`--password-stdin`)
- Org-bounded API key (`--api-key`)
- Org-bounded API key from stdin (`--api-key-stdin`)
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

## API key authentication

Org-bounded API keys provisioned by an admin (or root) of your organization can be used in place of a password. The CLI exchanges the plaintext key for a short-lived JWT via the signin endpoint and persists it like any other login session — every subsequent `omnistrate-ctl` call is then authenticated as the api-key's backing user, scoped to the role assigned at create time.

Direct (insecure — appears in shell history and process listings):

```sh
omnistrate-ctl login --api-key om_…
```

Recommended for scripts and CI:

```sh
cat ~/omnistrate_apikey.txt | omnistrate-ctl login --api-key-stdin
echo "$OMNISTRATE_API_KEY"   | omnistrate-ctl login --api-key-stdin
```

The two flags are mutually exclusive with each other and with all other login methods (`--email`, `--password`, `--password-stdin`, `--google`, `--gh`, `--entra`). Passing more than one is rejected by the CLI before any credential is sent.

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

### Microsoft Entra claim requirements

`omnistrate-ctl` sends the Entra device code to Omnistrate; the backend parses
and validates the resulting identity token. The required claims depend on the
Entra application configuration:

| Mode | Required claims | Assurance |
| --- | --- | --- |
| Compatibility | At least one of `email`, `upn`, or `preferred_username`; `xms_edov` absent | Preserves existing applications without verified-domain assurance. |
| Hardened (recommended) | `email` and boolean `xms_edov: true` | Entra asserts that the email domain owner is verified. |

If `email_verified` is emitted, it must be `true`. Explicit false or malformed
`email_verified` and `xms_edov` values are rejected.

`xms_edov` is not emitted by default. Omnistrate operators should add both
`email` and `xms_edov` to the optional ID-token claims of the Entra application
registrations used by CTL. Requesting the `openid email profile` scopes alone
does not enable `xms_edov`.

The compatibility claims are mutable and should not be treated as durable
authorization identifiers. Stable account linking should use Entra `tid` and
`oid` when supported by the backend identity model.
