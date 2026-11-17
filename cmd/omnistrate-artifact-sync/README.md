# omnistrate-artifact-sync

`omnistrate-artifact-sync` is the standalone, open-source runner for a MANUAL
managed-artifact copy plan. It copies every artifact by digest into the exact
destination repository prefix selected by the control plane, so legacy and
current prefixes are both preserved.

Build it from this repository:

```sh
go build ./cmd/omnistrate-artifact-sync
```

Pass either the `manualCopyPlan` JSON object or the full managed-sync response:

```sh
omnistrate-artifact-sync \
  --plan manual-sync.json \
  --aws-profile customer-production \
  --vetter ./vet-artifact \
  --signer ./sign-artifact
```

The vetter and signer are optional customer executables. They run directly,
without a shell, after each artifact is copied to a staging tag. Each receives
one JSON object on stdin with `artifactKey`, `sourceRef`, `destinationRef`,
`destinationDigest`, `destinationRegion`, and `awsProfile`. The runner also sets
`AWS_PROFILE`, `AWS_REGION`, and `AWS_DEFAULT_REGION` for the hook process.

The vetter runs first and the signer second. The canonical version tag and its
immutable digest alias are published only after both hooks succeed. A failed
hook leaves the staging tag available for inspection and returns a non-zero
exit status without publishing final tags.
