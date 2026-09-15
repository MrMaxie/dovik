# Implementation tasks

- [x] 1. Define and validate personas, project intent, policies, and private atomic persistence.
- [x] 2. Implement the gh command catalogue, account-specific execution, safe input transfer, and stream forwarding.
- [x] 3. Add daemon administration, shared client operations, and revocable restricted sessions.
- [x] 4. Implement native separate-account transports and isolation diagnostics.
- [x] 5. Implement Docker/Podman sessions and a private stdio bridge.
- [x] 6. Deliver CLI, persona import/status, local Git author updates, and Huh questionnaire.
- [x] 7. Integrate identity context and questionnaire access into the TUI; visually verify.
- [x] 8. Document setup, support boundaries, and reversible proxy installation.
- [x] 9. Run unit, protocol, bypass, integration, Windows/Linux/macOS and Docker/Podman checks; report unavailable environments explicitly.

## Verification evidence

- `just check`: passed on Windows, including Go tests/vet/build, ConPTY harness checks, and strict OpenSpec validation.
- `just test-linux`: passed the Linux Go tests, vet, and build.
- A verbose Linux test executed a real separate-UID client, rejected an incorrect peer, and verified denied access to operator storage.
- `just test-integration`: passed the existing daemon, CLI, persistence, and Linux process lifecycle integration suite.
- `just test-identity-integration`: passed the Docker bridge and bypass tests locally.
- Synthetic credential tests use a controlled HTTP server to verify concurrent accounts, output/exit preservation, authentication failures, no mutation retries, and account selection through `gh auth token` without `gh auth switch`.
- Repository-local Git credential helper tests verify that HTTPS fetch and push use the configured project persona while direct token export and isolated-session credential requests remain denied.
- ttyglass visually verified the identity workspace, questionnaire choices and final summary, save, and declining save without changing the store. Added a regression test preventing configuration before identity loading completes.
- Release candidate run `34994009376` for exact revision `5e84cdf5c2baa6a06778d89a8b973d3523a9f6ce` passed separate-account acceptance on a disposable Windows account, Linux, macOS x64, and macOS arm64.
- The same candidate passed the Docker and Podman identity adapters independently.
- Real PTY/ConPTY smoke tests on all native targets interrupted `dovik project configure` with Ctrl+C and rejected any saved project or identity state.
