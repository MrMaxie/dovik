# Approach

- Keep the existing tag-gated order: verify source, build native artifacts on matching runners, publish the GitHub Release, verify the downloaded public archives, then let the release event start npm staging.
- Read the version from the shared Go source during CI, candidate, and release jobs. Construct archive names from that value and the matrix target instead of embedding a release number in workflow YAML.
- Retain Docker and Podman tests as acceptance coverage, but remove the GHCR publication job and image verification from the public release path.
- Remove Scoop manifest generation and Scoop installation verification from the public release path. Keep historical 1.0.0 assets and implementation helpers intact so old links and local development workflows remain reproducible.
- Treat GitHub Release archives and npm packages as separate artifacts with separate verification. npm tarballs continue to be assembled only from the published and attested GitHub Release archives.

# Trade-offs

- Existing 1.0.0 Scoop and GHCR artifacts remain available but do not receive new versions. This avoids destructive withdrawal while keeping the supported release surface small.
- Container integration remains tested from source even though a versioned image is no longer published.
- Version discovery adds a small workflow step but removes repeated release-number edits and prevents stale artifact names.
