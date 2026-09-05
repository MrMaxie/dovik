# Approach

## Preserve peer lifetime and daemon authority

Add a daemon-launch boundary beside the protocol client rather than inside it. The TUI model may request one launch, but registry, lifecycle, status, and output operations continue to flow exclusively through `operatorclient.Client`. The launcher starts the daemon and immediately releases all client-side process resources. It does not wait for the daemon, register it with the Bubble Tea context, forward TUI cancellation, kill it during terminal cleanup, or expose daemon stop and restart operations.

The operating system necessarily records an initiating process at creation time. The product contract is that there is no parent-child lifecycle ownership after launch: the daemon is detached from the TUI terminal and process group, inherits no standard streams, and survives TUI exit, restart, or failure as an independent peer process.

## Limit launch eligibility

Offer `Start daemon` only when all of these conditions hold:

- the initial or retried registry request cannot connect;
- the client uses the native Windows default Named Pipe endpoint;
- no daemon launch or registry retry is already pending; and
- a colocated `dovikd.exe` can be resolved from the running `dovik.exe` distribution.

Do not search arbitrary working directories for the daemon. Production launch resolves the sibling executable from the current `dovik.exe` directory. Repeatable development commands build `dovik.exe` and `dovikd.exe` into one controlled development directory before starting the TUI. The browser harness uses a separate temporary directory for its isolated session and removes it only after its fixture daemon has stopped.

Custom endpoints remain connection-only. The Linux image continues to run `dovikd` as PID 1, so a TUI inside the container retries its existing Unix Domain Socket instead of starting another daemon.

## Start explicitly, then prove readiness through the protocol

Do not auto-start on TUI entry. In the unavailable native Windows state, reuse `s` as the contextual `Start daemon` action because no managed process is selected, and keep retry, details, and quit available. While the launch is pending, serialize the action and show `Starting daemon...` without claiming success from process creation alone.

After a successful process start request, retry the existing registry load through the versioned local protocol for a bounded period. A compatible response is the only readiness signal. If another client wins a concurrent launch race, or the endpoint becomes available between the failed probe and launch action, connect to that daemon and continue without surfacing the losing process's listener error as the primary outcome.

If the executable cannot start or no compatible daemon becomes ready before the deadline, return to the unavailable state with a concise failure and an applicable retry. Keep executable, operating-system, transport, and timeout detail behind the existing diagnostic disclosure.

## Keep verification isolated

Use injected launcher and readiness dependencies for deterministic model tests. Native Windows integration coverage uses unique fixture state and records the fixture daemon out of band so it can prove that the daemon remains alive after the TUI exits or restarts.

`just tui-web` must not start or stop the user's default daemon or read the user's registry during launch verification. Development-tagged code lets the harness inject exact colocated binary, endpoint, and registry locations that are unavailable to ordinary production invocation; passing a normal custom `--endpoint` does not enable daemon launch. The harness, not the TUI, owns fixture cleanup and stops only the daemon created for that isolated session before removing temporary artifacts.

# Trade-offs

- Explicit startup requires one action but avoids surprising process creation merely from opening the TUI.
- Sibling-only executable resolution requires development and distribution tooling to place both binaries together, but avoids executing an unrelated `dovikd` from the working directory or `PATH`.
- Protocol-based readiness cannot expose every early daemon failure, but it preserves the local control-plane contract and avoids making daemon standard output part of the TUI lifecycle.
- Limiting startup to native Windows leaves Linux container operation unchanged and avoids creating a second daemon beside the container's PID 1 entrypoint.
