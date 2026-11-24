# Proposed hierarchy decision records

Status: **foundational semantics accepted by the founder**, 2026-11-24. ADRs 011–017 extend [accepted ADRs 001–010](decisions.md). Implementation is limited to single-domain organization, broader applicability and retrieval. ADR 018 records the implemented subset; multi-domain and policy-governance features remain deferred.

## ADR 011 Independent organization classification applicability and access

Context: the current `domains` table is an isolation compartment, while `ClaimInput.Concepts` is embedded prose metadata. Using either as the entire hierarchy would conflate administration, relevance, scope, and access.

Decision: introduce stable organizational entities, conceptual classifications, independently versioned claim associations and explicit applicability judgments. Keep visibility domains as security boundaries. Preserve immutable claim revisions and justification semantics. Tenant-configured entity kinds need no schema changes.

Alternatives: a single generic scope tree is simpler to draw but cannot represent overlapping technical disciplines without copying knowledge or accidentally inheriting authority. Arbitrary untyped triples admit relationships with undefined propagation.

Consequences: users must distinguish placement, subject, applicability and permission in authoring. Context cards and tool schemas must make the distinction usable. Same labels do not merge identities.

Acceptance evidence: changing a classification or home placement cannot alter evidence posture; multiple concept paths deduplicate to the same revision; adding a parent grants no access.

Revisit terminology and interface grouping if pilot authors consistently confuse these dimensions, without collapsing their semantics.

## ADR 012 Typed DAGs and stable identities

Context: repositories and components may participate in multiple projects or concerns; a fixed organization/project/repository tree is unnecessarily restrictive.

Decision: organizational containment and conceptual broader relations are separate acyclic graphs. Permit multiple parents; use affiliation and related-concept edges for non-containment relationships. Assign a single accountable home steward to each claim family, with versioned transfers. Preserve IDs through renames and moves.

Alternatives: a strict tree is adequate for simple organizations but forces copies or fake parents for shared ownership. An unrestricted graph makes policy descent and cycle behavior ambiguous. A universal corporate ontology introduces structure without a demonstrated retrieval use.

Consequences: policy obligations from multiple containment paths accumulate; no preferred display path defines precedence. Structural edits require cycle checks and policy-impact review where applicable. Simple organizations can use only one parent.

Acceptance evidence: multi-parent discovery and policy deduplication work; concurrent reverse-edge insertions cannot create cycles; restructuring preserves provenance and historical contexts.

Revisit whether to expose multi-parent authoring in the first UI after actual need is measured; retain a representation that does not require claim duplication.

## ADR 013 Discovery is separate from local applicability

Context: the current exact-environment filter misses useful broader orientation, but merely relaxing it would make unrelated or unsupported claims look applicable.

Decision: context contains distinct policy obligations, factual cards with assessed envelope plus local applicability, and explicitly labeled transfer/orientation candidates. Use authorization-first bounded discovery, relevance gates, deterministic ranking, revision deduplication and progressive disclosure. Broader placement contributes no evidential weight.

Alternatives: strict exact-match retrieval remains a useful baseline but misses qualified analogies. Dumping ancestors wastes budget and encourages authority bias. A blended relevance/confidence score obscures different judgments. Mandatory embeddings or generated summaries add dependencies before retrieval value is shown.

Consequences: “supported elsewhere, applicability unknown here” is a valid response. It cannot satisfy a contract requiring local support. Relevant contradictory or stale knowledge must not be hidden by a preference for supported cards. Mandatory policy overflow makes the bundle incomplete for completion.

Acceptance evidence: a mismatched organization-level claim remains discoverable as orientation without becoming locally established; unrelated ancestors do not fill unused budget; compressed cards retain material qualification.

Revisit the candidate rules and budget allocations through flat-versus-hierarchical trials, not graph-size metrics.

## ADR 014 Generalization requires a new warranted assertion

Context: broader labels, repeated reports, or multiple projects can create the appearance of corroboration without new evidence.

Decision: upward generalization creates a distinguishable claim/envelope and justification. Lateral transfer begins as an explicitly attributed proposal or analogy. Keep common observation roots and methodology dependence. Invalidation follows actual premises, preserving alternative support. The existing finite-case policy cannot establish universal principles.

Alternatives: move a finding to an organization folder; copy it into every project; count associated projects as evidence; automatically use broader-concept relations as logical rules. Each inflates support or loses provenance.

Consequences: useful hypotheses can remain insufficient. An initial bounded multi-context policy can establish only its enumerated coverage; broader mechanism validation is a separate domain-specific decision. Classification-only links do not cause revalidation cascades.

Acceptance evidence: one experiment placed under many concepts yields no support gain; loss of a required local premise affects a generalization regardless of organizational location; an adequate alternative survives.

Revisit broader policies only when their falsifiers, coverage obligations and independence requirements can be reviewed explicitly.

## ADR 015 Explicit normative policy inheritance

Context: hierarchy-based precedence is inappropriate for facts, but authorized organizational requirements do need defined descendant behavior.

Decision: version organizational policy separately from evidence verification policy. Apply explicit target bindings and inheritance modes. Obligations accumulate; scoped exceptions require delegated authority, identify exact policy versions and expire explicitly. Incompatible obligations remain a conflict. Structural edits cannot silently weaken active work contracts.

Alternatives: nearest node wins; newest text wins; all high-level claims become instructions; every project curator can override organization policy. These allow authority escalation or hide contradictions.

Consequences: policy compatibility is evaluated only for a deliberately small structured subset. Free-text uncertainty requires resolution. Workers need authorized access to the effective normative view before receiving a completion-eligible assignment. Historical ADR imports are proposed policy until ratified by current authority.

Acceptance evidence: an authorized exception changes only its named requirement; an expired exception stops applying; changing concept tags cannot evade policy; a project move preserves history and triggers the required policy review.

Revisit richer policy expressions after concrete use cases exceed the structured subset; do not build a general rule engine now.

## ADR 016 Revocation-safe federation and explicit release

Context: a shared concept can connect differently restricted sources. The current retry path returns saved bytes after authenticating a single-domain principal; future partial grant changes require finer checks.

Decision: add explicit domain/action grants before federated retrieval. Authorize nodes, edges, snippets and aggregates before traversal/selection. Recheck permissions before cached-result replay, details and completion. Receipts are opaque references to server-side version dependencies. Keep ordinary cross-domain derivation disabled until intersection-safe destinations and release governance exist.

Alternatives: use one unrestricted service connection and filter only final cards; union input permissions; expose a supported summary without its restricted reasons; let old receipts authorize future access. Each can disclose protected knowledge or stale authority.

Consequences: unchanged authorized retries preserve response bytes, but revoked content is not replayed. Access changes do not imply factual refutation. Explicit sanitized release is a separate artifact with independent authorization and evidence judgments. Previously delivered bytes cannot be recalled from an agent.

Acceptance evidence: changing private concept membership cannot change an unauthorized user's visible response; a revoked grant blocks cached response delivery and completion; metadata moves and releases cannot launder restrictions; purge and restore cover new projections.

Revisit cross-boundary synthesis after these tests pass and release responsibilities are agreed. No separate authorization service is required for the pilot.

## ADR 017 Relational adjacency with incremental adoption

Context: expected queries are bounded ancestors, broader concepts, memberships, policy bindings and existing evidence dependencies. No measured bottleneck justifies another database.

Decision: use typed PostgreSQL adjacency and association tables, active projections with immutable history, bidirectional indexes, tenant-scoped structural mutation locking, and stable lock ordering for combined transactions. Keep closure/path representations as optional derived projections. Separate internal metadata, authorization and evidence versions.

Alternatives: closure tables improve reads at potentially quadratic storage/update cost; materialized paths favor trees and make reparenting expensive; nested sets fit static subtrees poorly; an external graph database adds consistency obligations. None is justified as initial authority.

Consequences: predictable pilot consistency at limited concurrency. Traversal exhaustion is an explicit incomplete/error state, not absence. Authorization-aware closure caching is harder than a global reachability table. Additive migrations must preserve existing IDs, body bytes, assessments and the running finite-case API.

Acceptance evidence: bounded traversal under diamond fan-out, atomic cycle rejection under concurrency, receipt consistency during moves/revocation, and migration/restore tests on the existing schema.

Revisit projections or locking only when representative scale measurements demonstrate the need; reject complexity that cannot beat the flat baseline on useful work.

## ADR 018 Single-domain implementation boundaries

Status: implemented under the approved scope, 2026-11-24.

Context: the next milestone is Wadjet integration. Existing schema/data must survive, and the five independent dimensions must remain separate without introducing a multi-domain authorization system or universal inference engine.

Decision: migration 2 adds typed organizational/conceptual metadata with immutable history. PostgreSQL adjacency is authoritative; bounded Go traversal operates inside the authorized visibility domain. The existing domain transaction lock serializes combined operations. A tenant-wide lock and closure cache are unnecessary while cross-domain edges are prohibited.

Organization/concept kinds are data-defined. Containment and broader-concept relations are separate DAGs; related/affiliated links are non-transitive. Explicit curation grants govern metadata independently from evidence-review roles. In this release the curator represents metadata authority over both ends of a transfer within the compartment; per-node delegation is deferred.

Placement and classification pin exact claim revisions. Family identity derives from the existing revision chain, but a successor is initially unplaced/unclassified until explicitly curated. This tightens the proposed family-home convenience rule: a newly broadened assertion must not borrow organization-level placement approval. Each revision has at most one accepted home/steward. Transfer is atomic and historical. Multiple concept paths never require copying a claim.

Applicability v2 is explicitly unknown or enumerates 1–16 complete finite contexts. There is no universal wildcard language. `software-context-matrix/v1` requires admissible reproduction for every context. Bounded generalizations also require premises, assumptions, limitations, inference kind and a reviewed warrant. Analogies remain insufficient. Two enforcement corrections prevent aggregate failures outside the claimed cases from refuting them, and prevent mere credential expiry from retracting historical evidence. Deliberate principal revocation retains quarantine semantics.

Knowledge, metadata and authorization versions are separate. Classification does not recompute evidence assessments. Receipts pin all three versions. Authorization changes fence cached response delivery without repeating the mutation; unchanged-authority retries preserve historical bytes. Migration invalidates old completion receipts through a metadata-version bump without rewriting bodies.

Retrieval resolves working IDs, deduplicates revisions and separates local applicability from evidence posture. Orientation has an adjustable raw-JSON budget share. Flat retrieval remains available over the same compartment. `applied` cannot contain non-matched orientation; `considered` records exploratory use. Pinned source references preserve import provenance but never count as execution evidence.

Alternatives rejected: universal scope predicates, automatic ontology induction, hierarchy confidence scores, ancestor-based factual precedence, cross-domain joins filtered only at the end, and editing migration 1. Legacy concept strings remain annotations until explicit curation, avoiding automatic same-name merging.

Consequences: ADRs 011–014 and 017 are implemented in a bounded form. ADR 016's single-domain replay/revocation safeguards are implemented; federated discovery, partial grants and declassification remain deferred. ADR 015's normative policy lifecycle is not implemented. Organization-level text never becomes an authoritative instruction through placement. Existing IDs, source bodies and historical assessments survive migration.

Validation: tests cover classification neutrality, positive coverage, lateral orientation, dependency alternatives, cycles, moves, replay, hidden metadata, migration and restore. Real HTTP/MCP tests execute the metadata-only Wadjet bootstrap twice without duplicating nodes. The [release notes](hierarchy-release.md) map capabilities and limits to tests. Retrieval effectiveness remains a live-agent experiment.
