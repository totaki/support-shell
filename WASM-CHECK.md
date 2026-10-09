# WASM contract smoke test

The example `plugins/diagnostics/src/lib.rs` is a Rust Extism plugin that describes and executes one read-only command: `diagnose cluster <name>`.

The host limits its Host Function `host_call` to `demo.*` command IDs; the demo executes the built-in `demo.cluster.health`. The WASM plugin gets JSON back, adds the `diagnostic` field, and returns JSON to the common registry. This proves plugin -> Host API -> Go Registry -> plugin return path when actually built and run.

## Build and run on macOS/Linux

```sh
rustup target add wasm32-wasip1
make plugin
go get github.com/extism/go-sdk
make run-plugin
```

Then type `diagnose cluster prod-01`; expect an object containing `diagnostic: Host API response`, `result.cluster: prod-01`, `result.status: Degraded`. If `go get` or build fails, plugin integration is NOT verified; check the installed Extism version and dependencies.

## Limitations

The WASM target and host runtime were not built or tested in the current execution environment (Rust/Cargo and Extism Go SDK are missing). This is source-only example for that part; basic Shell works without WASM and without agent credentials.
