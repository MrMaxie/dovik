# Implementation tasks

- [x] 1. Define and validate personas, project intent, policies, and private atomic persistence.
- [x] 2. Implement the gh command catalogue, account-specific execution, safe input transfer, and stream forwarding.
- [x] 3. Add daemon administration, shared client operations, and revocable restricted sessions.
- [x] 4. Implement native separate-account transports and isolation diagnostics.
- [x] 5. Implement Docker/Podman sessions and a private stdio bridge.
- [x] 6. Deliver CLI, persona import/status, local Git author updates, and Huh questionnaire.
- [x] 7. Integrate identity context and questionnaire access into the TUI; visually verify.
- [x] 8. Document setup, support boundaries, and reversible proxy installation.
- [ ] 9. Run unit, protocol, bypass, integration, Windows/Linux/macOS and Docker/Podman checks; report unavailable environments explicitly.

## Verification evidence

- `just check`: passed on Windows, including Go tests/vet/build, ConPTY harness checks, and strict OpenSpec validation.
- `just test-linux`: passed the Linux Go tests, vet, and build.
- A verbose Linux test executed a real separate-UID client, rejected an incorrect peer, and verified denied access to operator storage.
- `just test-integration`: passed the existing daemon, CLI, persistence, and Linux process lifecycle integration suite.
- `just test-identity-integration`: passed the Docker bridge and bypass tests; Podman was explicitly skipped because its executable was unavailable.
- Synthetic credential tests use a controlled HTTP server to verify concurrent accounts, output/exit preservation, authentication failures, no mutation retries, and account selection through `gh auth token` without `gh auth switch`.
- Repository-local Git credential helper tests verify that HTTPS fetch and push use the configured project persona while direct token export and isolated-session credential requests remain denied.
- ttyglass visually verified the identity workspace, questionnaire choices and final summary, save, and declining save without changing the store. Added a regression test preventing configuration before identity loading completes.
- Native CI now provisions a disposable Windows account and uses an unprivileged macOS account for the separate-account acceptance scenario.
- A real PTY/ConPTY smoke harness now interrupts `dovik project configure` with Ctrl+C and rejects any saved project or identity state.
- macOS ARM64 cross-build passed. This does not establish native macOS behavior.

## Outstanding acceptance evidence

- Run the suite and separate-account scenario on native macOS.
- Run the prepared positive separate-account scenario on a clean Windows runner.
- Run the Podman integration adapter on a prepared engine.
- Run the prepared questionnaire interruption smoke test on clean native terminals. It was not run locally because an unrelated Dovik daemon was already active.

These gaps keep task 9 open. No archival, commit, push, PATH installation, host-account changes, or version bump was performed.
