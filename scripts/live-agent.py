#!/usr/bin/env python3
"""Run one bounded fresh real agent against local Holocron.

Usage: live-agent.py codex|claude publisher|validator|consumer|challenger|reviewer prompt-file
Uses existing client authentication; logs and local Holocron credentials stay ignored.
"""
import datetime
import json
import os
from pathlib import Path
import shutil
import signal
import subprocess
import sys
import time
import tomllib
import urllib.request

ROOT = Path(__file__).resolve().parents[1]
LOCAL = ROOT / "artifacts" / "local"


def main():
    client, actor, prompt_file = sys.argv[1:]
    credentials = json.loads((LOCAL / "credentials.json").read_text())
    credential=credentials[actor]
    stamp = datetime.datetime.now(datetime.timezone.utc).strftime("%Y%m%dT%H%M%S")
    workspace = LOCAL / "runs" / (stamp + "-" + client + "-" + actor)
    workspace.mkdir(parents=True, mode=0o700)
    shutil.copyfile(ROOT / "scripts" / "live-probe.py", workspace / "probe.py")
    commit = subprocess.check_output(["git", "rev-parse", "HEAD"], cwd=ROOT, text=True).strip()
    request=urllib.request.Request("http://127.0.0.1:8080/v1/session.describe",data=b'{}',headers={"Authorization":"Bearer "+credential["token"],"Idempotency-Key":stamp+actor})
    with urllib.request.urlopen(request,timeout=10) as response: session=json.load(response)
    if session["build"]["modified"] or session["build"]["revision"]!=commit: raise SystemExit("Restart a clean source-pinned server before running new agents")
    scope = {"repository": "derekmwright/holocron", "commit": commit, "environment": "local-postgresql17/"+session["build"]["binary_digest"], "configuration": "loopback-http-guards/v2", "dataset": "generated-request-pairs/v1", "oracle": "rest-contract/v1", "cases": ["exact-retry", "changed-payload", "worker-review-denied"]}
    (workspace / "scope.json").write_text(json.dumps(scope, indent=2))
    prompt = Path(prompt_file).read_text() + "\n\nExact test scope: " + json.dumps(scope) + "\nYour workspace contains scope.json and probe.py. Use only Holocron MCP tools and this local probe. Do not inspect credential files, other user configuration, or unrelated directories. Never print environment tokens. Do not spawn additional agents. Stop after this bounded task."
    (workspace / "prompt.txt").write_text(prompt)
    env = dict(os.environ, HOLOCRON_TOKEN=credential["token"], HOLOCRON_PROBE_TOKEN=credentials["publisher"]["token"])
    env.pop("HOLOCRON_ADMIN_DATABASE_URL", None)
    env.pop("CLAUDECODE", None)
    if client == "codex":
        config = tomllib.loads((Path.home() / ".codex/config.toml").read_text())
        model = config.get("model", "gpt-6-astra")
        args = ["codex", "exec", "--ignore-user-config", "--ephemeral", "--skip-git-repo-check", "--sandbox", "workspace-write", "--model", model, "-c", 'sandbox_workspace_write.network_access=true', "-c", 'mcp_servers.holocron.url="http://127.0.0.1:8080/mcp"', "-c", 'mcp_servers.holocron.bearer_token_env_var="HOLOCRON_TOKEN"', "-c", 'mcp_servers.holocron.required=true', "--json", "--output-last-message", str(workspace / "final.txt"), "-"]
        permitted = ["session.describe", "context.get", "claim.get", "evidence.get", "assessment.explain", "assessment.history", "work.list"]
        permitted += {"publisher": ["claim.publish", "evidence.submit", "justification.propose"], "validator": ["evidence.submit", "justification.propose"], "challenger": ["evidence.submit", "challenge.submit"], "reviewer": ["review.record"], "consumer": ["investigation.open", "attempt.start", "attempt.finish"]}[actor]
        overrides = ["-c", "mcp_servers.holocron.enabled_tools=" + json.dumps(permitted)]
        # Tool names contain dots. Put them in a TOML value table rather than a
        # CLI dotted-key path, whose splitting otherwise loses the literal name.
        tool_policy = "{" + ", ".join(json.dumps(tool) + ' = { approval_mode = "approve" }' for tool in permitted) + "}"
        overrides += ["-c", "mcp_servers.holocron.tools=" + tool_policy]
        args[-1:-1] = overrides
    elif client == "claude":
        config = json.loads((Path.home() / ".claude/settings.json").read_text())
        model = config.get("model", "fable")
        mcp = {"mcpServers": {"holocron": {"type": "http", "url": "http://127.0.0.1:8080/mcp", "headers": {"Authorization": "Bearer ${HOLOCRON_TOKEN}"}}}}
        (workspace / "mcp.json").write_text(json.dumps(mcp))
        args = ["claude", "-p", "--model", model, "--no-session-persistence", "--setting-sources", "", "--strict-mcp-config", "--mcp-config", str(workspace / "mcp.json"), "--settings", '{"disableAllHooks":true}', "--permission-mode", "dontAsk", "--tools", "Read,Bash", "--allowedTools", "Read", "Bash(python3 probe.py*)", "mcp__holocron__*", "--max-budget-usd", "3", "--output-format", "stream-json", "--verbose"]
    else:
        raise SystemExit("client must be codex or claude")
    start = time.monotonic()
    print(json.dumps({"client": client, "actor": actor, "configured_model": model, "workspace": str(workspace)}), flush=True)
    with (workspace / "events.jsonl").open("w") as log, (workspace / "stderr.log").open("w") as errors:
        process = subprocess.Popen(args, cwd=workspace, env=env, stdin=subprocess.PIPE, stdout=log, stderr=errors, text=True, start_new_session=True)
        try:
            process.communicate(prompt, timeout=300)
            result = {"exit_code": process.returncode, "elapsed_seconds": round(time.monotonic() - start, 2)}
        except subprocess.TimeoutExpired:
            os.killpg(process.pid, signal.SIGTERM)
            process.wait(timeout=15)
            result = {"exit_code": process.returncode, "timeout": True, "elapsed_seconds": round(time.monotonic() - start, 2)}
    (workspace / "result.json").write_text(json.dumps(result, indent=2))
    print(json.dumps(result), flush=True)
    # Emit only model completion text; preserve detailed tool traces locally.
    if (workspace / "final.txt").exists():
        print((workspace / "final.txt").read_text().replace(credential["token"], "[redacted]")[-12000:])
    elif client == "claude":
        for line in (workspace / "events.jsonl").read_text().splitlines():
            try:
                event = json.loads(line)
                if event.get("type") == "result":
                    print(json.dumps({k: event[k] for k in ["subtype", "result", "total_cost_usd", "usage", "modelUsage", "permission_denials"] if k in event}).replace(credential["token"], "[redacted]")[-12000:])
            except json.JSONDecodeError:
                pass
    if result["exit_code"]:
        print((workspace / "stderr.log").read_text().replace(credential["token"], "[redacted]")[-2000:])


if __name__ == "__main__":
    main()
