# Architectural decision records

Records 001–009 were **accepted** by founder approval of the foundational design and Phase C scope on 2026-11-24. Supporting investigation is in [research](research.md); operational details are in [design](design.md). Record 010 documents concrete implementation choices within that scope.

## ADR 001 Explicit justifications and policy assessments

Context: artifacts differ in coverage, methodology, and shared origins. Agent agreement supplies no calibrated likelihood model.

Decision: use revision-specific AND/OR justifications, explicit challenges, and explainable policy assessments. Keep priority, relevance, utility, freshness, and applicability separate.

Alternatives: a numerical confidence score is easy to sort but unjustified across domains; a full Bayesian network requires likelihoods and dependency assumptions we do not have; a full argumentation or ATMS engine adds complexity before useful scope authoring is established.

Consequences: conservative states can be incomparable. Human review remains necessary for semantic warrants. The MVP can explain satisfied obligations but cannot claim a universal probability of truth.

Revisit when a narrow domain supplies calibrated predictions and enough held-out outcomes to validate probabilities.

## ADR 002 Immutable revisions with mutable operational projections

Context: correcting a claim must not retroactively alter what an earlier worker relied on.

Decision: immutable claim/evidence versions and append-only lifecycle records; transactionally maintained current projections. Preserve historical assessments and distinguish observed time from recorded time. Authorized deletion can remove protected payloads and derived material.

Alternatives: mutable claims lose reproducibility; full event sourcing increases reconstruction and migration complexity; permanent retention conflicts with privacy and security requirements.

Consequences: every reference must pin a revision. Storage grows and retention needs lineage-aware handling. Redaction may make old explanations incomplete, which must be explicit.

Revisit projection design if replay, audit, or throughput needs exceed ordinary relational transactions.

## ADR 003 PostgreSQL modular monolith

Context: the three graphs are logical distinctions with shared consistency and authorization requirements.

Decision: Go, pgx, explicit SQL, typed relational adjacency, PostgreSQL full-text retrieval, and bounded recursive traversal. Keep module boundaries without network services. No vector dependency, graph database, server inference, or NATS in MVP.

Alternatives: a graph database improves some traversals but introduces a second authority or replaces familiar transactional controls; microservices make atomic assessment/access changes harder; vector retrieval can help semantic recall but does not establish evidence quality.

Consequences: graph mutation throughput is deliberately limited. Search begins with a measurable lexical baseline. The database needs careful indexes and transaction tests.

Revisit only after representative workload measurements identify an actual bottleneck or retrieval gap.

## ADR 004 Conservative dependency revalidation

Context: a lost premise invalidates a justification, not necessarily its conclusion; asynchronous propagation can serve stale findings.

Decision: explicit typed dependencies, alternative justifications, acyclic derivation, and synchronous bounded reassessment in MVP. Reject over-bound mutations atomically. Context includes assessment and graph revisions.

Alternatives: mark all descendants false, which is unsound; leave propagation entirely asynchronous, which requires read barriers; implement general cyclic reasoning, which adds semantics beyond the pilot.

Consequences: conservative review load and reduced write concurrency. Independent adequate support can preserve a finding. Broad changes may require smaller explicit batches while readers retain consistent states.

Revisit with a dirty-epoch read barrier if measured graph sizes require asynchronous processing.

## ADR 005 Evidence trust and organizational authority are separate

Context: a human-approved decision can rest on weak evidence; an agent can submit a valid test without permission to change policy.

Decision: immutable policy versions, separate proposer/validator/assessor authority, and fixed work contracts. All assessments identify policy and inputs. Approval never silently changes factual posture.

Alternatives: one canonical flag conflates distinct judgments; worker-controlled acceptance criteria permit goalpost movement.

Consequences: policy administration is explicit work. Initial trusted reviewers are manually configured. Shared ownership is disclosed for independence checks.

Revisit delegation ergonomics after measuring validation bottlenecks; retain authority separation.

## ADR 006 Restricted derivation by default

Context: an accessible summary can leak an inaccessible source even when source links are hidden.

Decision: MVP derivations remain within one visibility domain. Future derived readership cannot exceed the intersection of all used sources without an authorized release review. Enforce application checks and RLS with tenant-qualified integrity.

Alternatives: inherit the destination ACL, which launders secrets; use only source-link ACLs, which leaves derived text exposed; per-statement arbitrary sharing initially, which greatly expands the test surface.

Consequences: early cross-team sharing is constrained. Useful public conclusions may require deliberate sanitized releases. Deleting a lineage edge does not widen access.

Revisit cross-domain sharing after access-revocation and inference-leak tests exist.

## ADR 007 Participation through small protocols and external gates

Context: MCP availability does not guarantee retrieval or reporting; mandatory contribution counts reward noise.

Decision: start/context receipt, deliberate evidence submission, challenge, refresh, and final disposition. Allow zero findings. Use Codex and Claude Code instructions/MCP, optional tested hooks, and coordinator/CI enforcement when required.

Alternatives: proprietary harness ownership reduces adoption; stop hooks alone miss interruptions; contribution quotas corrupt incentives.

Consequences: a receipt proves delivery only. Harness adapters are versioned integrations. Enforcement controls accepted outputs, not private thought.

Revisit adapter investment based on observed missed participation and user burden.

## ADR 008 Durable state first and leased coordination later

Context: Phase C needs independent agents and evidence transitions, not a distributed scheduler.

Decision: manually assigned attempts in MVP; later add pull acquisition, renewable leases, fencing, idempotency, and bounded contracts. Retain late evidence separately from expired completion rights. Use a PostgreSQL outbox for durable events.

Alternatives: building speculative or hierarchical coordination now increases scope without testing knowledge value; bare publish-after-commit loses events; a lease without fencing permits stale mutations.

Consequences: early experiments require orchestration outside Holocron. Events are delivered at least once. External worker actions remain outside Holocron's fence.

Revisit when scheduling overhead demonstrably limits the validated workload; introduce JetStream only for identified durable consumers.

## ADR 009 Compounding value requires a controlled experiment

Context: a successful three-agent demonstration or a large graph can coexist with negative net value.

Decision: compare no Holocron, ordinary artifact retrieval, and structured Holocron using fixed model/harness versions, held-out tasks, hidden correctness checks, randomized assignment, and full acquisition/maintenance costs.

Alternatives: self-reported helpfulness, contribution counts, and a single impressive demo cannot establish causal improvement.

Consequences: instrumentation and corpus control are part of the MVP. Small samples produce inconclusive results; no scaling claim follows from the pilot.

Revisit scope only after the preregistered decision criteria are evaluated, including negative results.

## ADR 010 Concrete prototype boundaries

Status: implemented within the approved Phase C scope, 2026-11-24.

Context: the conceptual vocabulary does not require a separate table for every noun. A minimal implementation must preserve semantics without introducing a universal ontology or pretending that human judgment has been automated.

Decision: an immutable evidence manifest combines one attributed observation/execution and versioned artifact references. Typed relational membership and provenance-parent tables enforce linked IDs. The fixed finite-case policy is a code constant; changing it requires a new version and migration, not a worker API. Scope matching uses exact environment dimensions and finite case containment. Module responsibilities live in separate files under one application package, with separate transport and telemetry packages.

Current assessments are recomputed for reads as well as graph writes, so expired review deadlines and revoked credentials cannot remain eligible through a stale projection. Writes store historical snapshots transactionally. Responses used for idempotency are stored as bytes to preserve exact budgeted output; structured knowledge remains JSONB plus relational constraints. Workers report completion; independent correctness adjudication is external to this prototype.

Retention uses explicit administrative whole-domain purge, including copied response and work payloads. Selective privacy redaction is deferred because lineage does not discover every copied free-text disclosure. Backup restores require deletion-ledger replay before service. Ordinary retraction remains append-only.

Alternatives: splitting every manifest component into separately mutable objects adds bookkeeping without making the reported execution true; storing retry responses as JSONB changes their serialized bytes; partial source deletion can leave sensitive summaries accessible; a broker dispatcher is unnecessary with a durable poll/ack outbox API.

Consequences: policy extensibility, cross-domain sharing, selective redaction, production availability, automatic scheduling, and live-model experiment results are not delivered by this increment. The prototype can exercise the eight knowledge-transition behaviors and export measurements; it cannot substantiate a productivity claim by itself.

Revisit these boundaries using the pilot's authoring burden, graph size, retention needs, and measured reuse results.
