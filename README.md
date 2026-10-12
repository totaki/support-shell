# Support Shell — runnable reference implementation

> **New agent / maintainer:** read [PROJECT_CONTEXT.md](docs/PROJECT_CONTEXT.md) and [AGENTS.md](AGENTS.md) **first**. This project is a research PoC, not an established support product. The sections below include historical development notes; the linked context and current code take precedence.

[![CI](https://github.com/totaki/support-shell/actions/workflows/ci.yml/badge.svg?branch=main)](https://github.com/totaki/support-shell/actions/workflows/ci.yml)
**MVP release baseline:** [v0.1.0 changelog](CHANGELOG.md) · [Release instructions](docs/releases.md). The Git tag and release must be published separately.


[![Go coverage](https://coveralls.io/repos/github/totaki/support-shell/badge.svg?branch=main)](https://coveralls.io/github/totaki/support-shell?branch=main)

> **Documentation:** [Русский README](README.ru.md) · [Architecture](docs/architecture.md) · [Providers](docs/providers.md) · [Shell commands](docs/shell.md) · [Agent behavior](docs/agent.md) · [WASM plugins](docs/plugins.md)
>
> This README contains historical PoC notes. The documents above describe current architecture and behavior. The agent's adaptive answer depth is prompt guidance, not a deterministic workflow.

Interactive Go shell with a shared command registry, persisted history, ANSI colors, tab completion, infrastructure providers, an OpenAI-compatible tool-calling agent, and an experimental Extism/WASM bridge.

## Current plugin discovery and agent prompt

For Extism-enabled runs, `PLUGIN_PATH` defaults to `./plugins`. The loader finds prebuilt `plugins/*.wasm` and locally compiled Rust `plugins/*/target/wasm32-wasip1/release/*.wasm`; it can load multiple modules. `SUPPORT_PLUGINS` and `SUPPORT_PLUGIN` are supported as fallbacks. Use `make deps` to synchronize `go.mod` and `go.sum`, `make plugin` to build the Rust example, and `make run-plugin` to start Extism Shell. See [plugins](docs/plugins.md).

The agent's embedded system instructions are in [internal/agent/system_prompt.md](internal/agent/system_prompt.md). The prompt encourages proportional tool use, awareness of previous conversation context, fresh verification of live state, and avoiding claims about objects not examined. See [agent](docs/agent.md).

## Run

Requirements: Go 1.23+, `stty` (macOS/Linux) for raw terminal editing; `kubectl` and kubeconfig only for real cluster commands.

```sh
make test
make run
# or go run ./cmd/support --command 'providers'
```

Sample commands:

```
help
commands
providers
ctx
connections
k8s contexts
k8s nodes
k8s pods default
k8s events default
k8s pod describe default coredns-abc
k8s pod logs default coredns-abc --tail=100 --since=15m
history
agent reset
exit
```

`k8s ...` commands perform real *read-only* `kubectl` calls with an explicit session-selected kubeconfig context. OpenStack is a second built-in provider using `python-openstackclient`. There is no shell command substitution, and `kubectl` is executed without invoking a system shell. External credentials are provided only to the trusted Go module, not the WASM plugin. Read operations may still retrieve sensitive information, so use a least-privilege kubeconfig.

Shell editing: `↑↓` history, `←→` cursor movement, `Tab` command/resource completion, `Ctrl+A/E` Home/End, `Ctrl+R` reverse history search, `Backspace/Delete`, `Ctrl+C` cancels editing or running commands (use `exit` to leave the shell). Colors are disabled by `NO_COLOR=1`. History is persisted under the user's configuration directory (usually `~/.config/support-shell/history` on Linux); do not enter secrets as commands.

## Agent configuration

Set a Chat Completions-compatible API endpoint (function/tool calling support required):

```sh
export SUPPORT_API_KEY='YOUR_KEY'
export SUPPORT_API_BASE='https://api.openai.com/v1'
export SUPPORT_MODEL='gpt-4.1-mini'
make run
```

An input that is not a registered command or shell directive goes to the agent automatically. The agent discovers **read-only commands only** and can invoke them; it cannot apply changes. Requests matching a known command with invalid arguments remain CLI errors. `agent reset` clears agent conversation context. Tool rounds are capped at 8, HTTP calls time out, and model responses are bounded in size. The configured provider receives the user's agent prompts, conversation and tool outputs. Never pass credentials as prompts. Models/providers that don't implement OpenAI-style tool calling won't work.

## Extism / WASM plugin (experimental, not runtime-verified here)

Requires Rust, `rustup`, the `wasm32-wasip1` target, and the Extism Go SDK (plus its supported native runtime prerequisites). Build instructions:

```sh
rustup target add wasm32-wasip1
cd plugins/diagnostics && cargo build --target wasm32-wasip1 --release && cd ../..
go get github.com/extism/go-sdk
SUPPORT_PLUGIN=plugins/diagnostics/target/wasm32-wasip1/release/support_diagnostics.wasm go run -tags extism ./cmd/support
# diagnose cluster prod-01
```

The plugin exports `describe`, `execute`, `complete`. Its host callback may call only explicitly allowlisted Kubernetes read-only commands and still goes through the registry's read-only checks. Plugin network and filesystem capabilities are not granted in the intended manifest; this is a proof of concept, **not a hardened production sandbox**. The Extism adapter is also compiled by CI with `-tags extism`. Full runtime end-to-end tests still require running a WASM plugin.

## Architecture

- `internal/core`: registry, command descriptors, input parser, read-only executor.
- `internal/provider`: provider registry and session-scoped connections; `internal/providers/kubernetes` and `internal/providers/openstack`: built-in read-only drivers.
- `internal/shell`: interactive ANSI REPL, input editor, history, agent fallback.
- `internal/agent`: OpenAI-compatible Chat Completions client, tool calls, conversation.
- `cmd/support/plugins_extism.go`: optional Extism host adapter behind Go build tag `extism`.
- `plugins/diagnostics`: sample Rust WASM extension that invokes the Go host API.

## Current limitations (intentional)

- Mutation is denied, including direct shell commands. Safe approvals, RBAC and audit trail must be designed before allowing destructive changes.
- CI includes Rust/WASM and MCP end-to-end tests; real Kubernetes/OpenStack integration and production security are not verified by CI.
- Kubernetes completions resolve namespaces and pod names dynamically using bounded read-only kubectl calls. An interactive completion menu is available.
- Commands have positional arguments and simple `--name=value` flags but not a full declarative JSON Schema / quoted-argument parser.
- MCP stdio server exists (`support mcp serve`), but no per-client remote auth, HTTP transport or dynamic connection selection.
- The agent uses the current Registry and conversation, but it does not automatically receive complete prior direct-shell command outputs. Full shared session context is a future extension.
- Shell's editor relies on `stty` and isn't a cross-platform terminal abstraction.

## Kubernetes contexts

```text
k8s contexts             # current context, name, cluster, namespace as a table
connections use kubernetes/<name>   # switch this Shell session, prompts Apply? [y/N]
Answer y + Enter to apply, or press Enter to cancel; the prompt stays `support ❯`, and `ctx` shows the active provider connections
```

`Tab` completes `connections use kubernetes/<name>` from the local kubeconfig. Connection selection only affects this Support Shell session; no `kubectl config use-context` is executed. Provider calls use an explicit `--context`, including tool calls made by the AI agent. The Agent conversation is reset after switching connections.

## Kubernetes display and completion

The shell renders `k8s nodes`, `k8s namespaces`, `k8s pods [namespace]`, `k8s events [namespace]` as compact tables. Results remain structured JSON internally for the agent and plugins. `k8s pod describe <namespace> <pod>` shows kubectl describe text. `k8s pod logs <namespace> <pod> [--tail=100] [--since=15m] [--container=NAME]` prints raw log text (bounded to 1 MiB per command). Default namespace for pods and events is `default`.

Press `Tab` after `k8s pod logs ` to list namespaces; after `k8s pod logs platform ` to list pods in that namespace; after the pod to list supported flags. Completion needs working `kubectl` authentication and access to list namespaces/pods. The input is still simple whitespace-tokenized CLI (no shell quoting). Large log streams / `--follow` are not yet supported.

## WASM smoke test

The example plugin in `plugins/diagnostics` exposes `diagnose k8s <namespace>` and invokes explicitly granted Kubernetes read capabilities through the Extism host function `host_call`. The separate `plugins/fixture` module provides deterministic test coverage without live Kubernetes credentials.

```sh
rustup target add wasm32-wasip1
make plugin
go get github.com/extism/go-sdk
make run-plugin
# In the shell: diagnose k8s kube-system
```

**Not end-to-end verified in this environment:** Rust/Cargo and Extism dependencies are unavailable here. Check the Extism SDK version and native runtime requirements on the build machine. A build failure means the integration is not yet proven; regular `make run` does not load WASM.

### Interactive completion menu

Press `Tab` to extend the common command prefix. When multiple matches remain,
press `Tab` again to open a navigable completion menu. Within the menu use
`↑`/`↓` or `Tab` to select, `Enter` to insert, and `Esc` to cancel.
Dynamic namespace and pod suggestions are retrieved from the active Kubernetes context.

### OpenCode Go

```sh
export SUPPORT_API_KEY='YOUR_OPENCODE_GO_KEY'
export SUPPORT_API_BASE='https://opencode.ai/zen/go/v1'
export SUPPORT_MODEL='glm-5.3-flash'
make run-plugin
```

The agent sends a stable `x-opencode-session` UUID and `User-Agent: support-shell/0.5` on every model request. `agent reset` clears conversation history and rotates the session ID. OpenCode Go is intended for coding-agent style traffic; follow the provider's acceptable-use requirements.


## v0.6: live diagnostics and tool tracing

Run `make plugin && make run-plugin` (requires Rust wasm32-wasip1 and Extism Go SDK).
`diagnose k8s <namespace>` calls read-only `k8s.pods` and `k8s.events` through a restricted Host API. Output provenance names kubectl as the single source. Demo `diagnose cluster` is replaced; previous demo-only WASM builds must be rebuilt. The LLM does not receive `demo.*` tools.

Agent tool calls are now printed with command ID, input, duration and status. Agent responses have simple inline Markdown styling stripped. Real kube access requires local kubectl and credentials; no real cluster was available to verify. Do not use unrestricted credentials with third-party LLMs: command results are sent to the configured model provider.

### OpenCode Go / slow model responses

Agent HTTP requests default to **180 seconds** (previously 65). Override before starting:

```bash
export SUPPORT_AGENT_TIMEOUT_SECONDS=240
make run-plugin
```

Read-only tool outputs are capped at 6,000 Unicode characters **per tool response** when sent to the LLM. Truncation is explicitly identified; direct Shell command results are unaffected. This is an interim limit, not full pagination or token-aware context budgeting. A model request that times out is retried once with the same session ID and request body. This cannot guarantee provider-side idempotence, but does not repeat local tool executions. Model requests may still time out if OpenCode Go is slow or the conversation grows too large; `agent reset` clears history.


## v0.8 Agent diagnostics

- `report`: show last completed agent response.
- `report save`: write the report to the user config directory `support-shell/reports` (mode 0600). **Reports may contain sensitive operational data**; review before sharing.
- Tool trace uses compact `[namespace=... pod=...]` formatting.
- `SUPPORT_AGENT_MAX_CALLS` (default 16, 1..100) limits tool calls **per question**.
- `SUPPORT_AGENT_MAX_ROUNDS` (default 8, 1..20) limits LLM rounds per question.
- Model prompt distinguishes verified observations from hypotheses and encourages checking identified anomalies. This is guidance, not guaranteed autonomous verification.
- Selected kubeconfig context is included in model prompt. Every actual `kubectl` request explicitly specifies the session-selected kubeconfig context via `--context`.
- `agent reset` clears both conversation and last report.

Agent uses read-only registry tools; Kubernetes API credentials never go into a WASM plugin.

## v0.9 — restart diagnostics and Markdown rendering

The agent now receives terminal-formatted Markdown responses: colored headings and emphasis, readable lists, aligned tables and code blocks. Use `NO_COLOR=1` to disable terminal colors. `report save` continues saving the original, unmodified Markdown.

New read-only Kubernetes commands:

```text
k8s pod restarts kube-system calico-kube-controllers-XXXX
k8s pod logs kube-system calico-kube-controllers-XXXX --previous=true --timestamps=true --tail=200
k8s node describe rnd-master-0
k8s top nodes
k8s top pods kube-system
```

`k8s pod restarts` returns structured PodStatus, last termination timestamps, restart counts, and configured probes. It does **not** calculate restart rate from historical counts. `--previous=true` reads the previous terminated container's logs where Kubernetes still retains them; these may no longer be available. The agent's tools now expose the optional `previous`, `timestamps`, `tail`, `since`, and `container` inputs. Existing WASM diagnostics remains available and can call the new read-only restart operation through the Host allowlist. Diagnose a restart using the actual pod name from `k8s pods kube-system`.

To upgrade an existing checkout, use `update.sh` with the ZIP. Go SDK for Extism is still required for `make run-plugin`. This distribution was verified with `go test ./...` and `go vet ./...`; live Kubernetes and Rust build are not available in the build environment.

## Plugin manager

Run `plugins list` or `plugins info diagnostics` to inspect loaded WASM modules, versions, paths, exposed commands and capabilities. Loading still requires `-tags extism`; without Extism these commands return an empty inventory. See [docs/plugins.md](docs/plugins.md).

Output defaults to a readable view (`plugins list` uses a table, `plugins info NAME` uses detail fields). Use `set format json` for JSON-formatted command results, and `set format table` to switch back. In JSON mode AI answers are wrapped as `{"answer":"..."}`. Command output is visually separated from the next prompt. See [docs/shell.md](docs/shell.md).

Plugin manifests now support optional title, description, author, JSON input/output schemas and examples. CLI discovery (`plugins list`, `plugins info network`, `help network health`) and AI tools share these fields. The current plugin API is `support.shell/v2alpha1`; MCP stdio is implemented (see [docs/mcp.md](docs/mcp.md)). Details: [docs/plugins.md](docs/plugins.md).

### Stdio MCP server

The same Registry now backs CLI, embedded AI and a read-only MCP server. Build with `go build -tags extism -o support ./cmd/support`; configure your MCP client to launch `support mcp serve` with an absolute `PLUGIN_PATH`. WASM plugin tools appear in `tools/list` automatically. See [docs/mcp.md](docs/mcp.md) for setup, safety and limitations.
