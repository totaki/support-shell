# Network WASM example

Independent Extism plugin for the Support Shell. The command `network nodes` uses the `k8s.nodes` Host capability, returning structured node addresses and conditions without direct kubeconfig access.

```bash
cargo build --manifest-path plugins/network/Cargo.toml --target wasm32-wasip1 --release
PLUGIN_PATH=./plugins make run-plugin
# then: network nodes
```

The plugin consumes `support-plugin-sdk` as a local Rust path dependency. It has no knowledge of the Go Registry implementation.

## Health findings

Run `network health` to analyze Ready and pressure conditions from a single `k8s.nodes` response. Findings contain node, severity, code, source and condition; coverage counts enumerate all nodes in the response. Missing conditions are **unknown**, not healthy. This does not establish restart history or root cause.

The pure analyzer is in `src/analysis.rs` with fixture-based Rust tests.
