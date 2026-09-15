# Approach

The root `dovik` package contains only Node launchers, product metadata, public documentation, and version-matched optional dependencies. Each scoped native package is restricted by npm `os` and `cpu` metadata and contains the three release binaries. The root package maps only `dovik` and `dovikd`; keeping `gh` out of every npm command map prevents an npm install from changing GitHub CLI routing. Release assembly preserves executable modes before deterministic tarballs are created on Linux.

The launcher resolves its native package through Node's package resolver and spawns the selected executable with inherited standard streams and hidden Windows process creation. Native binaries are built with the same `-trimpath` and stripped linker flags as portable release archives.

Version 1.0.0 created the package names through the authenticated npm CLI. That immutable version has no npm provenance. Later publications use a dedicated GitHub Actions workflow, the protected `npm-production` environment, short-lived OIDC credentials, `npm stage publish`, and npm provenance. The workflow verifies GitHub Release checksums and attestations, packs deterministically, and exercises every native target before staging. A maintainer approves the four native packages before the root package with 2FA.

# Trade-offs

The npm channel adds Node as a dependency only for npm consumers. Portable archives remain the zero-Node installation path. The launcher keeps one small parent Node process alive while a native command runs so that terminal input, output, and exit status remain transparent. Provenance cannot be added retroactively to the already published 1.0.0 package version; the trusted path applies to subsequent immutable versions.
