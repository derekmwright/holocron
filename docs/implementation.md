# Phase C implementation status

The approved design now has an executable Go/PostgreSQL prototype. Its evidence semantics are intentionally narrower than the long-term platform: one finite-case reproduction policy, exact Git revisions and environment dimensions, one visibility domain per credential, manually assigned work, and no server-side inference.

## Implemented behavior

| Capability | Implementation and verification |
| --- | --- |
| Scoped claims and immutable revisions | Typed inputs, full commit IDs, revision-specific references, append-only SQL records, successor conflict checks. |
| Evidence and lineage | Attributed manifests, source/result/expected digests, observed time, declared shared dimensions, typed parent links. Copied manifests and common observation ancestry cannot satisfy reproduction. |
| Supporting and contradictory evidence | Independent execution obligation, distinct reviewer authority, pending/accepted/rejected/inconclusive challenges, supported/contested/refuted assessments. Methodology objections disable only the targeted justification. |
| Dependent knowledge | AND premises and alternative OR justifications; cycles rejected under a domain transaction lock; alternative support survives loss of another path. Pending premise warnings propagate without pretending to be refutations. |
| Historical and current understanding | Immutable assessment snapshots, read-time deadline/credential checks, retraction, supersession, and current qualified cards. |
| Context | Scope filtering, PostgreSQL full-text ranking, whole-card byte budgets, provenance and limitations, work history, delivery receipts and stale-receipt gates. |
| Coordination | Immutable bounded investigation contracts, authenticated attempts, empty/negative outcomes, reported usage and applied knowledge. Scheduling, renewable leases, and speculative work remain deferred. |
| Access | Hashed expiring credentials, manually assigned roles, service checks, tenant/domain RLS, composite foreign keys, non-owner runtime-role guard, cross-origin protection, bounded input. |
| Delivery | Shared REST/MCP application service, idempotent writes and exact-response retries, atomic outbox, at-least-once consumer poll/ack. |
| Operations | Migration checksum, provisioning/revocation, whole-domain administrative purge, deletion-ledger replay test after backup restore, graceful shutdown, optional local OpenTelemetry exporters. |
| Measurement | Immutable experiment arm/model/harness/task-family records, token/time/cost reporting, reviewer export, and independent-score analysis that includes failures and exposes missing data. |

The walking-scenario integration test follows A publishing, B retrieving/applying, C challenging, dependency revalidation, and D receiving corrected understanding. These are separate authenticated **scripted participants** with synthetic manifests, not independent model runs or reproduced Wadjet findings. Their purpose is to test platform semantics.

## Validation boundary

Development checks completed on 2026-11-24: 18 Go integration/adversarial test functions (including their subcases) pass against PostgreSQL 17 with `-race`; the two experiment-analysis tests pass; `go vet ./...` passes. The scripted walking scenario is runnable with `bash scripts/demo.sh`. This is test evidence for the prototype, not evidence of agent productivity improvement.

The real PostgreSQL suite uses a non-owner runtime role and Go's race detector. It tests concurrent cycle insertion, retry conflicts, tenant isolation, pooled-connection identity clearing, malformed/unknown input, scope mismatch, no-trigger results, unknown provenance, infrastructure failure, copied/derived reports, stale completion, empty findings, interrupted work outcomes, history, revocation, and restore/purge behavior. REST/MCP tests exercise protocol initialization/discovery and shared state/authorization. The experiment-analysis tests ensure failures consume cost and unscored completion is not treated as correctness.

These checks do not establish that a worker's reported execution happened, that its oracle is correct, that all prompt injection is contained by a model, or that Holocron improves engineering performance. Reviewer trust and externally executed validation remain part of the system boundary. Real client configuration examples are supplied, but fresh Codex/Claude Code model workflows, adversarial model trials, and the randomized productivity experiment have not been run.

## Remaining work before broader use

1. Run the adapter participation protocol in fresh model sessions, including interruption and outage trials; pin model as well as harness versions.
2. Curate and independently review the bounded Wadjet source corpus. This increment does not import historical assertions as findings.
3. Run the pilot with external correctness scoring and measured usage. Preregister confirmatory thresholds/sample size before making a causal claim.
4. Introduce granular redaction and external artifact/backup deletion if the deployment's privacy requirements need them. Current privacy deletion removes the whole domain.
5. Add organization authentication, per-principal quotas, deployment hardening, and measured availability controls before multi-team production use.

No lease scheduler, semantic similarity deduplication, automatic generalization, automatic contradiction resolution, arbitrary policy editor, cross-domain declassification, or general numerical confidence score is present. These omissions preserve the agreed experimental scope.
