# MVP and experimental validation

Status: design and Phase C scope approved on 2026-11-24. This document specifies intended acceptance behavior. [Implementation status](implementation.md) distinguishes executed tests from pending live-agent and empirical validation.

## Deliberately constrained MVP

Implement one Go service with PostgreSQL, migrations, REST and a small MCP surface, seeded identities/roles, one Wadjet project visibility domain, and a second tenant for isolation tests. Include claim revisions, evidence/provenance manifests, explicit justifications, challenges, fixed policies, immutable assessments, dependency revalidation, scoped retrieval, manually assigned attempts, audit/outbox records, and experiment instrumentation.

Use one manually configured software-validation policy. Keep one supporting and one counterevidence workflow complete before introducing other policy domains. Require no server inference, graph database, embeddings, NATS deployment, enterprise frontend, general issue tracker, autonomous coordinator, billing, or managed worker runtime. Concept labels and questions are lightweight; automatic ontology formation and generalization are deferred. No broad Wadjet crawler in this increment.

The MVP is allowed to reject scopes it cannot compare, derivation cycles, cross-domain references, and graph mutations beyond its traversal bound. These are explicit unsupported operations, not silent approximations.

### Initial verification policy

Pin `software-case-reproduction/v1` to a finite enumerated query/fixture set, commit, configuration, environment specification, and oracle revision. Require the original result manifest, source identity/digest, command, expected-output lineage, actual output, applicable trigger counters, and an independently executed reproduction by an authorized validator distinct from the proposing principal. The validator must inspect the oracle lineage and trigger coverage; shared test, oracle, model, owner, and environment dimensions remain disclosed. Separate execution is repeatability evidence, not an assertion of full methodological independence.

Support requires both executions to agree with the specified oracle on all enumerated cases, satisfy required trigger coverage, have no invalid mandatory premise, and have a reviewed scope warrant. This policy establishes finite-case reproduction only. Broader correctness and performance claims remain insufficient under this policy. A falsification requires an independently reproduced mismatch on a named in-scope case with intact oracle provenance and engaged conditions. Conflicting admissible executions remain contested until their disagreement is explained or the policy's falsification conditions otherwise resolve it. A reviewer may reject bad methodology with recorded reasons, but cannot delete the inconvenient result.

## Walking scenario

Use a controlled fixture inspired by Wadjet's spill coverage lessons. The proposition below is a synthetic test scenario, not a newly asserted Wadjet defect.

1. Agent A publishes C1: “For query family Q, dataset D, configuration E, and commit K, the forced-spill result equals oracle O.” It supplies exact input identities, expected result provenance, output, command, and spill-engagement counters. It proposes a justification rather than self-awarding verification.
2. An authorized validator independently reruns the method and records what is and is not independent. The policy assessor issues a supported assessment limited to those conditions. C2, a proposed optimization eligibility claim, has a reviewed justification requiring C1 plus a separately checked prerequisite.
3. Agent B starts with a clean session, retrieves the scoped finding and its limits, and applies it to a bounded engineering task. Its attempt records the actual reused revision and passes a hidden check. Retrieval alone does not count as success.
4. Agent C supplies a reproducible in-scope counterexample. A pending challenge appears immediately. A validator verifies the counterexample and scope; accepted conflicting results move C1 to contested while the cause remains unresolved.
5. C2 loses its sole usable justification and becomes revalidation-required. A variant of the fixture gives C2 another adequate, unaffected justification; that variant remains supported with an explanation of the lost path.
6. Investigation finds the original coverage assertion was incorrect. Its justification is invalidated with the correction retained. C1 becomes refuted if the policy's falsification conditions are met. A narrower C1 revision is proposed and must be assessed afresh.
7. Agent D retrieves the new state. It receives the contradiction, narrower scope, and affected dependency status, not the old unqualified finding. A stale completion receipt is rejected until refreshed.
8. Compare equivalent independent work before and after this history using the experiment below. The scripted scenario establishes mechanics only.

## Integration and adversarial acceptance cases

Write these against real PostgreSQL transactions and both transports before expanding features. Use independent database connections and controlled concurrency barriers for race cases; avoid relying only on sleeps or mocks.

| Case | Required result |
| --- | --- |
| Publish scoped claim and attach evidence | Immutable records reference exact revisions, actor, activity, artifact identity, and conditions; missing required scope fails validation. |
| Three reports of one run | Shared origin is visible; no replication obligation is satisfied by report count. |
| New agent with same test/oracle | New execution is recorded, shared methodology remains explicit; independence is not inferred from ID. |
| Unknown provenance | Evidence can be retained, but cannot satisfy policy obligations requiring traceability or independence. |
| Self support, two-node cycle, concurrent cycle creation | Derivation mutation rejects atomically; no circular path confers support. |
| Valid AND premises and alternative OR justifications | All members of one justification are required; losing one alternative does not destroy another adequate path. |
| Agent files unsubstantiated challenge | Visible pending objection; no automatic refutation or erasure of support. |
| Accepted methodological challenge | The targeted justification becomes unusable; unrelated evidence survives. |
| In-scope contradictory result | Assessment retains both sides and reasons; resolution requires the configured authority. |
| Different commit or nonoverlapping environment | No automatic contradiction or reuse; applicability mismatch is explicit. |
| Unsupported generalization | Wider scope cannot inherit a finding; requires new warrant and review. |
| Test passes without triggering spill | Trigger obligation fails; result does not qualify as spill evidence. |
| Test runner crashes or times out | Infrastructure/inconclusive outcome; no false refutation or success. |
| Exhausted bounded search versus timeout | Only demonstrated coverage can support bounded absence; timeout remains unknown. |
| Supporting evidence retracted or deleted | Dependents reassessed and historical lineage preserved to permitted extent; no stale current finding. |
| Revalidation racing context retrieval | A reader sees one consistent revision; never a new source state paired with a falsely current dependent. |
| Traversal bound exceeded | Whole mutation rolls back with an explicit error; no partial propagation. |
| Claim revision or policy changed during validation | Old result remains bound to old revisions; it cannot complete the new contract. |
| Worker changes criteria or self-approves | Authorization denies the operation and records the denial. |
| Same idempotency key after lost response | Exact retry returns original IDs; altered payload conflicts. |
| Cross-tenant IDs, joins, search and graph traversal | No rows, counts, snippets, or foreign-key behavior expose protected content. |
| Connection pool switches tenants | Transaction-local settings cannot leak prior identity. |
| Restrict source or attempt cross-domain derivation | No derived-content widening; caches/receipts cannot bypass revocation. |
| Poison artifact requests secrets or tool execution | Returned only as untrusted evidence; service performs no requested execution. Separate agent red-team trial measures whether external gates contain misuse. |
| Oversized claim card under context budget | Qualification retained or card omitted; no silent overflow or authoritative fragment. |
| Empty contribution and interrupted agent | Empty valid disposition accepted; interruption is not fabricated as a finding or completed work. |
| Crash before commit, after commit, during outbox dispatch | No partial claim/evidence state; events retried and consumers tolerate duplicates. |
| Redaction and restore | Removed material stays inaccessible under the tested backup/restore policy; dependents disclose unavailable support. |

When leasing is introduced, add acquisition races, expired renewal, cancellation versus completion, stale fencing generation, clock skew, duplicate completion, late evidence, decomposition cycles, budget reservation races, and speculative winner/loser result retention before enabling multiple coordinators.

## Wadjet bootstrap

Start with a small reviewed corpus, provisionally 20–30 claim candidates across SQL correctness, spill/memory, and benchmark methodology. The count is a workload bound, not a success metric. Use the pinned ADRs in [research](research.md) as entry points, then follow their cited issues, exact commits, regression tests, and run artifacts. Add distributed execution, fault tolerance, and rejected optimizations only as needed by selected tasks.

For each source, retain repository identity, immutable revision, path or issue/comment ID, source timestamp, retrieval time, digest where permitted, author, importer, extractor version, and exact cited fragment. Session imports require an explicit visibility/retention review; do not ingest entire transcripts by default. Closed issues and accepted ADRs supply reported outcomes or organizational decisions, not automatically verified factual claims.

Separate extraction from validation. An agent proposes precise assertions, scope, limitations, and lineage; a reviewer checks fidelity to source. Imported evidence starts unassessed. Independently reproduce high-value claims where artifacts are available. A missing benchmark log remains a missing log, not permission to invent provenance.

Use source identity plus revision plus extraction identity for repeatable imports. Updates create successor records; paraphrase similarity suggests review, never deletes independent observations. Link multiple reports back to the original execution. Retain failed approaches with the exact conditions and reasons they failed; “never try this” is too broad unless justified.

For temporal experiments, restrict ingestion to artifacts available before the target task. Exclude the held-out solution, its follow-up summaries, and tests that disclose its answer. Record source licenses and permitted retention. Bootstrap cost includes extraction, review, reproduction, and cleanup.

## Controlled experiment

Primary estimand: the change in successful completion and total resource cost for subsequent independent agents attributable to accumulated organizational knowledge, with model and execution conditions fixed within each comparison.

Use three arms:

* A: baseline repository, tools, and ordinary project instructions, without Holocron.
* B: the same approved historical corpus available through straightforward lexical document retrieval.
* C: the same source corpus represented through Holocron, including qualification, contradiction, lineage, and revalidation.

A versus C estimates the practical package effect. B versus C helps distinguish structured knowledge maintenance from simply supplying more information. Keep model version, harness version, tool permissions, starting checkout, compute allocation, task deadline, and total token ceiling equal within matched comparisons. Charge all context, tool, reasoning, and output tokens where measurable; disclose any unobservable usage. Run separate strata for Codex and Claude Code instead of confounding harness/model differences with treatment.

Select task families before running: a correctness bug, a spill coverage investigation, a performance decision, a repeated failed approach, and coordinator onboarding. Use held-out variants and a temporal cutoff. Randomize fresh agent sessions within task/model blocks; never let the same session perform multiple arms. Blind correctness reviewers to the arm and use hidden regression oracles plus a rubric for non-code investigations. Separate infrastructure failures from agent failures, report both, and predefine any rerun rule.

First run a small pilot to estimate task variance and instrumentation reliability. Determine the confirmatory sample size from a chosen minimum useful effect and pilot variance; do not invent statistical power from an arbitrary task count. Freeze the confirmatory protocol and analysis before inspecting treatment outcomes. Use task-family-level uncertainty estimates so repeated attempts of one task are not treated as independent task diversity.

Measure:

| Outcome | Operational definition |
| --- | --- |
| Correctness | Hidden checks and blinded rubric satisfied within budget; report failures and censored runs. |
| Cost and time | Total tokens, tool/compute costs, elapsed time, and time to validated success, including retries. Do not compare only successful runs. |
| Redundant investigation | Blinded coding of repeated work whose applicable result was already available; distinguish necessary replication from waste. |
| Failed-approach recurrence | Reuse of a documented failed approach under matching conditions, assessed from traces. |
| Retrieval quality | Relevance and coverage of a reviewed gold set, warnings retained, and harmful omissions under a fixed budget. |
| Knowledge use | Delivered versus cited versus applied; useful application confirmed by task evidence, not agent praise. |
| Evidence quality | Reviewer checks of scope, reproducibility, lineage, trigger coverage, and justified independence. |
| Incorrect propagation | Seeded stale or false claim leads to an incorrect downstream action, measured before and after challenge/revision. |
| Onboarding | Cost to correctly identify constraints and execute a held-out coordinator task from a fresh session. |
| Contribution quality | Useful, faithful candidate findings per reviewed attempt, with empty outcomes allowed; include review burden. |
| Maintenance | Authoring, reproduction, assessment, correction, revalidation, and deletion time/cost. |

For compounding, create an acquisition cohort and later reuse cohorts. Freeze a snapshot for each randomized comparison, then publish the next reviewed snapshot between cohorts. Keep controls isolated from treatment discoveries. Track cumulative acquisition plus maintenance plus execution cost against cumulative successful tasks. A positive later-session result can still have negative overall return.

Preregister practical thresholds with the owner before confirmatory runs: acceptable correctness noninferiority margin, minimum cost/time improvement, unacceptable harmful-propagation rate, and maximum maintenance burden. These are organizational preferences, not scientific constants. If the interval includes material harm or fails to resolve the target effect, report inconclusive; do not expand infrastructure based on a favorable point estimate.

## Review and implementation sequence

1. Agree on vocabulary, immutable revisions, explicit justifications, policy assessments, and restricted scope/visibility rules.
2. Approve the bounded MVP and the distinction between pending challenges, accepted counterevidence, and refutation.
3. Implement database invariants and the walking-scenario integration/adversarial tests, then the application services and REST/MCP adapters until those tests pass.
4. Run clean-session participation tests for both target clients, including interruption, service outage, and empty findings.
5. Curate the small Wadjet corpus, preregister pilot conditions, and measure the three experiment arms.
6. Review failures and maintenance cost before adding leases, broad imports, automated consolidation, or semantic retrieval.

The design-review boundary has been satisfied. Confirmatory experiment thresholds still need preregistration before interpreting productivity results; this does not block implementation or protocol testing.
