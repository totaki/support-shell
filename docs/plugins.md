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


## Discovery and multiple plugins (PoC)

With `-tags extism`, the loader defaults to `PLUGIN_PATH=./plugins`. If the directory exists, it loads:
- `plugins/*.wasm` (prebuilt/distributed plugins)
- `plugins/*/target/wasm32-wasip1/release/*.wasm` (local Rust builds)

Use `PLUGIN_PATH=/opt/support/plugins` to load a different directory, or `PLUGIN_PATH=/opt/plugins/diagnostics.wasm` for one plugin. A comma-separated list of directories/files is also supported. For migration, `SUPPORT_PLUGINS` and `SUPPORT_PLUGIN` remain fallbacks when `PLUGIN_PATH` is unset. Discovery order is sorted. With the default directory missing or empty, loading silently succeeds with no plugins.

Each WASM module has its own Extism instance and capability map, and its calls into that instance are serialized. All plugin commands join the same registry; duplicate command IDs/paths fail registration and duplicate plugin names are rejected. The host restricts WASM calls to both declared capabilities and a read-only allowlist. This is a **startup-only loader**: no hot reload, signature verification or dependency resolution yet.

```bash
make deps
make plugin
make run-plugin
# or: PLUGIN_PATH=/opt/support/plugins make run-plugin
```

`make plugin` currently builds only the diagnostics example. To add another Rust plugin, place its source under `plugins/<name>`, export `describe/execute/complete`, and compile for `wasm32-wasip1`. Its release WASM is picked up on the next launch. The `extism` Go build tag is necessary.
