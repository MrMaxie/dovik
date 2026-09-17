---
impact: minor
visibility: public
components:
  - operator-clients
  - local-identity-profiles
  - product-documentation
---

## Added

### Open Dovik and inspect the current persona faster

Running Dovik without a command now offers help, project identity configuration, and the TUI in an interactive terminal. The new `dovik whoami` command provides a prompt-safe selected-persona status for shell integrations, including a portable Starship recipe.

## Changed

### Keep proxy-level setup focused on transparent identity routing

Project configuration now reserves policy and permission questions for isolated agent sessions while proxy-level projects keep the normal GitHub CLI surface.
