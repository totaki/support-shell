#!/usr/bin/env python3
"""End-to-end stdio MCP test using the compiled Extism fixture plugin."""
import json
import os
import subprocess
import sys

binary = os.environ.get("SUPPORT_BINARY", "./support")
env = dict(os.environ, PLUGIN_PATH="plugins/fixture/target/wasm32-wasip1/release/support_fixture.wasm")
messages = [
    {"jsonrpc": "2.0", "id": 1, "method": "initialize",
     "params": {"protocolVersion": "2025-03-26", "capabilities": {},
                "clientInfo": {"name": "e2e", "version": "1"}}},
    {"jsonrpc": "2.0", "method": "notifications/initialized"},
    {"jsonrpc": "2.0", "id": 2, "method": "tools/list"},
    {"jsonrpc": "2.0", "id": 3, "method": "tools/call",
     "params": {"name": "fixture.echo", "arguments": {"message": "hello"}}},
    {"jsonrpc": "2.0", "id": 4, "method": "tools/call",
     "params": {"name": "fixture.echo", "arguments": {"message": 42}}},
]
proc = subprocess.run(
    [binary, "mcp", "serve"],
    input="".join(json.dumps(m) + "\n" for m in messages),
    stdout=subprocess.PIPE, stderr=subprocess.PIPE, text=True,
    env=env, timeout=30, check=False,
)
if proc.returncode:
    sys.exit(f"MCP process exited {proc.returncode}: {proc.stderr}")
replies = [json.loads(line) for line in proc.stdout.splitlines()]
assert len(replies) == 4, f"expected 4 replies; got {replies}"
assert replies[0]["result"]["serverInfo"]["name"] == "support-shell"
listed = replies[1]["result"]["tools"]
tool = next((t for t in listed if t["name"] == "fixture.echo"), None)
assert tool is not None, f"fixture.echo missing: {listed}"
assert tool["inputSchema"]["required"] == ["message"], tool
result = replies[2]["result"]
assert result["structuredContent"]["echo"] == "hello", result
assert replies[3]["result"]["isError"] is True, replies[3]
print("PASS: real WASM via MCP tools/list, tools/call, input validation")
