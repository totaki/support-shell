# Support Shell MCP server (stdio)

The same Go Registry backs the interactive CLI, built-in agent and MCP transport. MCP clients discover read-only commands using `tools/list` and execute them through `tools/call`. A WASM plugin loaded with Extism automatically appears as a tool; it does not need to implement MCP.

## Build and run

```bash
make deps
make plugin
go build -tags extism -o support ./cmd/support
PLUGIN_PATH=./plugins ./support mcp serve
```

The process communicates using **newline-delimited JSON-RPC 2.0 via stdin/stdout**. Never print CLI banners or tool diagnostics to stdout in MCP mode. It accepts `initialize`, `notifications/initialized` (notification, no response), `ping`, `tools/list`, and `tools/call`. Version: `2025-03-26`. It is a basic synchronous stdio server: no HTTP transport, resources, prompts, sampling, subscriptions or live change notifications yet.

## Connecting a client

Configure a stdio MCP server command pointing at the compiled binary, with arguments `mcp`, `serve`. Set `PLUGIN_PATH` to an **absolute directory** of compiled WASM files when working outside the repository, e.g.:

```json
{
  "mcpServers": {
    "support-shell": {
      "command": "/absolute/path/to/support",
      "args": ["mcp", "serve"],
      "env": {
        "PLUGIN_PATH": "/absolute/path/to/support-shell/plugins"
      }
    }
  }
}
```

The exact configuration location and key names depend on the MCP client. The example uses a common stdio-MCP configuration shape.

For a smoke test without external clients:

```bash
printf '%s\n' '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-03-26","capabilities":{},"clientInfo":{"name":"test","version":"1"}}}' '{"jsonrpc":"2.0","method":"notifications/initialized"}' '{"jsonrpc":"2.0","id":2,"method":"tools/list"}' | PLUGIN_PATH=./plugins ./support mcp serve
```

## Security and limitations

Only Registry commands with `risk: read` are listed and callable. Non-read commands, including context switching, cannot be called over MCP. Each WASM module still has declared Host capabilities constrained by the Go allowlist. Tool responses and logs may reveal operational information: launch with least-privilege Kubernetes credentials. `read` here is a policy marker, **not** proof that all third-party plugin code is harmless. Stdio restricts who can connect to the local process; it does not authenticate remote users. Do not expose this over a network without an authenticated transport.

Tools use their Registry IDs as MCP names; `inputSchema` is published from the same metadata used by the embedded AI agent and CLI help. Without explicit schemas, the server creates a minimal object schema from positional argument names. Tool execution errors produce `isError: true` with a text message.

The protocol implementation is intentionally compact; protocol conformance and integration with third-party clients need end-to-end verification.

## Deterministic end-to-end smoke test

From the repository root, run `make test-e2e`. It builds an independent `plugins/fixture` WASM module (no Kubernetes credentials or network access needed), builds the Extism-enabled Go host, and runs `scripts/e2e_mcp.py`. The script launches a real `support mcp serve` subprocess with `PLUGIN_PATH` pointing exclusively to this fixture, then verifies `initialize`, discovery via `tools/list`, execution via `tools/call`, and rejection of an invalid argument type. CI runs the same test. This validates the MCP → Registry → Extism WASM path, not a third-party MCP client's behavior or Host API calls to Kubernetes.

## Per-interface permissions

MCP exposes only read-only Registry commands (demo cluster operations are excluded). To suppress sensitive read tools, set `SUPPORT_DENY_MCP=k8s.pod.logs,k8s.events` before starting `support mcp serve`. The list uses **exact Registry command IDs**, separated by commas. Denied tools disappear from `tools/list` and are also rejected by `tools/call`. The built-in agent and CLI have separate lists: `SUPPORT_DENY_AGENT` and `SUPPORT_DENY_CLI`. Denials do not grant write access; the CLI still requires its explicit approval path for mutating operations. Restart to apply changes. Use least-privilege kubeconfig credentials regardless of this feature.
