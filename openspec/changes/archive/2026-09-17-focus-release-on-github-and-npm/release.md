---
impact: minor
visibility: public
components:
  - release-distribution
---

## Changed

### Focus supported release channels on GitHub and npm

Publish new Dovik versions as verified native GitHub Release archives and version-matched npm packages. Existing 1.0.0 Scoop and GHCR artifacts remain available, but new releases no longer update those channels.

### Make release automation version-independent

Derive archive names and verification targets from the shared Dovik version so subsequent releases do not depend on manually replacing an earlier version in workflow files.
