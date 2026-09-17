---
impact: patch
visibility: public
components:
  - operator-clients
---

## Fixed

Make the configured `gh` proxy behave like GitHub CLI for ordinary project work, including commands added after Dovik and user-installed aliases or extensions. Keep credential-changing operations protected and retain strict policy enforcement inside isolated agent sessions.
