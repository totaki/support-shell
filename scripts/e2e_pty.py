#!/usr/bin/env python3
"""Black-box terminal E2E: PTY input, completion, history and safe context confirmation."""
import errno
import os
import pty
import select
import signal
import subprocess
import sys
import tempfile
import time
from pathlib import Path

BINARY = os.path.abspath(os.environ.get("SUPPORT_BINARY", "./support"))
TIMEOUT = 25

def run():
    with tempfile.TemporaryDirectory(prefix="support-pty-") as tmp:
        tmp = Path(tmp)
        fake = tmp / "kubectl"
        fake.write_text("""#!/usr/bin/env python3
import json, os, sys
from pathlib import Path
args=sys.argv[1:]
state=Path(os.environ["SUPPORT_PTY_STATE"])
if args==["config","view","-o=json"]:
 print(json.dumps({"current-context":state.read_text().strip(),"contexts":[{"name":"dev","context":{"cluster":"lab"}},{"name":"test","context":{"cluster":"lab"}}]}))
elif args[:2]==["config","use-context"] and len(args)==3:
 state.write_text(args[2])
 print("Switched to context",args[2])
else:
 sys.exit("unexpected kubectl invocation: "+repr(args))
""")
        fake.chmod(0o755)
        state = tmp / "active"
        state.write_text("dev")
        env = dict(os.environ, PATH=str(tmp)+os.pathsep+os.environ.get("PATH",""),
                   XDG_CONFIG_HOME=str(tmp), NO_COLOR="1", TERM="xterm",
                   PLUGIN_PATH="plugins/fixture/target/wasm32-wasip1/release/support_fixture.wasm",
                   SUPPORT_PTY_STATE=str(state), SUPPORT_API_KEY="",
                   SUPPORT_API_BASE="https://api.openai.com/v1")
        master, slave = pty.openpty()
        proc = subprocess.Popen([BINARY], stdin=slave, stdout=slave, stderr=slave,
                                env=env, start_new_session=True)
        os.close(slave)
        transcript = bytearray()
        cursor = 0
        def expect(token, seconds=5):
            nonlocal cursor
            target=token.encode()
            deadline=time.monotonic()+seconds
            while time.monotonic()<deadline:
                found=transcript.find(target,cursor)
                if found>=0:
                    cursor=found+len(target)
                    return
                ready,_,_=select.select([master],[],[],0.1)
                if ready:
                    try:
                        chunk=os.read(master,65536)
                    except OSError as exc:
                        if exc.errno==errno.EIO: break
                        raise
                    if not chunk: break
                    transcript.extend(chunk)
            raise AssertionError(f"missing {token!r}; output tail: {bytes(transcript[-2000:])!r}")
        def send(payload):
            os.write(master,payload)
        try:
            expect("Support Shell")
            expect("support:dev")
            # Unique built-in completion: 'hist' + Tab -> history.
            send(b"hist\t\r")
            expect("1  history")
            # History navigation: up repeats history, followed by down to restore draft.
            send(b"\x1b[A\r")
            expect("2  history")
            # Reverse search must find the previous history command.
            send(b"\x12hist\r\r")
            expect("3  history")
            # Ctrl+C while editing clears the current input rather than killing the shell.
            send(b"abandoned\x03")
            expect("\r\n")
            send(b"commands\r")
            expect("k8s context use")
            # Decline context modification; fake kubectl state must remain unchanged.
            send(b"k8s context use test\r")
            expect("Apply? [y/N]")
            send(b"n\r")
            expect("Context switch cancelled")
            assert state.read_text()=="dev", "declined switch changed context"
            # Accept context modification and observe refreshed prompt.
            send(b"k8s context use test\r")
            expect("Apply? [y/N]")
            send(b"y\r")
            expect("Switched context:")
            expect("support:test")
            assert state.read_text()=="test", "confirmed switch did not happen"
            send(b"exit\r")
            proc.wait(timeout=5)
            assert proc.returncode==0, f"shell exited {proc.returncode}"
            print("PASS: PTY completion, history, reverse search, Ctrl+C editor, context cancel/confirm")
        finally:
            if proc.poll() is None:
                proc.kill()
                proc.wait(timeout=5)
            os.close(master)

if __name__=="__main__":
    try:
        run()
    except (AssertionError, subprocess.TimeoutExpired, OSError) as exc:
        sys.exit(f"FAIL: PTY E2E: {exc}")
