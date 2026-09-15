# Implementation tasks

## Release contract and command behavior

- [x] 1. Add the shared 1.0.0 version source, successful binary help/version paths, and regression tests.
- [x] 2. Confirm the existing seven MCP tools remain unchanged and identity administration stays outside MCP.
- [x] 3. Add Apache-2.0, the consolidated 1.0.0 changelog, third-party license information, and a concise product README.

## Distribution

- [x] 4. Add exact native package assembly and verification for Windows x64, Linux x64, macOS x64, and macOS arm64.
- [x] 5. Restrict Docker build inputs, preserve UID 10001 and the no-port boundary, and add Linux amd64 image metadata and checks.
- [x] 6. Add the Scoop manifest with `dovik` and `dovikd` shims, checkver, autoupdate, and a release-hash gate.

## Documentation and agent integration

- [x] 7. Add and build the Astro Starlight documentation site at the `/dovik/` base path.
- [x] 8. Add installation, first-process, daily workflow, MCP, identity, platform, security, update, and release-verification documentation.
- [x] 9. Add a sanitized landing-page TUI preview and compare it with the real TUI in ttyglass at desktop and mobile widths.
- [x] 10. Add equivalent versioned Codex and Claude skills that keep normal GitHub and process work transparent, and include them in every native archive.

## Automation and verification

- [x] 11. Add multi-platform CI, documentation build/deploy, exact-SHA candidate, tag-gated release, GHCR, attestations, and public re-download verification workflows.
- [x] 12. Run `just check`, `just test-linux`, `just test-integration`, and `just test-identity-integration` where the required engines are available.
- [ ] 13. Verify native lifecycle, process-tree cleanup, identity accounts, denied peers, container-engine denial, and questionnaire interruption on Windows, Linux, macOS x64, and macOS arm64 runners.
- [ ] 14. Verify Docker and Podman adapters separately; report an unavailable engine without treating it as passed.
- [ ] 15. Build candidate archives on native runners and verify checksums, attestations, permissions, exact file sets, help/version output, and unpacked consumer smoke tests.
- [x] 16. Build and check documentation links and rendered desktop/mobile views.
- [ ] 17. Verify Scoop install, update, and uninstall in a clean Windows environment after the public artifact hash exists.

## Release gates

- [ ] 18. Archive `define-local-identity-profiles` only after its native and container acceptance evidence is complete.
- [ ] 19. Archive this change only after all implementation and candidate verification tasks pass.
- [ ] 20. Before tagging, verify a clean `master` equal to `origin/master`, no active OpenSpec changes, version/changelog alignment, and a green candidate for the exact commit.
- [ ] 21. After separately authorized publication, re-download and verify GitHub Release archives, GHCR tags, Pages, and Scoop without using the checkout.

## Current verification evidence

- `just check`, `just test-linux`, `just test-integration`, and `just test-identity-integration` pass on Windows with Docker available.
- The Docker identity adapter passes. The Podman adapter is explicitly skipped because Podman is unavailable on this host.
- The Windows package verifies its checksum, exact contents, executable versions, and help paths. The unpacked smoke test is reserved for a clean runner because this host already has an unrelated Dovik daemon.
- The runtime image starts as UID 10001, has no exposed ports, runs `dovikd`, and contains the expected OCI version and revision labels.
- Astro type checking, static build, internal link checking, dependency audit, and rendered desktop/mobile review pass.
- Native macOS, disposable-user Windows, native questionnaire interruption, attestations, clean Scoop lifecycle, and public artifact checks remain runner or publication gates.
