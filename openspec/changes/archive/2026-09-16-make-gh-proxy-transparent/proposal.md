# Why

Dovik currently treats every configured `gh` invocation as a governed operation from a closed catalogue. The installed GitHub CLI 2.98.0 command reference exposes 198 leaf commands across 35 root command groups, including user-installed aliases and extensions, while Dovik recognizes only 27 operations. This causes normal commands such as `gh auth status` and newly added GitHub CLI commands to fail even in a proxy-level terminal, despite proxy-level already being documented as bypassable by another process owned by the same OS user.

The ordinary `gh` alias must remain compatible with GitHub CLI instead of requiring agents and users to understand Dovik's internal command catalogue. Restrictive policy enforcement still belongs in explicitly isolated Dovik sessions, where it is an actual boundary.

# What changes

- Run every ordinary proxy-level `gh` command in the caller's terminal with the configured project persona.
- Preserve arguments, standard streams, terminal interaction, exit codes, extensions, aliases, and commands unknown to the installed Dovik version.
- Keep only a small explicit guard for GitHub CLI authentication mutation and direct token-export commands that would escape or expose the selected persona.
- Continue using the closed policy catalogue for native and container agent-isolation sessions.
- Add exhaustive compatibility coverage for the full command surface reported by GitHub CLI 2.98.0 and a regression proving that future unknown commands pass through.
- Keep agent guidance transparent: agents use ordinary `gh` and do not inspect or bypass Dovik routing.

# Out of scope

- Expanding the permissions available to an isolated agent session.
- Allowing direct token export or global GitHub authentication mutation through a configured project proxy.
- Replacing GitHub CLI validation of its own commands and flags.
- Adding a second identity administration surface.

# Approval

The user requested complete GitHub CLI coverage and passthrough for unknown commands on 2026-09-16. Implementation, verification, commit, push, and release remain separate stages under the repository workflow.

**Status:** Implementation Complete
**Completed:** 2026-09-16
