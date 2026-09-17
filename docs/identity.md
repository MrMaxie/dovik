# Project identity and transparent GitHub CLI

Dovik stores personas and project policies in a private, versioned store beside the daemon registry, outside project checkouts. A persona contains a Git author and a GitHub host/account. It never contains a token. Start the daemon as the operator and authenticate the intended accounts using the original GitHub CLI before configuring a project.

## Configure a project

Run bare `dovik` in the repository and choose `Configure this repository`, use `dovik project configure --root /absolute/project`, or open Projects in the TUI and press Enter on the selected project. A configured repository uses the `Edit this repository` label. Dovik derives the project and GitHub repository from the checkout when possible. The original `gh` executable is global daemon configuration: the first setup asks for it, then every later project reuses it. The questionnaire proposes existing personas and authenticated accounts discovered through the original CLI, then asks for the optional Git author and protection mode. Policy presets, exceptions, and alternative personas appear only for agent-isolation. Canceling leaves the identity store unchanged. Setting Git author information is optional and affects only repository-local `user.name` and `user.email`.

The default protection is `proxy-level`. It selects the configured persona for ordinary GitHub CLI invocations but is not a same-user security boundary. An agent running as the operator's OS account can use the original CLI or access the operator's files. TTY detection only enables the questionnaire; access to the operator IPC channel provides administrative authority. Isolated agents use a separate restricted channel and cannot obtain that authority by changing flags or environment variables.

Choose one explicit preset for isolated sessions:

| Preset | Starting permissions |
| --- | --- |
| `read-only` | Read repository, issues, PRs, and Actions |
| `collaborate` | Read, comment, create, edit, close/reopen, and review |
| `maintain` | All supported repository operations, including merge and Actions |

Persona switching is a separate permission and requires an approved alternative persona on the same GitHub host. Exceptions operate on `read`, `comment`, `create`, `edit`, `close`, `review`, `merge`, `actions`, and `persona`. Credential export and policy administration cannot be granted to an isolated agent.

The TUI keeps `Personas`, `Projects`, and `Processes` visible as counted top-level tabs in dependency order. Use left and right to switch workspaces, up and down to select an entity, `l` to refresh, and Enter for the selected entity's primary action. Processes groups process rows under project headings, separates process, project, state, and command fields, indents their values, and keeps applicable lifecycle actions in a contextual strip above the global shortcuts. Projects and Personas open one focused in-place form and use a visually separate confirmation modal before saving. The Projects workspace lists all eight repository-operation permissions once, using a filled marker for allowed and a hollow marker for blocked, independently from persona switching. The Personas workspace separates each saved persona's public account metadata, Git author fields, and assigned-project list without exposing credentials.

For unattended operator configuration, use `dovik --json project configure --file configuration.json`. The file contains explicit `persona`, `project`, and `ghPath` objects:

```json
{
  "ghPath": "/usr/bin/gh",
  "persona": {
    "id": "work", "name": "Work", "gitName": "Example Author",
    "gitEmail": "author@example.test", "host": "github.com", "account": "example"
  },
  "project": {
    "id": "example", "name": "Example project", "description": "Local development tools",
    "root": "/absolute/project", "repository": "example/project", "persona": "work",
    "alternatives": [], "mode": "proxy-level", "proxyEnabled": true,
    "policy": {"preset": "read-only", "exceptions": {}}
  }
}
```

Use native absolute paths on Windows. This input is an operator command, not an automatically discovered repository policy file. It does not modify Git authors or register process definitions; the interactive questionnaire offers those steps separately.

## Personas and policies

```console
dovik --json identity list
dovik identity put --id work --name Work --git-name "Example Author" --email author@example.test --host github.com --account example
dovik identity status --root /absolute/project
dovik identity status --root /absolute/project --short
dovik identity use --root /absolute/project --id work --apply-git
dovik --json policy set --project example --preset collaborate --deny merge --allow persona
dovik --json policy show --project example
```

`identity status --short` remains available for compatibility. `dovik whoami` is the prompt-oriented command: it prints the selected persona name, `?` for an unconfigured repository or a directory outside Git, and `!` when Dovik cannot answer within 400 milliseconds. It reads local Dovik configuration and does not validate the GitHub token over the network. `dovik --json whoami` returns the corresponding `configured`, `unconfigured`, `not_repository`, or `unavailable` state.

### Show the Dovik persona in Starship

Add a custom module to the Starship configuration:

```toml
[custom.dovik_persona]
command = "dovik whoami"
when = true
require_repo = true
format = "git:[$output]($style) "
```

Starship normally reads `~/.config/starship.toml` on Windows, Linux, and macOS. If `STARSHIP_CONFIG` is set, edit that file instead. The same module works with PowerShell, cmd, bash, zsh, and fish because Starship runs the command through its configured shell and `dovik` resolves from `PATH`. `when = true` enables the module, while `require_repo = true` prevents it from running outside Git repositories. Keep Starship's command timeout enabled so a damaged local setup cannot stall the prompt.

The rendered value is the Dovik persona display name, for example `git:Personal`. `git:?` means that the repository has no Dovik identity. `git:!` means the daemon or session context is unavailable. These values report local routing intent, not live GitHub authentication validity.

Legacy `git-user` import requires a selected JSON file containing an array of `{ "name", "username", "email" }` records:

```console
dovik identity import --file /absolute/personas.json
dovik identity import --file /absolute/personas.json --confirm
```

The first command previews the converted records. `username` initially supplies both the Git author and GitHub account; review that mapping before confirming. Imported IDs are `imported-1`, `imported-2`, and so on. Confirmation is atomic and refuses to overwrite an existing ID. Use `identity put` to supply corrected metadata explicitly. Dovik never discovers or overwrites an existing persona file automatically.

## Proxy behavior

Use `dovik gh -- pr view 123` without changing PATH. Once the proxy is installed, `gh pr view 123` performs the same operation. Successful calls are quiet except for the original CLI's stdout/stderr and retain its exit status. Writes are never retried automatically. The isolated-session executor filters credential bytes from output, including matches split across writes.

If the current repository has no configured project identity, the proxy delegates the invocation unchanged to the original GitHub CLI. It preserves normal authentication, output, and exit behavior and never opens the identity questionnaire. A configured proxy-level repository selects its saved persona without restricting the normal GitHub CLI command surface.

Project configuration installs a repository-local Git credential helper that invokes ordinary `gh`. Normal HTTPS Git commands such as `git fetch` and `git push` therefore use the project's configured persona without changing the globally active GitHub CLI account. The helper accepts only Git's credential `get` protocol for the configured host and optional repository path. Git's `store` and `erase` notifications are acknowledged without changing authentication state. The helper is unavailable inside isolated sessions and does not permit direct `gh auth token` access.

The daemon selects a credential for the configured host and account with the original CLI's [account-specific token command](https://cli.github.com/manual/gh_auth_token), verifies the account, and passes the credential only in the child environment. It never switches the globally active account. The ordinary proxy starts the original GitHub CLI in the caller's terminal and preserves exact arguments, input, output, interactive behavior, and exit status. It passes aliases, extensions, and commands unknown to Dovik through to GitHub CLI. Authentication login, logout, refresh, account switching, setup-git, token export, and `auth status --show-token` remain blocked in a configured project.

Every ordinary command, including interactive commands such as `gh pr create`, remains attached to the operator's current terminal. Dovik selects the configured account while the original GitHub CLI owns its normal prompts, input, output, validation, and exit code. Interactive commands are not available through isolated agent sessions.

Isolated sessions use a closed command subset:

| Area | Operations |
| --- | --- |
| Repository | `repo view` |
| Issues | `list`, `view`, `create`, `comment`, `edit`, `close`, `reopen` |
| Pull requests | `list`, `view`, `diff`, `checks`, `create`, `comment`, `edit`, `close`, `reopen`, `review`, `ready`, `merge` |
| Actions | `run list/view/rerun/cancel`, `workflow list/view/run` |
| REST | Approved `repos/OWNER/REPO` routes for repository, issues, PRs, reviews, workflow and run operations |

Inside an isolated session, unknown commands and flags, arbitrary GraphQL, URL-based repository overrides, auth operations, shell aliases, extensions, editors, and arbitrary programs are rejected before credential acquisition. Explicit `--repo` must match the bound repository. Issue, PR, and run selectors must be numeric; workflow selectors accept an ID or file name. REST calls accept only recognized `--method`/`-X`, literal `--raw-field`/`-f` fields, and output formatting. They do not accept custom headers or file expansion. These restrictions do not apply to ordinary proxy-level terminals.

Create commands require a title and body or `--body-file`. PR creation also requires explicit `--head` and `--base`, preventing [automatic head selection and pushing](https://cli.github.com/manual/gh_pr_create). The client reads body files, transfers at most 1 MiB of contents, and replaces the path with stdin. The daemon never opens an agent-provided body path. It runs gh outside the checkout, with interactive prompts disabled.

The operator can use `dovik policy disable --project example` to disable persona routing for a proxy-level project. `policy enable` restores it. An isolated project cannot disable its execution policy.

## Separate-account isolation

The operator provisions a separate non-administrative account and grants it read/write access to the selected checkout. Do not give that account administrator, sudo, Docker, Podman, or equivalent host-management privileges. Dovik does not create accounts or change their groups.

The operator's gh configuration directory must be private: owned by the operator with mode `0700` on Unix, or a DACL granting access only to the operator, SYSTEM, and Administrators on Windows. Redirected credential files are refused. macOS extended ACL grants are refused. `doctor` inspects metadata without reading or returning credentials.

```console
dovik --json doctor --principal agent-account
dovik --json session create --project example --backend native --principal agent-account
```

Set `DOVIK_AGENT_ENDPOINT` to the returned restricted endpoint in the agent account's terminal. Run `dovik session context` there, then inspect the session with `dovik --json doctor --session SESSION_ID` in the operator terminal. The session remains `awaiting-agent` until a connection passes OS peer authentication and project access checks. On Windows the service impersonates the Named Pipe client and checks its SID and groups. Linux uses socket peer credentials; macOS uses local peer credentials. A client-supplied role or account name is never authentication.

The agent channel permits only gh execution, its own context, and approved persona selection. It cannot administer the daemon or manage development processes. Removing `DOVIK_AGENT_ENDPOINT` does not grant access to the operator's IPC channel or credential store.

```console
dovik session persona --persona alternative
dovik --json session list
dovik --json session revoke --id SESSION_ID
```

Run the first command in the agent terminal; run administration in the operator terminal. Revocation blocks subsequent operations. Policy edits apply to the next operation. Restarting the daemon invalidates every session. Revocation does not undo an operation already accepted by GitHub.

## Docker and Podman sessions

Use a locally available image containing the Linux Dovik binary at `/usr/local/bin/dovik` and its gh proxy on PATH. The repository runtime image includes both; add the intended agent executable in your own image. The image and command are operator choices.

```console
dovik --json doctor --backend docker
dovik session run --project example --backend docker --image local-agent:dev -- agent-command
dovik session run --project example --backend podman --image local-agent:dev -- agent-command
```

The adapters use the runtime's stdio connection and do not publish a Dovik control port. The project is mounted at `/workspace`, the process runs as UID/GID 10001, all capabilities are dropped, and privilege escalation is disabled. Provision checkout permissions for that UID. Images are not pulled automatically. There is no fallback to proxy-level when the runtime, image, bridge, or permissions are unavailable.

Optional `--resource /absolute/source=/resources/name` arguments add read-only resources. Dovik rejects mounts overlapping its private state, operator credentials, executable locations, or known runtime configuration. Mounted trees cannot contain sockets, devices, or named pipes. Arbitrary runtime flags, privileged launchers, host namespaces, and custom launch adapters are not accepted.

`session run` forwards command output and exit status. `--input FILE` or `--input -` transfers up to 1 MiB of command input. It does not allocate an interactive PTY. The bridge is private to this local agent command; the session is revoked and the container removed when the command exits. This is not a general container orchestration interface.

## Reversible PATH installation

Build the proxy into a new, dedicated directory chosen by the operator:

```console
mise exec -- go build -o /absolute/dovik-proxy/gh ./cmd/gh
```

Use `gh.exe` on Windows. Approve the original CLI's absolute path first with `dovik identity gh --path ORIGINAL_GH`. Add the new proxy directory before the original CLI's directory in PATH. Do not replace the original executable or existing `git-user`/`git-users` files. Check command resolution in a fresh terminal. To undo installation, remove the dedicated directory from PATH; keep the original CLI and persona data intact.

## Verification boundary

Native Windows tests exercise the proxy, private storage, CLI, and rejection of incorrect Named Pipe peers. Linux tests also execute a real separate-UID socket client in a disposable container. Integration tests use synthetic credentials and a controlled HTTP server, exercise concurrent account selection, stream preservation, and failed writes without retries. Container integration exercises the real adapter and bridge.

Run `just check`, `just test-linux`, `just test-integration`, and `just test-identity-integration`. The last command tests Docker and Podman when installed and reports missing adapters as skipped. Native macOS runtime verification and a positive separate-account Windows session must be performed on prepared hosts; cross-compilation and incorrect-peer rejection do not establish those guarantees. Keep unavailable platform checks explicit before accepting the OpenSpec change.
