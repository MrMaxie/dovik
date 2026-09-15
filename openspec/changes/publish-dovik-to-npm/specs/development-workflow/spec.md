## MODIFIED Requirements

### Requirement: Reproducible developer toolchain

The repository SHALL define module `github.com/MrMaxie/dovik` and pin Go `1.27.0`, Node `24.18.0`, just `1.58.0`, and goimports `0.49.0` through mise. The Dovik binaries and native archives SHALL remain independent of Node and npm. The optional npm installation channel MAY use a small Node launcher to select and execute the matching native package.

#### Scenario: Install developer tools

- **WHEN** a developer runs `mise install`
- **THEN** mise installs the pinned Go, Node, just, and goimports versions
- **AND** native Dovik builds and portable archives do not require Node at runtime
- **AND** only consumers who choose the npm installation channel require Node
