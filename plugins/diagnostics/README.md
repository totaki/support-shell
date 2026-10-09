# Diagnostics — Kubernetes workload triage

The `diagnostics` WASM module performs a compact, **read-only** scan of Kubernetes workloads in a specified namespace. It helps an operator quickly identify pods that are not Running, containers with restarts or waiting reasons, and recent Warning events. It does **not** change workloads or infer root causes.

## Commands

`diagnose k8s <namespace>`: analyze pods and events from the chosen namespace. If the namespace is omitted, the plugin defaults to `default`. For example:

```text
diagnose k8s kube-system
set format json
diagnose k8s kube-system
```

The structured result includes `pod_count`, `problematic_pods`, `warning_events`, `errors` and `provenance`. A pod is listed when its phase is not Running, its containers have recorded restarts, or a container is waiting. A nonzero cumulative restart counter is **not** proof of ongoing restarts. Events can expire; an empty event list does not prove a clean history.

## Dependencies and permissions

- Extism WASI module with `support-plugin-sdk`.
- Declared Host capabilities: `k8s.pods`, `k8s.events`.
- Host checks both declared capabilities and an explicit read-only allowlist.
- Both operations use the configured Kubernetes context via the Go Host; their output is one evidence source, not independent corroboration.

## Build and usage

From the repository root:

```bash
rustup target add wasm32-wasip1
make plugin
make run-plugin
# plugins info diagnostics
# diagnose k8s kube-system
```

The default `PLUGIN_PATH=./plugins` detects the compiled WASM in `plugins/diagnostics/target/wasm32-wasip1/release/`. See [the plugin architecture guide](../../docs/plugins.md) for adding other modules.

## Limitations

This is a triage overview, not a substitute for previous-container logs, container exit codes, node conditions, or historical monitoring. Information returned through the Host may contain sensitive operational data; the agent can forward tool output to the configured LLM provider.
