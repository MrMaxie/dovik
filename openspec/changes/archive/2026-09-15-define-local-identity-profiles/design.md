# Approach

Keep process supervision intact. Add a daemon-owned identity service and private versioned storage alongside the registry. Extend the shared operator client for administration and use a separate restricted transport for agent sessions. The trusted executor validates an explicit command/flag catalogue, then executes the original gh outside the project with an allowlisted environment and per-call credentials. Never run agent-provided executables, editors, aliases, hooks, or filesystem paths in that executor.

Use Huh v2.0.3 with the existing Bubble Tea v2 stack for the questionnaire. Persona metadata is not authentication proof. Native sessions authenticate OS peers; container sessions use a private runtime stdio bridge. Revocation is checked for each operation, and sessions are never persisted across daemon restarts.

# Trade-offs

Proxy-level is deliberately bypassable by a process with the operator's OS privileges. Isolation requires a separate principal or container boundary. Container agents use Linux even on Windows/macOS hosts. Operator-provisioned accounts support native tools at higher setup cost. Docker/Podman access and privileged container overrides are forbidden inside an isolated session. Customization is limited to validated adapters.

Authentication uses the original gh credential store. Credential acquisition output is private to the executor; credential values must never enter RPC results, process arguments, state files, or diagnostics. Existing project files and process output are untrusted data.
