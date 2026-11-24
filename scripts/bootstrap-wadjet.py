#!/usr/bin/env python3
"""Seed reviewed organization/concept metadata, never verified findings.

Default is a read-only plan. --apply requires a curator token in HOLOCRON_TOKEN.
Use one stable namespace and principal for retry-safe bootstrap, and save the
returned working IDs for agents. No repository content is fetched by this script.
"""
import argparse
import json
import os
from pathlib import Path
import urllib.error
import urllib.request
import uuid


def plan():
    nodes = {
        "organization": {"category": "organization", "kind": "organization", "name": "Engineering"},
        "project": {"category": "organization", "kind": "project", "name": "Wadjet"},
        "repository": {"category": "organization", "kind": "repository", "name": "wadjet", "repository": "derekmwright/wadjet"},
    }
    for key, name in {"query-planner": "Query planner", "execution-engine": "Execution engine", "storage": "Storage", "coordinator": "Distributed coordination", "hash-aggregation": "Hash aggregation"}.items():
        nodes[key] = {"category": "organization", "kind": "component", "name": name}
    subjects = {
        "engineering": "General engineering knowledge", "backend": "Backend engineering",
        "databases": "Database systems", "distributed": "Distributed execution",
        "architecture": "Wadjet project architecture", "planning": "Query planning",
        "execution": "Execution engine", "memory": "Memory management",
        "spill": "Spill behavior", "correctness": "SQL correctness", "fault-tolerance": "Fault tolerance",
    }
    for key, name in subjects.items():
        nodes[key] = {"category": "concept", "kind": "technical-subject", "name": name, "description": "Curated discovery subject; association conveys no applicability or evidential support."}
    edges = [("contains", "organization", "project"), ("contains", "project", "repository")]
    edges += [("contains", "repository", child) for child in ["query-planner", "execution-engine", "storage", "coordinator"]]
    edges += [("contains", "execution-engine", "hash-aggregation")]
    edges += [("broader", "engineering", child) for child in ["backend", "databases", "memory"]]
    edges += [("broader", "databases", "correctness"), ("broader", "databases", "planning"), ("broader", "memory", "spill"), ("related", "memory", "distributed"), ("related", "distributed", "fault-tolerance")]
    classifications = [("repository", "backend"), ("repository", "databases"), ("repository", "distributed"), ("project", "architecture"), ("query-planner", "planning"), ("execution-engine", "execution"), ("hash-aggregation", "memory"), ("hash-aggregation", "spill"), ("coordinator", "distributed"), ("coordinator", "fault-tolerance")]
    return {"nodes": nodes, "edges": edges, "classifications": classifications, "claims": [], "note": "Metadata only; no findings, policy authority, cross-domain access, or empirical applicability are created."}


def apply(args):
    token = os.environ.get("HOLOCRON_TOKEN")
    if not token:
        raise SystemExit("HOLOCRON_TOKEN is required for --apply")

    def call(operation, arguments, key):
        request = urllib.request.Request(args.url.rstrip("/") + "/v1/" + operation, data=json.dumps(arguments).encode(), headers={"Authorization": "Bearer " + token, "Idempotency-Key": key, "Content-Type": "application/json"})
        try:
            with urllib.request.urlopen(request, timeout=20) as response:
                return json.load(response)
        except urllib.error.HTTPError as error:
            raise SystemExit(f"{operation}: HTTP {error.code}; {error.read().decode()}") from None

    session = call("session.describe", {}, uuid.uuid4().hex)
    if not session.get("capabilities", {}).get("curate"):
        raise SystemExit("The selected principal has no explicit curation grant; no metadata was written.")
    spec = plan()
    ids = {}
    for key, body in spec["nodes"].items():
        ids[key] = call("node.create", body, args.namespace + ":node:" + key)["id"]
    for relation, parent, child in spec["edges"]:
        call("edge.add", {"relation": relation, "from": ids[parent], "to": ids[child], "reason": "Reviewed initial Wadjet discovery structure"}, args.namespace + ":edge:" + parent + ":" + child)
    for entity, concept in spec["classifications"]:
        key = args.namespace + ":class:" + entity + ":" + concept
        association = call("association.propose", {"kind": "entity_concept", "entity": ids[entity], "target": ids[concept], "reason": "Subject classification only"}, key)["id"]
        call("association.review", {"id": association, "verdict": "accepted", "reason": "Curated metadata; not evidence validation"}, key + ":review")
    working = {"organization": ids["organization"], "project": ids["project"], "repository": ids["repository"], "components": [ids["hash-aggregation"]], "concepts": [ids["memory"], ids["spill"]]}
    return {"namespace": args.namespace, "identity": session["identity"], "node_ids": ids, "working": working, "orientation_request": {"orientation_only": True, "working": working, "query": "hash aggregation spill correctness", "byte_budget": 16000}, "note": spec["note"]}


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--apply", action="store_true")
    parser.add_argument("--url", default="http://127.0.0.1:8080")
    parser.add_argument("--namespace", default="wadjet-hierarchy-v1")
    parser.add_argument("--output", type=Path)
    args = parser.parse_args()
    if not 1 <= len(args.namespace) <= 48:
        parser.error("namespace must have 1–48 characters")
    result = apply(args) if args.apply else plan()
    serialized = json.dumps(result, indent=2) + "\n"
    if args.output:
        args.output.parent.mkdir(parents=True, exist_ok=True)
        args.output.write_text(serialized)
    else:
        print(serialized, end="")
