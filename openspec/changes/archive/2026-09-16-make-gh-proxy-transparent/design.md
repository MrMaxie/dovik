# Approach

Separate ordinary proxy-level execution from governed session execution at the CLI boundary.

An ordinary configured `gh` invocation has no `DOVIK_SESSION` and no isolated agent endpoint. The proxy obtains the selected persona credential through the existing bounded Git credential operation, then starts the original GitHub CLI directly in the caller's process context. It forwards the exact arguments, stdin, stdout, stderr, terminal interaction, and exit code. GitHub CLI remains responsible for parsing commands and flags. Dovik does not require a command to appear in its catalogue, so aliases, extensions, newly introduced commands, and other unknown invocations pass through.

Before acquiring credentials, the ordinary proxy rejects only authentication operations that directly export a token or mutate GitHub CLI authentication state: token export, login, logout, refresh, account switching, setup-git, and `auth status --show-token`. Safe inspection such as `gh auth status` passes through. The rejection remains a product boundary, not an attempt to sandbox another process owned by the same user.

An invocation with `DOVIK_SESSION` or `DOVIK_AGENT_ENDPOINT` remains governed by the existing closed operation catalogue and project policy. Unknown commands, aliases, extensions, arbitrary API access, credential helpers, and authentication operations remain unavailable in that isolated channel. This preserves the real security boundary while removing restrictions that only broke the transparent local alias.

# Compatibility audit

The installed GitHub CLI 2.98.0 command reference reports 35 root command groups and 198 leaf commands, including user-installed aliases and extensions, through `gh help reference`. Dovik's existing catalogue contains 27 entries. The implementation therefore must not grow another version-specific allowlist for ordinary proxy calls.

Tests will exercise every root group from the current reference through the ordinary proxy runner, assert exact argument forwarding, and cover representative interactive and non-interactive commands. A synthetic command absent from the reference will prove future-command passthrough. Separate tests will cover the explicit authentication guard and show that isolated sessions still use the closed catalogue.

# Security and privacy

The credential is retained only in the trusted proxy process environment for the lifetime of the original GitHub CLI child. It is not written to arguments, output, diagnostics, state, or project files. Existing secret filtering remains in place for governed executor output.

Aliases and extensions execute user-installed code with the same OS user's privileges. This does not weaken the documented proxy-level boundary because the same user can already run the original CLI and access their own credential store. They remain forbidden in isolated sessions.

# Trade-offs

Ordinary proxy-level execution is compatibility-oriented, not a same-user sandbox. Policy enforcement is deliberately limited to explicit isolated sessions. A small authentication guard means configured proxy behavior is not byte-for-byte identical to the original CLI for direct token export and authentication mutation, but it preserves project persona integrity and the existing no-token-export contract. User-installed aliases and extensions run with the operator's own privileges and are outside that same-user boundary.
