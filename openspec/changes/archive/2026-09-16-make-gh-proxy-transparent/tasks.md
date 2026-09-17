# Implementation tasks

- [x] 1. Capture the installed GitHub CLI 2.98.0 command inventory and add compatibility tests covering all 35 root groups and 198 leaf commands.
- [x] 2. Split ordinary proxy-level execution from governed session execution without changing isolated-session policy semantics.
- [x] 3. Run ordinary configured commands directly in the caller's terminal with the selected persona and exact argument, stream, interaction, and exit-code forwarding.
- [x] 4. Add the minimal authentication mutation and direct token-export guard while allowing safe commands such as `gh auth status`.
- [x] 5. Add future-command, alias, extension, interactive command, authentication guard, and isolated-session regressions.
- [x] 6. Align agent guidance and public documentation with transparent ordinary `gh` behavior and governed isolated sessions.
- [x] 7. Verify unit, integration, identity, and Windows terminal behavior without opening additional console windows.
