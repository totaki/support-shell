# Provider-first architecture

Support Shell separates infrastructure access from diagnostic workflows.

```text
CLI / Agent / MCP
        |
   Core Registry
  policy / approval
        |
  Provider Registry ------- WASM plugin host
        |                      |
   Provider operation <--- granted read capability
        |
 Kubernetes (kubectl) / future OpenStack, PostgreSQL, Kafka drivers
```

## Provider responsibilities

A provider owns connectivity, context, infrastructure clients and basic resource operations.
A provider registers a catalog of typed operations (ID, risk, input schema, handler, optional completion).
Core publishes those operations as tools and applies CLI/Agent/MCP policies.
Providers do not implement incident runbooks or AI decisions.

The first provider is `kubernetes`, implemented in `internal/providers/kubernetes`.
Its public operations have IDs like `provider.kubernetes.pods`, `provider.kubernetes.events`,
`provider.kubernetes.nodes` and `provider.kubernetes.context.use`.
CLI paths remain ergonomic (`k8s pods`, `k8s nodes`) and are not stable API identifiers.

`internal/provider` defines `Provider`, `Operation`, `Catalog` and `Registry`.
The catalog is validated on registration: provider ID, operation namespace, unique IDs and risks.
To inspect available providers from the shell, run `providers` or `providers capabilities`.

## Plugin responsibilities

WASM plugins implement diagnosis, analysis and runbooks. Each plugin declares exact provider
capabilities in its manifest. The host validates these grants against the **registered provider
operations**, not a hard-coded command allowlist. Plugins invoke the dedicated read-only
provider execution entrypoint, **never the Core command registry**. Invalid arguments and
mutating capabilities are rejected at this boundary.

The plugin wire API version is `support.shell/v2alpha1`. Older v1alpha1 plugin
manifests are intentionally incompatible. Update and rebuild your Rust plugins against
the current `plugins/sdk`.

## Safety and scope

A read capability is not a guarantee that output contains no sensitive data; credentials,
Pod environment and logs may contain secrets. The provider host remains read-only.
Mutating provider operations remain gated by Core approval and are not grantable to a WASM
plugin. The existing approval workflow is **not yet fully integrated** for generic commands.

The current Kubernetes driver uses the local `kubectl` and kubeconfig, including a
process-global kubeconfig context switch. Multi-connection context isolation is a planned
provider-layer feature, not an implemented guarantee. Do not concurrently share a mutable
kubeconfig with unrelated operators.

OpenStack, PostgreSQL and Kafka will be separate drivers registered through the same
provider contract. Their operations and credentials should stay provider-specific; avoid
inventing universal CRUD semantics across fundamentally different APIs.

## Tests

- `go test ./internal/provider ./internal/providers/kubernetes`
- `go test ./...`
- `go test -tags extism ./...`
- `make plugin && make test-e2e` (requires Rust target `wasm32-wasip1`)

WASM E2E tests verify a plugin calling registered provider capabilities through the host,
including denial of calls without a declared capability.
