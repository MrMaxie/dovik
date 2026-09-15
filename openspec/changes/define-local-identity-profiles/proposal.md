# Local identity profiles and governed GitHub CLI

**Change ID:** `define-local-identity-profiles`
**Status:** Implemented locally; platform acceptance evidence incomplete
**Created:** 2026-09-06

## Problem

Developers need consistent Git authorship and GitHub accounts per project, with explicit permissions for coding agents. A PATH wrapper alone cannot prevent a same-user process from bypassing it.

## Outcome

Provide daemon-owned personas, project context, a terminal questionnaire, a governed `gh` proxy, and revocable agent sessions. Default to proxy-level enforcement. Offer container isolation with Docker or Podman and native isolation using an operator-provisioned separate account on Windows, Linux, and macOS. Keep credentials on the trusted execution side.

## Scope

Include identity import and prompt status replacing git-user, project-local Git author settings, presets and operation exceptions, CLI/TUI visibility, a restricted agent channel, and defensive integration tests. Container support is limited to local agent sessions. Do not install system accounts, container engines, PATH shims, or credentials automatically. Do not add general Git interception, remote orchestration, releases, or version changes.

## Acceptance

The questionnaire requires an explicit policy selection. Recognized gh operations preserve output and exit codes and use the selected account without changing the global active account. Agents cannot request credential export or policy administration. Isolation must fail closed and never silently become proxy-level. Each supported OS and isolation adapter requires native integration evidence before being claimed verified.
