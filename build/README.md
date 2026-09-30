# Runtime zlib security fix

Both published runtime images use `build/zlib.sh` to build a temporary zlib
package. As of September 30, 2026, Alpine 3.24 still distributes `1.3.2-r0`,
which is affected by CVE-2026-85091. An `apk upgrade` alone cannot fix it.

The script pins upstream commit
`df84af25dc1942490e1d1c899a07619152a46148`, the maintainer's September 17 fix for
the non-blocking `gzwrite` buffer overflow, and verifies the source archive's
SHA-512 checksum before compiling. This is an **unreleased upstream snapshot**,
not a new stable release. Its upstream library version is `1.3.2.1-motley`;
the local apk version `1.3.2.1_git20260917-r0` records the snapshot date.

The builder runs upstream `make check` and creates an architecture-specific
package containing only the runtime shared library and license. Installing it
through apk removes the old library and preserves file ownership, dependencies,
and source-commit metadata. The local unsigned package is installed with
network access disabled; repository package installation still verifies normal
Alpine signatures. No compiler, source archive, or local apk is shipped in the
runtime image. The builder and runtime stages stay on Alpine 3.24 for ABI and
apk-format compatibility.

The security scan, package, and release workflows refresh the runtime `app`
stage so cached package-upgrade layers cannot hide newly available OS fixes.
The Grype high/critical threshold and reporting remain unchanged.

## Removing the workaround

Once Alpine provides a stable zlib package containing this fix:

1. Remove `zlib-builder` and the local package installation from both runtime
   Dockerfiles, retaining the OS package upgrades.
2. Remove `build/zlib.sh` and its package-workflow path filters.
3. Build both images for `linux/amd64` and `linux/arm64`, rerun Grype, and check
   the CLI version, curl/jq/CA bundle, nginx configuration, HTTP serving, and
   gzip compression. Keep the fresh runtime-stage build policy.
