## omnistrate-ctl managed-artifact sync copy

Copy and optionally sign a MANUAL managed artifact sync

### Synopsis

Fetch the server-authored copy plan and copy every artifact by digest to the persisted destination prefix in the destination ECR registry. Every artifact is copied to a staging tag first; the optional customer vetter then runs for every artifact, followed by the optional signer. Canonical and immutable version tags are published only after both hooks succeed for every artifact, and staging tags are then removed. After a successful copy, the command asks the control plane to refresh the sync status and prints the result; use 'sync refresh-status' to request destination verification again.

Hooks run directly without a shell, receive one JSON object on stdin, and inherit AWS_PROFILE, AWS_REGION, and AWS_DEFAULT_REGION for the selected destination. With --aws-profile, hooks also receive the profile's resolved credentials as AWS_ACCESS_KEY_ID, AWS_SECRET_ACCESS_KEY, and AWS_SESSION_TOKEN in place of any inherited AWS credential variables, so they authenticate as the same identity as the copy. The JSON includes artifactKey, sourceRef, destinationRef, destinationDigest, destinationRegion, and awsProfile. Hook output is written to stderr.

Re-running the command is safe: artifacts whose version tag already references the planned digest are skipped, including their hooks. Source registries are read with local Docker credentials. Repository policies that grant other accounts pull access are not managed by this command.

```
omnistrate-ctl managed-artifact sync copy [flags]
```

### Examples

```
omnistrate-ctl managed-artifact sync copy --id spabs-123 --aws-profile customer-prod --vetter ./vet-artifact --signer ./sign-artifact
```

### Options

```
      --aws-profile string   AWS shared-config profile for the destination account
  -h, --help                 help for copy
      --id string            Private MANUAL sync ID (spabs-*) (required)
      --signer string        Customer signing executable; receives one JSON object on stdin per artifact
      --vetter string        Customer vetting executable; receives one JSON object on stdin per staged artifact
```

### Options inherited from parent commands

```
  -o, --output string   Output format (text|table|json) (default "table")
  -v, --version         Print the version number of omnistrate-ctl
```

### SEE ALSO

* [omnistrate-ctl managed-artifact sync](omnistrate-ctl_managed-artifact_sync.md)	 - Inspect private artifact syncs and public ECR publications

