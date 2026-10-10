# Changelog

## v0.1.0 — first MVP release

### Features
- Interactive Go support shell with commands, completion, history, reverse search, reporting, and Kubernetes context management.
- Shared command registry, argument validation, per-interface read-only tool policies, and YAML/env configuration.
- OpenAI-compatible agent with bounded tool calls, context cancellation, timeout retry and tool-event trace.
- Extism/WASM plugin runtime, Rust plugin SDK and example diagnostics/network plugins.
- Read-only MCP JSON-RPC stdio server exposing registry tools.

### Reliability
- Regression tests for command validation, access policies, agent failures and cancellation, and MCP protocol behavior.
- WASM timeout, interruption/recovery and malformed-output tests with a real compiled guest.
- CLI subprocess and interactive PTY E2E tests, including completion, history, Ctrl+C and context confirmation.
- CI status and Coveralls coverage badges.

### Scope and limitations
- This release is an MVP/PoC. MCP is stdio only; Kubernetes operations depend on a configured kubectl and kubeconfig.
- AI responses and tool outputs must be reviewed before operational decisions.
- Rust/WASM binaries require compatible runtime/host builds; cross-platform release artifacts are not yet verified.
