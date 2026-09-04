<!-- arcantry:start -->
## Arcantry

Use `arcantry.toml` for shared Arcantry configuration.
Treat configured OpenSpec sources as accepted product and engineering intent.
Use configured todo.txt sources for quick intake and changelog sources for consumer-facing release history.
<!-- arcantry:end -->

## Project scope

- Dovik manages the lifecycle of local development processes owned by local projects.
- Dovik does not provide DNS, HTTPS, reverse proxying, domain management, remote execution, scheduling, or container orchestration.
- Cadder remains an independent optional future integration for endpoint allocation. Do not introduce a Cadder dependency or contract without an accepted OpenSpec change.
- The initial runtime targets native Windows and Linux inside Docker. A containerized daemon may manage only processes inside its own container.

## Sources of truth

- Treat accepted specifications in `openspec/specs/` as current product behavior.
- Treat active changes in `openspec/changes/` as planned behavior until implementation, verification, and archival are complete.
- Use `todo.txt` only for uncommitted future directions that do not belong to an accepted OpenSpec change.
- Keep project-facing artifacts and source code in English.

## Development and release

- Use Go, `mise`, and `just` through the repository configuration.
- Use `just tui-web` as the standard visual verification path when changing the TUI layout, colors, styling, navigation, keyboard interactions, or visual behavior. Inspect the rendered browser terminal and capture a screenshot for visual changes; use the adjacent live diagnostic log when investigating behavior.
- Do not add frameworks or dependencies without a direct use in the current implementation.
- Add Testcontainers only with the first functional Linux runtime integration test and keep it out of unit tests and native Windows process tests.
- Treat `1.0.0` as the SemVer target currently in soak, not as a public stable release.
- While `1.0.0` remains in soak, `master` is the only required branch. Do not create tags, releases, or version bumps without explicit authorization.
