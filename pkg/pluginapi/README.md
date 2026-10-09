# Plugin API (v1alpha1)

The `pkg/pluginapi` package defines versioned, JSON-serializable descriptions of Support Shell extensions.

The Extism loader now validates `describe()` with `Manifest.Validate()` before registering any command. The diagnostics Rust plugin has migrated to the new format.

## describe() example

```json
{
  "apiVersion": "support.shell/v1alpha1",
  "name": "diagnostics",
  "version": "0.1.0",
  "capabilities": ["k8s.pods", "k8s.events"],
  "commands": [{
    "id": "diagnostics.k8s.inspect",
    "path": "diagnose k8s",
    "description": "Inspect Kubernetes pods and events",
    "risk": "read",
    "args": ["namespace"]
  }]
}
```

The `capabilities` list is **not** automatic authorization. Extism Host calls are permitted only when the command is both requested in the manifest and present in the Host's hard-coded read-only allowlist. Unsupported capabilities cause plugin loading to fail.

The legacy manifest without `apiVersion`, `version`, and `capabilities` is no longer accepted. Rebuild the diagnostics WASM plugin after updating.

`Request` and `Result` types are transport-neutral. The existing `execute()` response remains a raw JSON value, not yet wrapped in `Result`, to preserve compatibility with Shell/Agent.

## Testing

```bash
go test ./...
go vet ./...
make plugin
make run-plugin
```

GitHub Actions runs Go tests/race/vet and compiles the Rust WASM example. Full Extism Host integration still requires an end-to-end run with the Extism-tagged binary.
