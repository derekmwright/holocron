# Wadjet knowledge bootstrap and agent onboarding

This procedure uses the implemented single-visibility-domain hierarchy. It seeds metadata, then builds a small evidence-backed corpus through real investigations. It does not ingest an entire repository or assume that historical ADRs are verified facts.

## Prepare one shared compartment

Choose one tenant and visibility domain for this pilot. All projects represented inside it are readable by its authenticated participants. A project folder or conceptual label is not an access boundary; keep restricted material in another domain, which this release will not search.

Provision separate worker, validator and reviewer identities. Provision a dedicated metadata curator with `Role: worker` and `Curator: true`; it does not gain evidence-review authority. Use high-entropy expiring credentials. The [local helper](operations.md#local-service) now provisions a curator profile for its development compartment on restart; production provisioning remains an administrative operation.

Connect the intended Codex/Claude client through MCP, with its own token supplied via the environment. Copy [the participation instructions](../examples/agent-instructions.md) into the appropriate repository instructions. Call `session.describe` to check identity and capabilities. Its build revision describes **Holocron**, not the Wadjet binary under investigation.

## Review and apply the metadata plan

```sh
python3 scripts/bootstrap-wadjet.py --output artifacts/wadjet-plan.json
```

Review that plan. It contains Engineering → Wadjet → repository/component containment, including query planner, execution engine, storage, distributed coordination and hash aggregation. Concepts cover general engineering, backend, database systems, distributed execution, architecture, query planning, execution, memory, spill, SQL correctness and fault tolerance. A repository/component can have multiple classifications without being duplicated.

With a curator credential in `HOLOCRON_TOKEN`:

```sh
python3 scripts/bootstrap-wadjet.py --apply \
  --namespace wadjet-hierarchy-v1 \
  --output artifacts/wadjet-context.json
```

The output contains stable node IDs, the working-context object and an orientation request. It creates **no claims, evidence, policy obligations or findings**. Save the output for agent handoff. Retry using the same namespace and principal; do not create a new namespace just to recover an uncertain response. The bootstrap is a sequence of idempotent transactions, not one global transaction. If an authority change fences an old response, reconcile existing metadata before resubmitting writes.

## Acquire a small reviewed corpus

Start with the selected Wadjet ADRs on correctness-gate boundaries, spill engagement and performance methodology. Locate them in the actual checkout, inspect their referenced tests/issues/commits, and choose at most 20–24 useful candidate assertions. The count is a scope bound, not a contribution target.

For each source, record the exact repository commit, artifact URI, section/line fragment and SHA-256 of the referenced bytes. A local command such as `git show COMMIT:path/to/adr.md` retrieves the versioned source; hash that content rather than a mutable working file. Preserve original author/time in the explanation when available. Do not copy full sessions or sensitive source code into the knowledge base merely because it is available.

Publish a source-derived hypothesis with explicit unknown applicability:

```json
{
  "statement": "The cited ADR reports that a spill correctness gate must establish that spilling engaged",
  "kind": "hypothesis",
  "applicability": {
    "version": 2,
    "state": "unknown",
    "description": "Reported historical guidance; current component applicability has not been reproduced"
  },
  "assumptions": ["The historical mechanism may still be relevant"],
  "limitations": "Source report only; no current execution has been verified",
  "sources": [
    {
      "uri": "IMMUTABLE_ARTIFACT_URI",
      "revision": "ACTUAL_SOURCE_REVISION",
      "fragment": "EXACT_SECTION_OR_LINES",
      "digest": "sha256:ACTUAL_CONTENT_DIGEST"
    }
  ]
}
```

Replace every placeholder with observed provenance. This is an example request shape, not an importable finding. `sources` retains provenance but cannot satisfy evidence policy. Imported policy language stays reported guidance; this release does not authorize it as organization policy.

Use `association.propose` to attach the actual claim revision to appropriate concept IDs and a maintained organizational location. A curator accepts appropriate associations. Use `claim_concept` for a subject, `entity_concept` for a resource's discipline, and `placement` with an explicit steward for its home. Reuse one claim across concepts instead of publishing copies. A revised claim requires explicit new classification/placement acceptance.

## Run the first bounded investigation

Use the task: validate a regression gate for hash aggregation under forced spill, separating non-engagement, legal nondeterminism and actual wrong answers. Choose actual existing test targets after inspecting the checkout; do not treat illustrative case names as tests that ran.

1. Retrieve orientation using `artifacts/wadjet-context.json`. Unknown applicability is useful context, not local proof.
2. Establish the real Wadjet repository commit, runtime/configuration, dataset, oracle version and finite case identifiers. Resolve the working IDs with `context.resolve`, then request `context.get` with that exact scope.
3. Open a bounded investigation and start an attempt using the actual model/harness versions. Keep acceptance criteria and oracle independent of the worker's preferred answer.
4. Run the selected test/experiment. Record commands, actual/expected result digests, coverage, trigger engagement, environment and artifact identity. If the implementation fails or no discovery results, report that honestly.
5. Publish a precise `finite_cases` assertion and submit the execution manifest. Immediately propose a justification linking that evidence; it remains insufficient until the policy obligations are met.
6. Have another authenticated participant reproduce the conditions and a distinct reviewer inspect provenance/coverage. Shared code, oracle, operator and environment remain disclosed; extra agent IDs do not manufacture independence.
7. Classify the finding under both memory management and spill behavior. A fresh agent should find the same revision through either concept and retain its scope and caveats.
8. If broader transfer is worth investigating, create a separate generalization. Start with unknown applicability. To establish a finite multi-context claim, enumerate complete contexts and supply reproduction for each plus actual premise lineage and an explicit bounded-generalization warrant. Do not turn two successes into a universal memory principle.

Before finishing, refresh context. Put locally matched findings in `applied`; put relevant but unvalidated analogies in `considered`. A fresh receipt is required after metadata or authority changes. Empty findings are accepted. Record actual client usage externally when it is unavailable at the moment an agent submits its outcome.

## Observe before expanding

Keep a record of missed relevant findings, irrelevant ancestor material, incorrect classifications, repeated investigations, misleading transfer, context truncation and receipt-refresh burden. Compare `discovery: flat` and `discovery: hierarchical` with the same authorized source corpus, scope, model and budget. Charge source review and taxonomy maintenance as well as execution.

Do not infer a productivity gain from a single successful session. The immediate milestone is a usable Wadjet loop with trustworthy evidence and useful retrieval. Add scheduling, multi-domain sharing or more inference machinery only after observed behavior identifies a concrete need.
