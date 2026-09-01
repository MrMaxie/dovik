# Tasks

- [ ] Implement project, process definition, runtime, and state-machine models with transition tests.
- [ ] Implement the private per-user registry with atomic persistence and startup reconciliation tests.
- [ ] Implement direct process execution and platform adapters for Windows Job Objects and Linux process groups.
- [ ] Implement controlled start, idempotent stop, confirmed restart, natural-exit observation, and process-tree cleanup.
- [ ] Implement bounded stdout and stderr event capture with sequence, stream, tail, and truncation behavior.
- [ ] Implement the versioned JSON protocol over a current-user Windows Named Pipe and a container-local Unix Domain Socket.
- [ ] Implement CLI commands for projects, process definitions, lifecycle, status, and log tailing.
- [ ] Add Windows acceptance coverage for duplicate prevention, start failure, exit codes, restart, daemon interruption, and tree cleanup.
- [ ] Add Linux container acceptance coverage for lifecycle, output, registry persistence, IPC, and container isolation.
- [ ] Add `github.com/testcontainers/testcontainers-go` v0.44.0 through `go get` with the first functional Linux runtime integration test.
- [ ] Use Testcontainers to build the local image, start `dovikd`, execute `dovik` inside the container without TCP, verify the lifecycle and logs, and always clean resources.
- [ ] Verify output truncation, stdout and stderr attribution, private state handling, and the absence of environment values from list and status responses.
- [ ] Run `just check`, `just test-linux`, Windows smoke tests, and the container integration suite when Docker is available.
- [ ] Review `release.md` against the implemented and verified behavior before archival.
