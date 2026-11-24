# Prototype API

The service exposes authenticated `POST /v1/{operation}` and MCP Streamable HTTP at `/mcp`. `GET /healthz` reports process liveness only. Both transports call the same application service. Responses are non-cacheable; requests from foreign browser origins are rejected.

The [hierarchy release](hierarchy-release.md) documents metadata operations, envelope v2, working-context resolution, curation authority and migration 2. The [Wadjet bootstrap procedure](wadjet-bootstrap.md) provides a concrete onboarding path.

REST requires `Authorization: Bearer <token>`, `Idempotency-Key: <unique-key>`, and a JSON object body. MCP uses the same bearer credential and arguments shaped as `{"key":"unique-key","arguments":{...}}`. MCP tool schemas describe every field. Do not put a token in arguments. The service resolves tenant, domain, principal, and role from the credential.

The MCP catalog now advertises categorical enums, case/array limits, digest/URI patterns, numeric bounds and conditional scope/envelope rules. For source-only claims, omit `scope` and use a version-2 unknown-applicability envelope; pinned repository/commit provenance stays in `sources`. The accepted experiment arms are `A`, `B`, `C`, `demo`; attempt outcomes are `completed`, `failed`, `partial`, `timed_out`, `cancelled`, `abandoned`. An unfinished blocked attempt can be recorded as `partial` with its blocker in the disposition/questions. Nothing is silently coerced into another outcome.

| Operation | Input and behavior |
| --- | --- |
| `session.describe` | Empty object; authenticated principal/role and running binary digest, embedded source revision, dirty-build flag, Go version, and policy. No credentials are returned. An empty revision is explicitly unpinned. |
| `claim.publish` | Statement, legacy exact scope or explicit applicability envelope v2, kind, limitations; optional assumptions, source references, review deadline and previous revision. Legacy concept strings are annotations, not curated graph associations. |
| `evidence.submit` | Execution/artifact manifest with declared roots and optional `derived_from` evidence IDs. Unknown provenance is retained but cannot qualify. |
| `evidence.get` | ID; returns full manifest, attribution, and lifecycle. |
| `justification.propose` | Claim ID, evidence IDs, premise claim IDs, warrant, limitations. Evidence within a justification is jointly required; justifications are alternatives. |
| `challenge.submit` | Claim, kind (`counterexample` or `methodology`), reason, optional evidence; methodology requires a justification belonging to the claim. |
| `review.record` | Reviewer-only target kind, target ID, accepted/rejected/inconclusive verdict, reason, coverage and provenance checks. Self-review is denied. Accepted counterexamples require reproduced evidence. |
| `record.retract` | Claim/evidence kind, target ID, reason. Author or reviewer; preserves history. This is not privacy deletion. |
| `assessment.explain` | Claim ID; current policy posture, lifecycle, dependency health, reasons, pending challenges and justification paths. |
| `assessment.history` | Claim ID; latest 100 stored assessments in descending graph epoch. Current time/credential eligibility can differ from history. |
| `claim.get` | Claim ID; full qualified card including evidence references. |
| `context.get` | Exact scope or orientation-only request, optional working IDs/query/objective, discovery mode and byte budget (512–262144); qualified cards, local applicability, work, truncation and receipt. |
| `investigation.open` | Objective, question, scope, completion criteria, per-attempt token budget, deadline, support/pending-challenge requirements. Contract is immutable. |
| `attempt.start` | Investigation ID, model, harness, arm (`A`, `B`, `C`, `demo`), task family. Records an attempt; no leases or worker control. |
| `attempt.finish` | Attempt, receipt, outcome, disposition, locally matched applied IDs, considered orientation IDs, findings, negative results, questions and measured usage. Empty findings are valid. |
| `work.list` | Empty object; up to 1000 contracts, attempts, and outcomes in the current domain. |
| `experiment.export` | Reviewer-only work export. Costs and completion are agent-reported; external adjudication is necessary. |
| `events.poll` | Reviewer-only consumer name; up to 100 unacknowledged metadata-only events. Repeat polls redeliver until acknowledged. |
| `events.ack` | Consumer and event ID; idempotent acknowledgement after durable consumer handling. |

Unknown fields are rejected. Arguments are limited to 256 KiB. With unchanged authorization, retries return original bytes even if the snapshot is historical. Authority changes fence old response replay without repeating the mutation. Use new keys for fresh reads, but reconcile previously committed mutations before submitting new mutation keys. Changed arguments under an existing key conflict.

REST errors: 400 invalid arguments, 401 missing/expired credential, 403 insufficient authority, 404 inaccessible or missing record, 409 revision/receipt/idempotency conflict, 422 graph bound, 500 internal failure with details suppressed. MCP returns tool errors for the same application failures. Direct access and cross-tenant requests do not reveal which protected object exists.

## Example claim

```json
{
  "statement": "The two named fixture cases match the pinned oracle under forced spill",
  "kind": "finite_cases",
  "scope": {
    "repository": "wadjet-fixture",
    "commit": "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
    "environment": "linux-fixture-v1",
    "configuration": "forced-spill-v1",
    "dataset": "fixture-v1",
    "oracle": "oracle-v1",
    "cases": ["group-null", "group-decimal"]
  },
  "limitations": "Only these fixtures; no universal SQL correctness claim"
}
```

## Evidence manifest obligations

An execution manifest includes `scope`, `root`, the actual `artifact` URI, artifact `digest`, method/command, reported outcome, observation time, cases, trigger count, provenance, limitations and shared dimensions. `file:///absolute/path/to/retained-capture.json` is valid. Do not substitute a GitHub source URL for different probe output. A URI locates the artifact; its digest binds the referenced bytes. Digests use `sha256:` plus 64 hexadecimal characters.

Known provenance requires a recorded origin and valid artifact digest, not a golden output. Comparison fields `result_digest`, `expected_digest`, and `oracle_source` may be omitted or empty at capture time. They remain mandatory for comparison-based establishment; a reviewer cannot waive them. When both comparison digests are supplied for known provenance, the reported pass/fail must agree with their equality. An actual expected property or metamorphic relation can provide an oracle; never copy the result digest simply to manufacture a pass. Normalize actual and expected representations under the declared method before hashing.

`observed_at` must be the actual RFC3339 observation timestamp, not a deadline. The server allows at most five minutes of future clock skew and identifies timestamp errors separately from a negative `trigger_count`. Assessments include `evidence_profile` describing referenced records, declared provenance, reported outcomes and comparison-field presence. These are recorded conditions, not confidence scores or proof that the server executed/fetched the artifact. Unknown provenance is no longer mislabeled as an unavailable artifact.

Supporting pairs require different authenticated producers, distinct manifests, no shared declared observation ancestry, matching method/oracle, and at least one authorized validator. Both must cover every claimed case and report positive engagement. Counterexamples require a common in-scope failing case. Different identities and roots do not prove independence; the review still checks methodology and disclosures. Credentials identify who submitted the report, not whether the reported execution happened.

Artifacts are never fetched or executed by the server. Put sensitive artifacts in customer-controlled stores. Unknown provenance, infrastructure failure, timeout, or bounded search exhaustion is retained without becoming correctness support.

## Read and write consistency

All service operations take the visibility-domain row lock. Relevant mutations increment its epoch, validate the acyclic graph, and write assessment history plus an outbox event in the same transaction. The default bounds are 1000 claims and 10000 core graph records per domain. Over-bound mutations roll back, including their epoch update. This intentionally simple prototype favors consistent explanations over throughput.

Context and completion recalculate eligibility against current review deadlines and participant revocations. A completion requires a current receipt owned by the attempt author, scope coverage, unexceeded contract budget/deadline, and the requested applied-knowledge gates. Failed, partial, timed-out, cancelled, and abandoned outcomes remain reportable after expiry. None of these reports automatically marks an investigation independently verified.
