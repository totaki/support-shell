# Coverage reports

Go unit and integration tests run in GitHub Actions with race detection and produce `coverage.out`. On successful `main` builds, the official `coverallsapp/github-action@v2` uploads the Go profile to [Coveralls](https://coveralls.io/github/totaki/support-shell) using the built-in GitHub Actions token. No manually created coverage secret or OIDC configuration is needed.

## Setup

1. Trigger the project's CI workflow on `main`. The Coveralls GitHub Action can create the repository entry on first successful upload.
2. Open [Coveralls for support-shell](https://coveralls.io/github/totaki/support-shell) to see coverage history, per-file information, and the README badge.
3. If the upload reports a repository permissions/authentication error, sign into Coveralls with GitHub and grant access to the public repository, then rerun CI.

Only successful non-PR builds upload coverage. The Go coverage artifact (`coverage.out`, `coverage.txt`, `go-tests.log`) and workflow summary remain available in GitHub Actions.

The reported percentage covers **Go statements reached by Go tests**; it does not include Rust unit tests or compiled WASM → MCP E2E execution, which are separate CI steps.

## What is tested

Go unit/integration coverage is collected with `go test -race -v -covermode=atomic -coverprofile=coverage.out ./...`. Important suites include:

- `internal/core/runtime_boundaries_test.go`: command ID/path collisions, write approval, missing arguments, primitive schema types and enum restrictions.
- `internal/mcp/protocol_boundaries_test.go`: malformed JSON-RPC requests, notification suppression, protocol errors, unknown tools and failures returned as MCP tool results.
- `internal/mcp/server_test.go`: MCP discovery, successful tool call, policy restrictions and shared argument validation.
- `cmd/support/tool_policy_test.go`: YAML policy parsing and environment deny overrides.

Run `go test ./internal/core ./internal/mcp ./cmd/support` for a focused check. The aggregate Coveralls percentage includes other Go packages; it is not an indication of WASM/Rust coverage or real Kubernetes behavior. No minimum threshold is enforced until we have a confirmed baseline.

- `internal/agent/runtime_boundaries_test.go`: per-turn tool-call budget, suppression of advertised tools after the budget is spent, rollback of conversation history on cancellation, and rejecting schema-invalid model tool calls before handler execution.

## Agent and WASM failure-path checks

- `internal/agent/hardening_test.go`: model HTTP errors and malformed responses roll back unfinished conversation turns; malformed tool-call arguments never reach the handler.
- `internal/agent/retry_test.go`: simulated model HTTP timeout after a tool call causes only the model HTTP request to retry, not a duplicate tool execution.
- WASM host execution checks context cancellation around the plugin call and caps the returned JSON payload at 4 MiB. These guards do not yet prove that every Extism/WASI guest can be preempted immediately while it is running.

- `scripts/e2e_mcp.py` also exercises a real WASM plugin that returns malformed JSON, verifies an MCP tool error, and then calls the same plugin instance again to confirm recovery. This does **not** yet constitute a verified hard-interruption test of a nonterminating WASM guest.

## WASM interruption regression

Extism go-sdk v1.7.1 enables wazero `WithCloseOnContextDone(true)` only when `extism.Manifest.Timeout` is nonzero. The host now sets this timeout from `SUPPORT_PLUGIN_TIMEOUT_MS` (default 10000 ms), and recreates a guest instance after timeout or execution trap because the interrupted module may be closed. `scripts/e2e_wasm_timeout.py` runs an infinite-loop guest under a hard 15-second process watchdog and verifies a subsequent call works. The CI run is the authoritative confirmation that the runtime interruption and recovery work in the built environment.

## MCP and CLI compatibility smoke tests

- `internal/mcp/compatibility_test.go` verifies the MCP initialize handshake, string JSON-RPC IDs, notification suppression, ping, discovery metadata, and structured tool results.
- `scripts/e2e_cli.py` starts the compiled CLI non-interactively with the real WASM fixture, checks help, command listing, JSON output, history, and clean exit. The test uses a temporary config directory to avoid modifying the runner's normal shell history.
- Interactive terminal keystrokes (Tab, Ctrl+R and Ctrl+C) still need a dedicated pseudo-terminal E2E harness; the non-interactive smoke test does not claim to cover those.

## Interactive terminal E2E

`scripts/e2e_pty.py` uses the Python standard-library `pty` module on Linux to test the built CLI as a real terminal. It covers Tab completion, Up-arrow history, Ctrl+R reverse search, Ctrl+C draft cancellation, and rejecting/accepting a Kubernetes context switch. A temporary executable `kubectl` stub and isolated config directory prevent the test from touching any real Kubernetes cluster or user history. The PTY test is part of the CI WASM job after building the Extism-enabled binary.
