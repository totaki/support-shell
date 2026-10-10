#!/usr/bin/env python3
"""Black-box non-interactive CLI smoke test against the compiled Support Shell."""
import os
import subprocess
import sys
import tempfile

binary = os.environ.get("SUPPORT_BINARY", "./support")
env = dict(os.environ, PLUGIN_PATH="plugins/fixture/target/wasm32-wasip1/release/support_fixture.wasm",
           NO_COLOR="1", SUPPORT_API_KEY="", SUPPORT_API_BASE="https://api.openai.com/v1")
commands = "help\ncommands\nhelp fixture echo\nset format json\nfixture echo hello\nhistory\nexit\n"
with tempfile.TemporaryDirectory(prefix="support-shell-e2e-") as home:
    env["XDG_CONFIG_HOME"] = home
    proc = subprocess.run(
        [binary], input=commands, capture_output=True, text=True,
        env=env, timeout=25, check=False,
    )

if proc.returncode:
    sys.exit(f"FAIL: shell exited {proc.returncode}: {proc.stderr}")
out = proc.stdout
required = (
    "Support Shell", "fixture echo", "Output format: json",
    '"echo": "hello"', "history", "exit",
)
missing = [part for part in required if part not in out]
if missing:
    sys.exit(f"FAIL: CLI output missing {missing}: stdout={out!r}, stderr={proc.stderr!r}")
print("PASS: CLI help, command listing, JSON output, history and exit")
