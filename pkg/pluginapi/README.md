# Plugin API (v2alpha1)

The `pkg/pluginapi` package defines the JSON wire contract for Extism/WASM plugins.
Version `support.shell/v2alpha1` is intentionally incompatible with v1alpha1.

## describe() example

```json
{
  "apiVersion": "support.shell/v2alpha1",
  "name": "diagnostics",
  "version": "0.1.0",
  "capabilities": ["provider.kubernetes.pods", "provider.kubernetes.events"],
  "commands": [{
    "id": "diagnostics.k8s.inspect",
    "path": "diagnose k8s",
    "description": "Inspect Kubernetes pods and events",
    "risk": "read",
    "args": ["namespace"]
  }]
}
```

Capabilities refer to operations in the **Provider Registry**, not arbitrary CLI commands.
The host checks that every requested capability exists and is read-only. A plugin may call only
capabilities declared in its own manifest. Input is validated before provider execution.
Unknown or mutating capabilities are rejected; no direct infrastructure credentials or Core
Executor handle is passed to the WASM guest.

The plugin exports `describe()` and `execute()` through the Extism PDK. The host request
contains a capability ID and JSON input. Results are JSON. Plugins implement domain-specific
analysis and orchestration; provider drivers implement infrastructure APIs and connection logic.

The `plugins/sdk` Rust crate has the same `v2alpha1` manifest version. Rebuild each WASM
artifact after updating the SDK. Existing binaries are not automatically migrated.

## Testing

```bash
rustup target add wasm32-wasip1
go test ./...
make plugin
make test-e2e
```

The E2E suite runs the actual Extism host with a WASM fixture and MCP adapter. Access to a real
Kubernetes installation is not required for the fixture tests.
