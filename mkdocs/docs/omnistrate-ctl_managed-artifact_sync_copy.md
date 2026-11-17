## omnistrate-ctl managed-artifact sync copy

Copy and optionally sign a MANUAL managed artifact sync

### Synopsis

Fetch the server-authored copy plan, copy every versioned artifact to the persisted destination prefix, optionally invoke customer vetting and signing executables, and publish final tags only after both hooks succeed.

Hooks run directly without a shell, receive one JSON object on stdin, and inherit AWS_PROFILE, AWS_REGION, and AWS_DEFAULT_REGION for the selected destination. The JSON includes artifactKey, sourceRef, destinationRef, destinationDigest, destinationRegion, and awsProfile. The vetter runs before the signer while only a staging tag is present; canonical and immutable version tags are published only after both hooks succeed.

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

