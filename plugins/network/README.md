# Network WASM example

Independent Extism plugin for the Support Shell. The command `network nodes` uses the `k8s.nodes` Host capability, returning structured node addresses and conditions without direct kubeconfig access.

```bash
cargo build --manifest-path plugins/network/Cargo.toml --target wasm32-wasip1 --release
PLUGIN_PATH=./plugins make run-plugin
# then: network nodes
```

The plugin consumes `support-plugin-sdk` as a local Rust path dependency. It has no knowledge of the Go Registry implementation.
