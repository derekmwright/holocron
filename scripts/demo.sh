#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."
# Scripted protocol participants, not a claim of independent model improvement.
bash scripts/test-integration.sh -run TestWalkingScenario -v
