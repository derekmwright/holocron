#!/usr/bin/env python3
"""Real loopback API checks; emits an evidence manifest without publishing it.

The agent must inspect and submit the manifest itself. All credentials come from
the process environment and are excluded from artifacts and output.
"""
import datetime
import hashlib
import json
import os
from pathlib import Path
import urllib.error
import urllib.request
import uuid


def digest(value):
    return "sha256:" + hashlib.sha256(value).hexdigest()


def call(operation, body, key=None, observer=False):
    token=os.environ["HOLOCRON_TOKEN"] if observer else os.environ.get("HOLOCRON_PROBE_TOKEN",os.environ["HOLOCRON_TOKEN"])
    request = urllib.request.Request("http://127.0.0.1:8080/v1/" + operation, data=json.dumps(body).encode(), headers={"Authorization": "Bearer " + token, "Idempotency-Key": key or uuid.uuid4().hex, "Content-Type": "application/json"})
    try:
        response = urllib.request.urlopen(request, timeout=15)
    except urllib.error.HTTPError as error:
        response = error
    with response:
        return response.status, response.read()


if __name__ == "__main__":
    scope = json.loads(Path("scope.json").read_text())
    subject_status,subject_raw=call("session.describe",{})
    observer_status,observer_raw=call("session.describe",{},observer=True)
    subject,observer=json.loads(subject_raw),json.loads(observer_raw)
    if subject_status!=200 or observer_status!=200 or subject["identity"]["role"]!="worker":
        raise SystemExit("Cannot establish a worker test subject and authenticated observer")
    if subject["build"]["revision"]!=scope["commit"] or subject["build"]["modified"] or subject["build"]["binary_digest"] not in scope["environment"]:
        raise SystemExit("Running server does not match the pinned source/binary scope")
    fixture_scope=dict(scope,cases=["internal-request-fixture"])
    claim = {"statement": "Disposable HTTP guard probe " + uuid.uuid4().hex, "kind": "finite_cases", "scope": fixture_scope, "limitations": "Probe fixture only; not an established finding"}
    key = uuid.uuid4().hex
    first_status, first = call("claim.publish", claim, key)
    second_status, second = call("claim.publish", claim, key)
    altered = dict(claim, statement=claim["statement"] + " changed")
    conflict_status, conflict = call("claim.publish", altered, key)
    forbidden_status, forbidden = call("review.record", {"target_kind": "justification", "target": "0" * 32, "verdict": "accepted", "reason": "Permission check; deliberately invalid worker action", "coverage_checked": True, "provenance_checked": True})
    observed = {"exact-retry": first_status == second_status == 200 and first == second, "changed-payload": conflict_status == 409, "worker-review-denied": forbidden_status == 403}
    expected = {name: True for name in scope["cases"]}
    root = uuid.uuid4().hex
    output = Path("observation-" + root + ".json").resolve()
    record = {"root": root, "scope": scope,"subject":subject,"observer":observer,"denial":json.loads(forbidden), "observed_at": datetime.datetime.now(datetime.timezone.utc).isoformat(), "statuses": [first_status, second_status, conflict_status, forbidden_status], "observed": observed, "expected": expected, "response_digests": [digest(first), digest(second), digest(conflict), digest(forbidden)]}
    payload = json.dumps(record, sort_keys=True).encode()
    output.write_bytes(payload)
    manifest = {"scope": scope, "root": root, "artifact": output.as_uri(), "digest": digest(payload), "result_digest": digest(json.dumps(observed, sort_keys=True).encode()), "expected_digest": digest(json.dumps(expected, sort_keys=True).encode()), "method": "holocron-http-guards/v2", "command": "python3 probe.py", "oracle_source": "HTTP contract: identical retry, conflicting changed payload, forbidden worker review", "outcome": "pass" if observed == expected else "fail", "observed_at": record["observed_at"], "cases": scope["cases"], "trigger_count": len(observed), "provenance": "known", "limitations": "Three finite HTTP checks; subject identity and build self-description recorded; same probe and oracle across sessions; not exhaustive security verification", "shared": ["method", "oracle", "environment", "operator", "test-subject"]}
    Path("manifest.json").write_text(json.dumps(manifest, indent=2) + "\n")
    print(json.dumps({"observed": observed, "manifest": manifest}, indent=2))
