# Agent entry point — Support Shell

**First read [docs/PROJECT_CONTEXT.md](docs/PROJECT_CONTEXT.md).** It records the project's research motivations, current PoC status, tested implementation and explicit non-goals.

Then inspect [architecture](docs/architecture.md), [providers](docs/providers.md), [plugins](docs/plugins.md), [MCP](docs/mcp.md), and the relevant source code. Do not rely on historical sections of README as an accurate description of the current implementation.

**Key constraints:** This is a research PoC, **not a validated support product**. Keep the terminal prompt `support ❯`, use `ctx` to inspect session-local provider connections, and do not silently mutate kubeconfig. WASM has capability-gated host functions but must **not** be represented as a fully hardened sandbox. MCP stdio exists. External tool results can contain sensitive information.

**Build/check:** `go test ./...`, `go test -tags extism ./...`, `make plugin`, `make test-e2e`; verify the exact commit's CI, including PTY/MCP/WASM checks. Only claim successful verification after the relevant tests finish.

**Project direction:** Avoid speculative feature expansion. New work should validate a clearly stated technical hypothesis or solve a concrete bug.
