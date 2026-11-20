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

The destination must be a private ECR registry in the plan's region, and the
selected AWS profile must be able to describe, create, push to, and untag
repositories in that registry account. Source registries are read with the
local Docker credential store. Only OCI registry sources are supported.

Every artifact is copied to a staging tag first. The vetter and signer are
optional customer executables that run directly, without a shell. The vetter
runs for every staged artifact, then the signer runs for every staged artifact.
Each receives one JSON object on stdin with `artifactKey`, `sourceRef`,
`destinationRef`, `destinationDigest`, `destinationRegion`, and `awsProfile`.
The runner also sets `AWS_PROFILE`, `AWS_REGION`, and `AWS_DEFAULT_REGION` for
the hook process, and writes hook output to stderr.

The immutable digest alias and canonical version tag are published only after
both hooks succeed for every artifact; staging tags are then removed. A failed
hook leaves the staging tags available for inspection and returns a non-zero
exit status without publishing final tags. Re-running is safe: artifacts whose
version tag already references the planned digest are skipped, including their
hooks.

This standalone tool does not call the Omnistrate API. After it succeeds, ask
the control plane to verify the destination instead of waiting for periodic
reconciliation:

```sh
omnistrate-ctl managed-artifact sync verify --id spabs-123
```

`omnistrate-ctl managed-artifact sync copy` performs this verification
automatically.

This tool does not manage repository policies. Grant pull access to any other
accounts that must consume the artifacts separately.
