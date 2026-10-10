#!/usr/bin/env python3
"""Bounded process-level test of Extism guest interruption and recovery."""
import json
import os
import subprocess
import sys
import time

binary = os.environ.get("SUPPORT_BINARY", "./support")
env = dict(os.environ, PLUGIN_PATH="plugins/fixture/target/wasm32-wasip1/release/support_fixture.wasm",
           SUPPORT_PLUGIN_TIMEOUT_MS="200")
messages = [
    {"jsonrpc": "2.0", "id": 1, "method": "tools/call",
     "params": {"name": "fixture.hang", "arguments": {}}},
    {"jsonrpc": "2.0", "id": 2, "method": "tools/call",
     "params": {"name": "fixture.echo", "arguments": {"message": "still alive"}}},
]
start = time.monotonic()
try:
    proc = subprocess.run(
        [binary, "mcp", "serve"],
        input="".join(json.dumps(m) + "\n" for m in messages),
        stdout=subprocess.PIPE, stderr=subprocess.PIPE, text=True,
        env=env, timeout=15, check=False,
    )
except subprocess.TimeoutExpired:
    sys.exit("FAIL: WASM infinite loop did not terminate within 15s (subprocess killed)")
elapsed = time.monotonic() - start
if proc.returncode:
    sys.exit(f"FAIL: host exited {proc.returncode}: {proc.stderr}")
replies = [json.loads(line) for line in proc.stdout.splitlines()]
assert len(replies) == 2, f"missing replies: {replies}"
assert replies[0]["result"].get("isError") is True, replies[0]
assert replies[1]["result"]["structuredContent"]["echo"] == "still alive", replies[1]
assert elapsed < 10, f"timeout took {elapsed:.2f}s"
print(f"PASS: infinite WASM guest interrupted, host recovered in {elapsed:.2f}s")
