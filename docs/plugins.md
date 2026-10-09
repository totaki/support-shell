# Extism / WASM plugins

Support Shell loads a WASI plugin when built with `-tags extism` and `SUPPORT_PLUGIN` points to a WASM file. Its `describe()` response is validated as `pluginapi.Manifest` (`support.shell/v1alpha1`).

The plugin exports `describe`, `execute` and `complete`. Command metadata is registered in the shared Registry. `execute` receives a command ID and JSON input.

## Capabilities

A plugin declares requested Host operation IDs in `capabilities`. Host calls are permitted only if **both** requested and present in the built-in read-only allowlist; unknown capabilities prevent loading. This is not yet a general permissions framework or fully hardened sandbox. Credentials live in trusted Go modules.

## Demo

```bash
rustup target add wasm32-wasip1
make plugin
make run-plugin
```

In Shell, run `diagnose k8s kube-system`. The Rust diagnostics module calls `k8s.pods` and `k8s.events` through `host_call` and returns a structured summary with common provenance, not independent evidence.

The new manifest requires `apiVersion`, `name`, `version`, `commands` and declared `capabilities`. A previously built old WASM file must be recompiled. `pkg/pluginapi/README.md` documents the schema.
