# Architecture

Support Shell is an experimental interactive support console written in Go.

```text
User input -> Shell parser -> Command Registry -> Executor -> Go modules (kubectl)
                  |                 ^
                  +-> AI Agent -----+ (read-only tool calls)
                                    ^
                   Extism/WASM -> host_call (capability-filtered)
```

## Components

- `cmd/support`: entrypoint; registers modules and optional Extism adapter (`-tags extism`).
- `internal/core`: command metadata, Registry and Executor.
- `internal/shell`: terminal editor, completion, history, command dispatch and output rendering.
- `internal/modules`: Go commands including the Kubernetes adapter, using `kubectl` without a shell.
- `internal/agent`: Chat Completions client, context, tools and tracing.
- `pkg/pluginapi`: JSON-serializable versioned plugin manifest and request/result types.
- `plugins/diagnostics`: Rust example compiling to WASI and calling Host capabilities.

The CLI, agent and WASM plugins share command IDs and the Executor. WASM plugins do not receive direct Kubernetes credentials; Host calls are gated by manifest capability declarations and an explicit read-only allowlist.

## Boundaries

The system remains a PoC: no independent RBAC or complete audit trail, mutation approvals, MCP transport, or guaranteed sandbox hardening. Plugin `execute` still returns raw JSON for compatibility; it does not yet use the `pluginapi.Result` envelope.

## Validation

GitHub Actions (`.github/workflows/ci.yml`) runs Go tests/race/vet/build and Rust WASM compilation. This is not an end-to-end Kubernetes test.

## MCP adapter

An optional stdio MCP server (`internal/mcp`) publishes the same read-only Registry commands through JSON-RPC `tools/list` and `tools/call`; it does not execute standalone plugin implementations or bypass Registry risk checks. WASM plugins remain implementation modules with their own Extism Host capabilities. See [mcp.md](mcp.md).

