# Implementation tasks

- [x] 1. Define the root launcher and four native package manifests at version 1.0.0.
- [x] 2. Keep `gh` out of npm command shims while retaining the proxy binary in native packages.
- [x] 3. Add native package preparation and isolated package verification tooling.
- [x] 4. Build and verify the current native package and inspect all npm tarball file sets.
- [x] 5. Publish all native packages and the root package under the `candidate` tag.
- [x] 6. Install the exact public version with a fresh cache outside the checkout and verify version, help, and the proxy boundary.
- [x] 7. Promote all five verified packages to `latest`.
- [x] 8. Add the stage-only OIDC workflow, exact release-asset assembly, deterministic packing, manifest checks, and four-platform clean-room verification.
- [x] 9. Commit and push the npm distribution source and workflow to the default branch.
- [ ] 10. Create the protected `npm-production` GitHub environment and configure all five npm packages to trust only `MrMaxie/dovik`, `.github/workflows/npm.yml`, and that environment with staged publishing enabled and traditional tokens disabled.
- [ ] 11. Run the non-staging workflow for `v1.0.0` to verify the checked-in release path without attempting to republish the immutable version.
