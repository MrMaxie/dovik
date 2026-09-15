# Why

Dovik already publishes portable native archives, but installation still requires users to select and unpack a platform-specific asset. An npm package can provide one familiar cross-platform install command while preserving the same native binaries and release boundaries.

# What changes

- Publish `dovik` as a small launcher package.
- Publish one optional native package for each supported Windows x64, Linux x64, macOS x64, and macOS arm64 target.
- Expose `dovik` and `dovikd` through npm without claiming or replacing the user's `gh` command.
- Verify exact release assets and locally packed candidates on all four native runners before registry staging.
- Stage subsequent versions through npm trusted publishing with OIDC and provenance, without an npm token.
- Require maintainer 2FA approval for platform packages before the root package becomes public.

# Out of scope

- Changing the native GitHub Release archives or the already published 1.0.0 release.
- Automatically configuring the Dovik daemon, project identity, or the optional `gh` proxy.
- Adding Scoop Main, Homebrew, Winget, installers, signing, or notarization.

# Approval

The user authorized npm publication on 2026-09-15 and directed the subsequent release path to follow Cadder's stage-only trusted publishing model. Commit, push, npm trust settings, GitHub environment changes, and changes to the existing GitHub Release remain separate authorization boundaries.
