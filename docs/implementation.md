# Phase C implementation status

The approved design has an executable Go/PostgreSQL prototype with a [single-domain hierarchy extension](hierarchy-release.md). Verification covers exact finite cases and explicitly enumerated context matrices; it does not establish unrestricted principles. Credentials still select one visibility domain, work is manually assigned, and no server-side inference is required.

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

The hierarchy release includes 36 Go integration/adversarial test functions plus their subcases, two experiment-analysis tests and one private-credential persistence regression. It retains the original live-pilot tests and adds migration, context-matrix, curation, metadata-history and Wadjet-bootstrap gates. Run `bash scripts/test-integration.sh` for the real PostgreSQL/race-enabled suite. This is test evidence for platform behavior, not agent productivity improvement.

The real PostgreSQL suite uses a non-owner runtime role and Go's race detector. It tests concurrent cycle insertion, retry conflicts, tenant isolation, pooled-connection identity clearing, malformed/unknown input, scope mismatch, no-trigger results, unknown provenance, infrastructure failure, copied/derived reports, stale completion, empty findings, interrupted work outcomes, history, revocation, and restore/purge behavior. REST/MCP tests exercise protocol initialization/discovery and shared state/authorization. The experiment-analysis tests ensure failures consume cost and unscored completion is not treated as correctness.

These checks do not establish that a worker's reported execution happened, that its oracle is correct, that all prompt injection is contained by a model, or that Holocron improves engineering performance. Reviewer trust and externally executed validation remain part of the system boundary. A [first live pilot](live-pilot.md) has now exercised twelve fresh Codex/Claude Code sessions, including a methodology challenge, inconclusive review, revised evidence, and subsequent reuse. Broader adversarial model trials and the randomized productivity experiment remain outstanding.

## Remaining work before broader use

1. Extend the live adapter pilot to interruption and outage trials; pin model as well as harness versions for controlled comparisons.
2. Curate and independently review the bounded Wadjet source corpus. This increment does not import historical assertions as findings.
3. Run the pilot with external correctness scoring and measured usage. Preregister confirmatory thresholds/sample size before making a causal claim.
4. Introduce granular redaction and external artifact/backup deletion if the deployment's privacy requirements need them. Current privacy deletion removes the whole domain.
5. Add organization authentication, per-principal quotas, deployment hardening, and measured availability controls before multi-team production use.

No lease scheduler, semantic similarity deduplication, automatic generalization, automatic contradiction resolution, organizational policy-inheritance engine, cross-domain declassification, or numerical confidence score is present. Classification and placement are curated metadata, not truth or authority. See [Wadjet onboarding](wadjet-bootstrap.md) for the next milestone.
