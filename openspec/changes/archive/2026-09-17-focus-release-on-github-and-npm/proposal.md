# Why

Dovik 1.0.0 proved the native archives and npm packages, but its release workflow still publishes and verifies Scoop and GHCR artifacts that are no longer part of the immediate distribution strategy. The workflows also embed 1.0.0 archive names, preventing a safe subsequent release without manual edits.

# What changes

- Make release and candidate archive names derive from the version under test.
- Publish supported native archives and checksums through GitHub Releases.
- Stage the version-matched npm launcher and native packages through the accepted trusted-publishing workflow.
- Stop producing a Scoop manifest and stop publishing a GHCR image in the release workflow.
- Verify GitHub Release assets and npm packages as separate public distribution channels.

These outcomes stay together because the tag-gated workflow must have one complete and internally consistent distribution contract before another public version can be released.

# Out of scope

- Removing existing v1.0.0 release assets or container tags.
- Submitting Dovik to Scoop Main, Homebrew, Winget, or another package repository.
- Adding installers, code signing, or macOS notarization.
- Removing Docker and Podman integration tests from release candidates or CI.
