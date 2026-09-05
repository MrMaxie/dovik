---
impact: minor
visibility: public
components:
  - operator-clients
  - development-workflow
---

## Added

### Start the local daemon from the TUI

On native Windows, an unavailable TUI can explicitly start the colocated Dovik daemon as an independent background process, reconnect when it is ready, and leave it running for later CLI and TUI sessions.
