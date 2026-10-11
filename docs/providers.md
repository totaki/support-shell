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
 Kubernetes (kubectl) / OpenStack (python-openstackclient) / future PostgreSQL, Kafka
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

Kubernetes connections are discovered from local kubeconfig contexts. Each Shell session
keeps its own active connection, initially adopting the current kubeconfig context. The
optional `SUPPORT_KUBERNETES_CONNECTION` environment variable chooses a different initial
context. A local selection never writes to kubeconfig; resource operations always pass
`kubectl --context <selected>`, including requests from the AI Agent and WASM plugins.

In Shell, use `ctx` (brief active connections), `connections` (all choices), `connections current`, and
`connections use kubernetes/<name>`. Selection asks for `Apply? [y/N]`
but does **not** require infrastructure mutation approval.

The MCP stdio process receives one initial connection selection at startup; use
`SUPPORT_KUBERNETES_CONNECTION` to set it. Per-client MCP sessions and persisted connection
credentials are not yet implemented. There is no global kubeconfig mutation.

OpenStack is implemented in `internal/providers/openstack` as a second built-in
provider. It uses the `openstack` command from `python-openstackclient` and
discovers profiles from `clouds.yaml` (via `OS_CLIENT_CONFIG_FILE` or standard
locations). Profile contents, passwords and tokens are never returned by discovery.

```text
connections
connections use openstack/production
openstack servers
openstack networks
openstack subnets
openstack routers
openstack ports
openstack volumes
```

Cloud operations are read-only and execute with `openstack --os-cloud <name> ... -f json`,
with ambient OS_* authentication variables removed for named profiles to prevent
cross-project credential leakage. If `OS_AUTH_URL` is set, `openstack/env` exposes
the current OS_* environment as an explicit connection. Set `OS_CLOUD` to choose
a default named profile, or `SUPPORT_OPENSTACK_CONNECTION` in `.env` to select
one explicitly. Without either, multiple named clouds require manual selection.

The CLI only validates and selects a cloud; it does not invoke `openstack`
until you execute a cloud operation. Errors from the OpenStack client do not
dump stderr, which can contain credentials. CLI access must have appropriate
OpenStack RBAC; provider policy is not a replacement for server-side authorization.

PostgreSQL and Kafka can be added as separate drivers registered through the same
contract. Their operations and credentials should stay provider-specific.

## Tests

- `go test ./internal/provider ./internal/providers/kubernetes`
- `go test ./...`
- `go test -tags extism ./...`
- `make plugin && make test-e2e` (requires Rust target `wasm32-wasip1`)

WASM E2E tests verify a plugin calling registered provider capabilities through the host,
including denial of calls without a declared capability.
