# Project Context — Support Shell

> Handoff for a new developer or AI agent. Updated 2026-10-12. This is the **current intent and status**, not a product roadmap or a claim of production readiness.

## 1. What this project is

Support Shell is a **research PoC / engineering laboratory** written in Go: an interactive infrastructure REPL combining a shared tool registry, built-in provider operations, optional LLM fallback, a local MCP stdio server, and Extism/WASI plugins written in Rust.

Repository: https://github.com/totaki/support-shell  
Related personal lab: https://github.com/cloudboy-tech/cloudboy-tech.github.io

**Important outcome of our exploration:** we deliberately **do not** treat this as a validated support product. An operator already has `kubectl`, `openstack`, shell scripts, documentation and general-purpose LLM clients. A prettier shell or a speculative autonomous incident investigation workflow did not establish a compelling advantage. Keep the project as a useful, runnable experimental reference unless a concrete new hypothesis is agreed upon.

## 2. Original motivations (and what was learned)

1. **Custom infrastructure tools:** generic CLIs do not capture company/product-specific behavior. But simple custom tasks are often solved more cheaply with shell scripts; a new REPL must justify its complexity.
2. **Agent tool integration:** configuring many MCP servers, plugins and credentials separately in every agent is inconvenient. Adding *another built-in agent* does not inherently solve that problem. The promising research direction is a **tool runtime / MCP facade that existing agents consume**, without a new competing agent.
3. **WASM/WASI and Host Functions:** strong personal engineering interest in portable, constrained plugins. A guest should access specific host-provided capabilities instead of receiving kubeconfig, OpenStack credentials or unrestricted shell/network access.

These were **research questions**, not proven market requirements. Failure to find product-market fit is not a bug or a reason to invent features.

## 3. Implementation snapshot

| Layer | Current state |
| --- | --- |
| Go CLI/REPL | Working; editing, history, Tab menu, Ctrl+C cancellation, prompt **`support ❯`** |
| Built-in commands | `help`, `commands`, `ctx`, `connections`, `connections use <provider>/<name>`, `set format table/json`, `plugins list/info`, `report`, `agent reset` |
| Provider framework | `internal/provider`: discovery, operation IDs, session-local selections and read-only capability execution |
| Kubernetes | `internal/providers/kubernetes` uses installed `kubectl`; resource calls specify `--context`; selection does **not** change the user's kubeconfig |
| OpenStack | `internal/providers/openstack` uses `python-openstackclient`, profiles from `clouds.yaml` or explicit `OS_*` environment; read-only server/network/subnet/router/port/volume listings |
| Extism/WASI | Go host behind build tag `extism`; Rust SDK `plugins/sdk`; `diagnostics`, `network`, and test `fixture` guests; API `support.shell/v2alpha1` |
| MCP | Implemented, **stdio JSON-RPC**, `support mcp serve`; read-only `tools/list` and `tools/call`; plugins appear through the Core registry |
| LLM | Optional OpenAI-compatible tool-calling client; unmatched CLI input falls back to agent; not a deterministic investigation engine |
| Policy | Per-interface deny lists/YAML config, bounded read-only calls; approval is **not** generally integrated for mutating operations |
| Output | Tables by default, JSON on request; `ctx` displays active provider connections, not long paths in the prompt |

Core paths: `cmd/support`, `internal/core`, `internal/provider`, `internal/providers/{kubernetes,openstack}`, `internal/shell`, `internal/agent`, `internal/mcp`, `internal/pluginmanager`, `pkg/pluginapi`, `plugins/`, `scripts/`.

Architecture: `docs/architecture.md`, provider behavior: `docs/providers.md`, shell: `docs/shell.md`, MCP: `docs/mcp.md`, plugin contract: `docs/plugins.md`, agent: `docs/agent.md`.

### Responsibility boundaries

- **Providers** own credentials/connectivity and native infrastructure reads; commands are named `provider.<id>.<operation>`.
- **Core** registers and validates operations, enforces interface policy and brokers execution.
- **WASM plugins** implement optional higher-level tools; manifests request explicit **read-only provider capabilities**; the guest does not receive raw provider credentials from the host.
- **Shell, embedded agent and MCP** are interfaces over the same registry. The MCP process currently binds its initial provider connections at startup, not one set per remote client.

## 4. Build / smoke tests

Requirements: Go 1.23+, Rust and `wasm32-wasip1` for WASM examples; macOS/Linux terminal; `kubectl` for real Kubernetes access; `openstack` CLI for OpenStack access.

```bash
make deps
go test ./...
make run                         # REPL without Extism plugins
make plugin                      # build Rust WASM examples
make run-plugin                  # REPL with Extism enabled
go test -tags extism ./...
make test-e2e                    # MCP + compiled WASM fixture
```

Inside the REPL:

```text
support ❯ help
support ❯ ctx
support ❯ connections
support ❯ connections use kubernetes/<kubeconfig-context>
support ❯ k8s nodes
support ❯ connections use openstack/<cloud-name>
support ❯ openstack servers
support ❯ plugins list
```

A standalone MCP invocation uses a binary built with `-tags extism` and `support mcp serve`; see `docs/mcp.md`. Configuration starts from `.env.example` (never commit real secrets).

**CI baseline:** the final run after short-prompt / `ctx` changes passed both Go and Rust/WASM jobs, including interactive PTY tests: https://github.com/totaki/support-shell/actions/runs/38162477026 (commit `28320a0b70e04dd3ad5ad6be072bd663ef91d98f`). This is a historical checkpoint, **not a guarantee about future commits or live infrastructure**.

## 5. Security and verification boundaries

- **Not production hardened.** WASM reduces ambient access through explicitly granted host calls, but overall isolation still depends on Extism/runtime configuration, host-call design, resource limits and permissions. Do not advertise an absolute security sandbox.
- Read-only is **not** synonymous with non-sensitive. Logs, resource descriptions and tool results may disclose operational data to configured LLM providers.
- Credential handling stays on the trusted host side. Use least-privilege kubeconfig, OpenStack RBAC and per-surface tool policy. Do not add generic shell execution to WASM host functions by default.
- Connections in REPL are **session local**, with confirmation and independent Kubernetes/OpenStack selection; the user asked to keep the prompt exactly short (`support ❯`) and use `ctx` for connection inspection.
- No per-client remote MCP identities, central multi-tenant authorization, robust audit system, generic safe write approval workflow, signed plugin supply chain or hot reload yet.
- CI uses mocks/fixtures, **not** a real live Kubernetes/OpenStack environment. Avoid saying those integrations are production verified.

## 6. Decisions and non-goals

- **Do not** turn this PoC into a commercial SaaS or a generic support automation product just to justify existing work.
- Do not start adding PostgreSQL, Kafka, approval managers or incident workflow engines without a specific question to validate.
- A possible *future experiment*, **not a committed roadmap**: remove the need for the built-in LLM by providing a portable WASM tool runtime via a single MCP entry point for external Claude/Codex/OpenCode-like agents.
- The interesting test is whether untrusted, third-party WASM modules can call narrowly scoped host functions **without access to provider secrets or ambient network/filesystem**, and what the residual risk is.
- Backward compatibility with old PoC plugin contracts was explicitly not required. The current manifest/API is `support.shell/v2alpha1`.
- Keep changes focused, runnable and backed by tests; user previously preferred direct commits to `main` when they explicitly request implementation. Do not invent green CI results.

## 7. What a new agent should do first

1. Read **this file**, `docs/architecture.md`, `docs/providers.md`, `docs/plugins.md` and relevant command docs.
2. Inspect actual code and test fixtures before assuming documentation is perfectly current; some historical release/PoC notes remain.
3. Establish whether the task is a research question, a bug, a docs fix, or an actual product requirement. Avoid feature inflation.
4. For code changes, run the Go suite, Extism-tag tests and relevant Rust/PTY/MCP E2E through CI; verify the run for the exact commit before claiming success.
5. When documenting a new experiment, distinguish *hypothesis*, *test conditions*, *observation*, *limitation* and *result*. A negative finding is valid.

## 8. Relationship to Cloudboy

Cloudboy (https://github.com/cloudboy-tech/cloudboy-tech.github.io) is a **separate** personal website and engineering lab about cloud technologies and distributed systems generally. Support Shell can appear there as an honest case study about Extism/WASI, MCP and the limits of building a new REPL—not as the flagship commercial offering or the sole topic of the site. The website has its own `docs/PROJECT_CONTEXT.md`.
