#!/usr/bin/env python3
"""Summarize Holocron exports with independent outcome adjudications.

Usage: analyze-experiment.py export.json scores.json
scores.json maps outcome IDs to {"correct": true|false, "reviewer": "..."}.
Missing outcomes and unscored outcomes remain visible; agent claims are not scores.
"""
import json
import sys
from collections import defaultdict


def summarize(export, scores):
    outcomes = {row["body"]["attempt"]: row for row in export["outcomes"]}
    groups = defaultdict(lambda: {
        "attempts": 0, "reported": 0, "adjudicated": 0, "correct": 0,
        "tokens_reported": 0, "elapsed_ms_reported": 0, "cost_micros_reported": 0,
        "missing_outcomes": 0, "unscored_outcomes": 0,
    })
    for attempt in export["attempts"]:
        data = attempt["body"]
        key = (data["model"], data["harness"], data["task_family"], data["arm"])
        group = groups[key]
        group["attempts"] += 1
        outcome = outcomes.get(attempt["id"])
        if outcome is None:
            group["missing_outcomes"] += 1
            continue
        group["reported"] += 1
        body = outcome["body"]
        for field in ("tokens", "elapsed_ms", "cost_micros"):
            value = body[field]
            if not isinstance(value, int) or value < 0:
                raise ValueError(f"invalid {field} for outcome {outcome['id']}")
            group[field + "_reported"] += value
        score = scores.get(outcome["id"])
        if score is None:
            group["unscored_outcomes"] += 1
            continue
        if type(score.get("correct")) is not bool or not score.get("reviewer"):
            raise ValueError("scores require a boolean correctness verdict and reviewer")
        group["adjudicated"] += 1
        if score["correct"] and body["outcome"] == "completed":
            group["correct"] += 1
    rows = []
    for key, group in sorted(groups.items()):
        group.update(zip(("model", "harness", "task_family", "arm"), key))
        group["validated_success_fraction_all_attempts"] = group["correct"] / group["attempts"]
        complete = group["missing_outcomes"] == group["unscored_outcomes"] == 0
        group["complete_observation"] = complete
        group["tokens_per_validated_success"] = (
            group["tokens_reported"] / group["correct"]
            if complete and group["correct"] else None
        )
        rows.append(group)
    return {
        "groups": rows,
        "limitations": [
            "Descriptive totals only; no randomized-treatment or significance claim.",
            "Usage is agent-reported and needs independent metering for confirmatory work.",
            "Acquisition and maintenance costs must be added separately.",
            "Missing outcomes are not zero-cost successes; incomplete cost ratios are omitted.",
            "Exports cap each collection at 1000 rows; check completeness before analysis.",
        ],
    }


if __name__ == "__main__":
    if len(sys.argv) != 3:
        raise SystemExit(__doc__)
    with open(sys.argv[1], encoding="utf-8") as src:
        export = json.load(src)
    with open(sys.argv[2], encoding="utf-8") as src:
        scores = json.load(src)
    print(json.dumps(summarize(export, scores), indent=2))
