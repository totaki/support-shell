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
