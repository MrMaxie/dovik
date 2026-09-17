# Approach

## Keep daemon authority and make the default entrypoint useful

The root command detects an interactive terminal before presenting a Huh selection form. Help remains the first and default choice. Git repository discovery is local; configured-versus-unconfigured labeling is a bounded best-effort query through the existing operator client. Redirected invocations print help and exit successfully without opening a form or contacting the daemon. JSON mode without a command retains its current validation error.

Repository configuration and TUI actions call the existing implementations. If configuration is selected while the native default Windows daemon is unavailable, the CLI may explicitly start the same detached sibling daemon used by the TUI and prove readiness through the versioned local protocol before opening the questionnaire. Help and status inspection never start the daemon. Custom endpoints and non-Windows runtimes remain connection-only.

## Provide a prompt-safe persona query

`dovik whoami` resolves the current Git root, then queries either the operator identity snapshot or the restricted session context. A 400 ms deadline bounds the entire query. Human output is exactly one persona name, `?`, or `!` followed by a newline. Expected status outcomes exit zero so shell prompts remain stable; invalid arguments still exit with status two.

JSON output distinguishes `configured`, `unconfigured`, `not_repository`, and `unavailable` with `state`, nullable `configured`, and nullable `persona` fields. The command reports configured local intent only and performs no GitHub network or token validation.

## Share the terminal presentation and narrow the questionnaire

Move the existing Huh theme construction into the shared terminal-style package and use it for both the root menu and identity questionnaire. Keep the existing graphite, ivory, orange, success, attention, danger, and border colors.

Proxy-level configuration describes transparent persona routing. It asks for identity and optional repository-local Git author settings, but not isolated-agent policy details. Agent-isolation continues to ask for the preset, explicit permission exceptions, and alternative personas. Editing a proxy-level project preserves its stored policy so switching to isolation later does not discard prior intent; a new project starts with the existing read-only policy default.

## Keep workstation setup private

The public documentation contains only a portable Starship example using `dovik whoami` and `require_repo = true`. The local Starship file is edited in place to replace the existing custom helper, while its absolute path, neighboring configuration, and workstation details remain outside project artifacts.

# Trade-offs

- A 400 ms status deadline can temporarily display `!` on an overloaded local daemon, but it prevents prompt stalls and clearly distinguishes unavailable state from an unconfigured project.
- The operator menu performs a bounded status probe for its dynamic label, but the default help action remains available even when the daemon is down.
- Plain output intentionally compresses several unavailable or inconsistent states to `!`; JSON output retains the actionable state category without exposing transport diagnostics in the prompt.
