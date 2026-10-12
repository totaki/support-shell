# Architecture

Support Shell is a Go-based **research PoC**, not a validated operator support product. See [PROJECT_CONTEXT.md](PROJECT_CONTEXT.md) for the original hypotheses, current findings and handoff.

```text
          Operator / external MCP client
                      |
       +--------------+--------------+
       |              |              |
    CLI/REPL      Built-in AI      MCP stdio
       |              |              |
       +--------------+--------------+
                      |
                 Core Registry
          input validation / policies
                      |
       +--------------+----------------+
       |                               |
  Provider Registry             WASM Plugin Host
       |                        Extism / WASI
       |                               |
       +<---- granted read-only -------+
       |
 +-----+-------------------+
 |                         |
 Kubernetes Provider    OpenStack Provider
  kubectl --context      openstack --os-cloud
```

## Components

- `cmd/support` wires built-in providers, Core, optional WASM plugins and interfaces.
- `internal/core` implements command dispatch, input validation, interface policies and the initial approval token mechanism.
- `internal/provider` implements the infrastructure provider contract, registry, operation discovery and a read-only execution boundary for extensions.
- `internal/providers/kubernetes` implements provider-specific Kubernetes operations and session-local kubeconfig context selection.
- `internal/providers/openstack` implements read-only OpenStack operations and named `clouds.yaml` connections.
- `internal/shell` implements REPL, history, completion, terminal rendering and operator interaction.
- `internal/agent` provides model tool calling, execution budget and context handling.
- `internal/mcp` exposes policy-filtered read-only operations via stdio JSON-RPC.
- `pkg/pluginapi` and `plugins/sdk` implement the v2alpha1 WASM plugin contract.

Provider capabilities use stable IDs prefixed with `provider.<provider-name>.`, such as
`provider.kubernetes.pods` and `provider.kubernetes.nodes`. Display paths (`k8s pods`)
are UI commands rather than capability identifiers. Run `providers` or
`providers capabilities` to inspect active drivers.

## Responsibility boundaries

Providers own infrastructure connectivity, contexts, native operations and resource semantics.
Plugins own diagnostics and runbooks. Plugins declare exactly which provider capabilities they need;
WASM host calls go directly to the *read-only* provider execution boundary, never to arbitrary
Core commands. CLI, MCP and AI expose tools via Core and follow its surface policies.

The plugin API version changed to `support.shell/v2alpha1`. This is intentionally breaking;
rebuild old WASM artifacts. There is no compatibility alias for old `k8s.*` operation IDs.

Operator approval is not fully integrated with generic CLI/Agent workflows yet. Kubernetes
and OpenStack connection selections are session-local; selection does not mutate global kubeconfig or clouds.yaml. MCP currently binds
one connection per server process; connection-specific credentials and remote connection
stores are future work.
Neither part should be treated as a completed production security boundary.

See [providers.md](providers.md) for concrete contracts and test commands.

## CI and testing

CI runs Go tests (with race detection and coverage), Rust tests, WASM host builds, CLI/PTY E2E,
MCP end-to-end tests and runaway WASM interruption/recovery. No real Kubernetes or OpenStack
cluster is provisioned by CI. Reports are uploaded to Coveralls.
