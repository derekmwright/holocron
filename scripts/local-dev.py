#!/usr/bin/env python3
"""Persistent loopback development instance. State and credentials stay in ignored artifacts/local.

Usage: python3 scripts/local-dev.py start|status|stop
"""
import datetime
import json
import os
from pathlib import Path
import secrets
import signal
import subprocess
import sys
import time
import urllib.request

ROOT = Path(__file__).resolve().parents[1]
STATE = ROOT / "artifacts" / "local"
PG = Path(os.environ.get("PG_BIN", "/usr/lib/postgresql/17/bin"))
BIN = STATE / "holocron"
ADMIN = f"postgresql:///postgres?host={STATE / 'socket'}&sslmode=disable"
APP = f"postgresql://holocron_app@/postgres?host={STATE / 'socket'}&sslmode=disable"


def run(args, **kwargs):
    return subprocess.run(args, check=True, **kwargs)


def status():
    try:
        with urllib.request.urlopen("http://127.0.0.1:8080/healthz", timeout=2) as response:
            return json.load(response)
    except Exception:
        return {"status": "stopped"}


def start():
    STATE.mkdir(parents=True, exist_ok=True, mode=0o700)
    STATE.chmod(0o700)
    (STATE / "socket").mkdir(exist_ok=True, mode=0o700)
    if status().get("status") == "alive":
        print(json.dumps({"url": "http://127.0.0.1:8080", "status": "already responding; no process changed"}))
        return
    env = dict(os.environ)
    env.setdefault("GOCACHE", "/tmp/holocron-go-cache")
    env.setdefault("GOMODCACHE", "/tmp/holocron-mod-cache")
    run(["go", "build", "-buildvcs=true", "-o", str(BIN), "./cmd/holocron"], cwd=ROOT, env=env)
    with (STATE / "setup.log").open("a") as log:
        if not (STATE / "postgres" / "PG_VERSION").exists():
            run([str(PG / "initdb"), "-D", str(STATE / "postgres"), "-A", "trust", "--no-locale", "-E", "UTF8"], stdout=log, stderr=log)
        running = subprocess.run([str(PG / "pg_ctl"), "-D", str(STATE / "postgres"), "status"], stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL).returncode == 0
        if not running:
            run([str(PG / "pg_ctl"), "-D", str(STATE / "postgres"), "-l", str(STATE / "postgres.log"), "-o", f"-k {STATE / 'socket'} -h ''", "start"], stdout=log, stderr=log)
        admin_env = dict(env, HOLOCRON_ADMIN_DATABASE_URL=ADMIN)
        run([str(BIN), "migrate"], env=admin_env, stdout=log, stderr=log)
        credentials = STATE / "credentials.json"
        if not credentials.exists():
            actors = {}
            for name, role in [("publisher", "worker"), ("consumer", "worker"), ("challenger", "worker"), ("validator", "validator"), ("reviewer", "reviewer"), ("outsider", "worker")]:
                token = secrets.token_urlsafe(32)
                record = {"Tenant": "local-pilot" if name != "outsider" else "isolated-pilot", "Domain": "live-agents", "ID": name, "Owner": "local-human-operator", "Role": role, "Token": token, "Expires": (datetime.datetime.now(datetime.timezone.utc) + datetime.timedelta(days=7)).isoformat()}
                run([str(BIN), "provision"], env=admin_env, input=json.dumps(record), text=True, stdout=log, stderr=log)
                actors[name] = {"token": token, "role": role, "expires": record["Expires"]}
            credentials.write_text(json.dumps(actors, indent=2) + "\n")
            credentials.chmod(0o600)
    server_env = dict(env, HOLOCRON_DATABASE_URL=APP, HOLOCRON_LISTEN="127.0.0.1:8080")
    server_env.pop("HOLOCRON_ADMIN_DATABASE_URL", None)
    with (STATE / "server.log").open("a") as log:
        process = subprocess.Popen([str(BIN), "serve"], cwd=ROOT, env=server_env, stdin=subprocess.DEVNULL, stdout=log, stderr=log, start_new_session=True)
    (STATE / "server.pid").write_text(str(process.pid))
    for _ in range(50):
        if process.poll() is not None:
            raise SystemExit("server exited; inspect artifacts/local/server.log")
        if status().get("status") == "alive":
            print(json.dumps({"url": "http://127.0.0.1:8080", "mcp": "http://127.0.0.1:8080/mcp", "pid": process.pid, "state": str(STATE), "credentials": "private file created; tokens not printed"}))
            return
        time.sleep(0.1)
    raise SystemExit("server did not become healthy")


def stop():
    pidfile = STATE / "server.pid"
    if pidfile.exists():
        pid = int(pidfile.read_text())
        try:
            if Path(f"/proc/{pid}/exe").resolve() == BIN:
                os.kill(pid, signal.SIGTERM)
        except ProcessLookupError:
            pass
    if (STATE / "postgres" / "PG_VERSION").exists():
        subprocess.run([str(PG / "pg_ctl"), "-D", str(STATE / "postgres"), "-m", "fast", "stop"], check=False)


if __name__ == "__main__":
    command = sys.argv[1] if len(sys.argv) > 1 else "status"
    if command == "start":
        start()
    elif command == "stop":
        stop()
    elif command == "status":
        print(json.dumps(status()))
    else:
        raise SystemExit(__doc__)
