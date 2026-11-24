# Hierarchy implementation and validation plan

Status: **architecture approved; bounded single-domain implementation delivered**, 2026-11-24. This remains the full acceptance specification, including deferred features. [The release notes](hierarchy-release.md) identify implemented tests and exclusions. No live Wadjet productivity result is claimed. This extends [the original experimental design](validation.md).

## Deliverable map

| Requested deliverable | Review location |
| --- | --- |
| Vocabulary | [Independent dimensions](hierarchy-design.md#vocabulary-and-independent-dimensions) |
| Domain model | [Conceptual model and relational responsibilities](hierarchy-design.md#conceptual-model) |
| Relationships and propagation | [Relationship semantics](hierarchy-design.md#relationships-and-propagation) |
| Placement and ownership | [Restructuring and ownership](hierarchy-design.md#restructuring-and-ownership) |
| Hierarchical context | [Assembly algorithm](hierarchy-design.md#hierarchical-context-assembly) |
| Generalization and transfer | [Applicability and knowledge flow](hierarchy-design.md#applicability-and-knowledge-flow) |
| Security and derived knowledge | [Security and consistency](hierarchy-design.md#security-and-consistency) |
| Physical alternatives | [Storage and transaction choices](hierarchy-design.md#physical-storage-and-transaction-choices) |
| Architectural decisions | [Proposed ADRs 011–017](hierarchy-decisions.md) |
| Incremental plan | [Implementation gates](#incremental-implementation-gates) |
| Adversarial tests | [Acceptance specification](#adversarial-acceptance-specification) |
| Wadjet experiment changes | [Bounded scenario](#bounded-wadjet-scenario) and [experiment extension](#experiment-extension) |

## Incremental implementation gates

| Stage | Scope | Required exit evidence |
| --- | --- | --- |
| H0 Baseline and migration contract | Freeze current finite-case behavior, existing live-pilot artifacts, API compatibility and storage invariants. Define independent metadata/auth/evidence versions and receipt semantics. | Existing race-enabled suite remains green; snapshot hashes/IDs survive migration round trips; privacy deletion and restore inventories include all new structures. |
| H1 Domain-local organization and concepts | Tenant-configurable entity kinds, stable entities/concepts, containment/broader DAGs, classifications, placement/stewardship history, explicit curator operations. Multiple projects in one visibility domain. | No evidence-posture change from classification; concurrent cycle rejection; revision deduplication; reconstruction of a pre-move hierarchy. |
| H2 Qualified hierarchical retrieval | Resolve working context, bounded deterministic candidate generation, versioned applicability envelope, orientation cards, progressive disclosure, metadata-aware receipts. Seed a small authorized normative policy subset. | Local applicability and evidence posture stay separate; relevant counterevidence survives budget packing; completion cannot use an orientation card as a supported local premise. |
| H3 Explicit multi-domain access | Separate identity/credentials/grants, authorization-filtered shared catalog, federated read snapshots, replay reauthorization and receipt refresh. Keep ordinary evidence derivations domain-local. | All cross-domain non-disclosure, cache, revocation and race tests pass. No access widening through any hierarchy mutation. |
| H4 Bounded synthesis and normative policy lifecycle | Generalization/transfer proposals, reviewed finite multi-context justification policy, explicit organizational policy exceptions and restructuring impact checks. Manual gap proposals only. | Eight-step Wadjet scenario passes with preserved alternatives; exceptions are versioned and delegated; a broader hypothesis cannot become supported through placement alone. |
| H5 Controlled experiment and conditional expansion | Compare hierarchical and flat structured retrieval on the same corpus and workloads. | Preregistered correctness, cost, misleading-transfer and maintenance criteria evaluated, including negative/inconclusive outcomes. |

H1 and a narrow H2 can establish the first useful hierarchy behavior. H3 is foundational **before** multi-domain discovery; it is not an optional security patch to add afterward. H4's broad inference policy is separately reviewable and cannot be implemented as a label mapped to the existing `finite_cases` evaluator.

Defer cross-domain synthesis/declassification, autonomous scheduling, automatic taxonomy induction/merging, universal knowledge scores, unrestricted policy languages, closure caches, embeddings and server inference. They are not prerequisites for the first hierarchy experiment. Do not defer stable identity, revisioned relationships, access-safe association semantics, budget limits or applicable-policy visibility.

### Concrete migration constraints

Do not edit `schema.sql` and accept a checksum mismatch on an existing deployment. Add numbered migrations with old/new-reader compatibility and explicit backfill checks. New projections must be rebuildable from their recorded assertions. Cut over with feature flags while the legacy API continues to operate.

Existing claim-row IDs remain revision IDs. Add family mapping without changing previous/successor history. Embedded concept strings migrate to proposed associations using tenant/scheme-specific IDs; equal strings across projects do not automatically merge concepts. An administrator maps legacy credentials to identities/grants explicitly; equal principal names across domains are not evidence that they are one person or one independent producer.

Keep existing exact scope as envelope v1. It must not become universal when mapped to v2. Add a tagged envelope/policy dispatch path before accepting repository-independent hypotheses. Known-unconstrained, unknown and not-applicable fields have different semantics. The `generalization` claim kind stays insufficient until an appropriate verification policy exists.

Suggested tool additions are `context.resolve`, organizational/concept lookup, association proposal/review, placement transfer proposal/review, transfer/generalization investigation proposals, and progressive detail expansion. Tool names and schemas must be tested in the actual Codex/Claude adapters, including literal dotted-name approval configuration. No worker tool may grant access, authorize a release or change verification criteria implicitly.

Classifications and topology changes append events and update metadata projections atomically. Existing evidence assessment history is not rewritten. Split authentication/grant revocation from trust/quarantine events through an explicit compatibility migration. Preserve conservative behavior where existing records cannot be safely disambiguated and require administrative disposition.

## Adversarial acceptance specification

These are proposed executable test requirements. They have not been run against a hierarchical implementation. Use real PostgreSQL, non-owner runtime credentials, independent connections with synchronization barriers, both REST and MCP, and fresh agent sessions where model behavior is the subject.

| ID | Scenario | Required observation |
| --- | --- | --- |
| H01 | Repository classified as backend and data engineering | Both authorized paths discover the same claim revision; one card, two visible reasons; no copied claim or assessment. |
| H02 | Organization-placed claim lacks local repository applicability | Relevant orientation is labeled unknown/mismatched/proposed transfer; its source posture remains intact; it cannot satisfy a local-supported completion gate. |
| H03 | Local finding differs from a broader claim | Exact overlap, assumptions and quantifiers are evaluated. In-scope counterevidence can challenge; disjoint scopes do not silently overwrite either claim. |
| H04 | Multiple projects quote one benchmark through several reports | Common roots remain visible to authorized evaluators; repeated paths/reports do not satisfy an independent reproduction obligation. |
| H05 | Generalization requires an invalidated component finding | Only actual dependent justification paths become unusable, regardless of placement or concept path. |
| H06 | Same generalization has an adequate unaffected alternative | Alternative remains eligible and explainable; no blanket false/invalid label from reachability. |
| H07 | One successful experiment is promoted to an organization principle | New generalization remains insufficient without a broader warrant; relabeling or moving the old claim changes no support. |
| H08 | Restricted repository associates a finding with a public concept | Unauthorized concept search, snippets, counts, details, related paths and summaries remain unchanged whether that association exists or not. |
| H09 | Claim is reached through several concepts and ancestors | Deduplicate by exact revision; retain authorized retrieval reasons without path-count ranking inflation. |
| H10 | Project moves between units or a repository is renamed | Stable identities and old assessments/provenance survive; current policy/context reflects the reviewed move; historical receipts retain old topology. |
| H11 | Organization policy and authorized project exception overlap | Only the named policy revision/targets are excepted, within delegated authority and expiry. Other obligations remain. |
| H12 | Unrelated ancestor claims are numerous and marked important | They do not displace task-relevant facts or warnings merely through placement/importance; empty spare budget is acceptable. |
| H13 | Proposed containment or broader-concept cycle, including concurrent opposing inserts | Invalid mutation is rejected atomically. Legal related-concept cycles remain bounded and do not affect derivation semantics. |
| H14 | Grant revoked after a bundle was delivered | New reads, detail handles, exact-key cached retries and completion respect the revocation. No retained receipt grants access. |
| H15 | Public endpoints linked only through an inaccessible intermediate node | Hidden node is not used as a discovery bridge; no path, hop count or hidden relationship is inferred. |
| H16 | Two authorized containment parents impose conflicting policies | Both obligations are identified; no shortest-path or nearest-parent winner. Completion requiring compliance remains blocked pending resolution. |
| H17 | Curator moves an entity or changes its kind to escape policy | Separate policy-impact approval is required; existing contracts are not weakened by metadata edits. |
| H18 | A new repository joins a project covered by an empirical claim | Existing assessment's captured target set does not expand. Relevant orientation is possible; new local applicability remains unvalidated. |
| H19 | A low-privilege agent chooses an organization-level home or edits shared classifications | Proposal can be retained within authorized space; accepted placement/publication requires curator authority. It grants no evidence or policy approval. |
| H20 | Private evidence is added as an alternative to a publicly visible claim | Mutation cannot silently change public canonical posture or reveal restricted reasoning. Reject or route to an explicitly restricted derivation/release workflow. |
| H21 | Restricted input link is removed from a derived summary | Visibility cannot widen through unlinking. Historical information dependencies and retention obligations remain. |
| H22 | Exact-key replay follows a permission or classification change | No unauthorized old bytes are returned; mutation is not repeated; internal idempotency outcome remains auditable. |
| H23 | Unrelated private topology changes | Public payloads, rankings, visible counts and exposed versions remain equal in a paired database test. Timing is measured separately rather than claimed perfectly hidden. |
| H24 | Context is assembled while grants, topology or policies change | Bundle reflects one valid snapshot; completion rechecks the relevant versions. No mixed grant/topology view or stale normative receipt. |
| H25 | Mandatory policy/exceptions or factual qualifications exceed budget | Explicit incomplete/expansion result; no falsely complete receipt and no stripped-warning assertion. |
| H26 | Traversal depth/fan-out bound is reached in a diamond DAG | Explicit truncation/incomplete result. A graph bound is never interpreted as policy absence, an acyclic proof or absence of evidence. |
| H27 | Two concepts share a label, or a client supplies inconsistent context identifiers | No unauthorized alias resolution or automatic merge. Ambiguity and invalid combinations are explicit without exposing hidden candidates. |
| H28 | A concept definition changes after claims were classified under it | Classification review can be suggested; factual revalidation occurs only when an actual definition/warrant dependency exists. |
| H29 | A policy exception expires or policy version changes | Read-time applicability and completion update even if no background job ran; the old exception does not migrate itself. |
| H30 | The agent loses a grant while attempting a legitimate stewardship transfer | Access, source/destination acceptance and transfer commit are checked consistently. Ownership cannot launder missing access. |
| H31 | New taxonomy tables, caches and receipts exist when a domain is purged and an old backup restored | Purge/ledger replay removes protected payloads and references across the extended model; no dangling public summary or revived cached response. |
| H32 | Agent labels fetched material as an organizational policy | It remains untrusted data unless an authorized policy record exists; no tool execution or privilege derives from retrieved prose. |
| H33 | Agent uses an analogy while reporting a locally established finding | Contract rejects the unsupported local application or records it as explicitly permitted exploratory reasoning; it cannot silently satisfy verification. |
| H34 | Token expires or a read grant is removed from an otherwise trusted observer | Future access is denied; historical observation is not thereby declared false. Explicit distrust/quarantine is a different event. |
| H35 | Status prose conflicts with updated structured assessment | Rendering preserves the discrepancy and attribution; no hidden prose rewrite. Authoring guidance discourages transient status claims, and the model's handling is evaluated. |
| H36 | Two paths reveal the same release and original restricted source | Release is not counted as a fresh root; unauthorized provenance fields stay hidden according to the approved release record. This is a gate for future release support, not an initial feature. |
| H37 | Agent floods broad concept associations or pending objections | Unaccepted classification does not gain curated authority; repeated paths do not increase rank; bounded warnings preserve material disputes without rewarding volume. |

Include property tests: adding classification paths never raises evidence posture; every ordinary derived audience is no broader than its inputs; classification-only changes leave assessment content unchanged; a new grant cannot mutate historical evidence; arbitrary renames preserve IDs; removal of one alternative leaves others intact. For confidentiality, run paired states differing only in hidden records and compare complete public response bodies, pagination behavior, and search results.

## Bounded Wadjet scenario

Use a frozen source revision and the already identified ADRs in [the research sample](research.md), then follow only the tests and artifacts needed for the task. The historical ADRs are sources to extract from, not automatic verification or current organizational policy.

Seed organizational entities for Engineering, Wadjet, its repository, query planning, execution engine and hash aggregation. Use conceptual IDs for general engineering, database systems, distributed execution, memory management, spill behavior, SQL correctness, and fault tolerance. Project architecture is a maintained collection of claim associations, not a numerical concept strength. Classify the repository under several concepts. Tenant-defined kinds and labels remain editable without DDL.

Also seed a second project representing a lateral-transfer target, plus a restricted project in a separate visibility domain. Transit may be used as the target label, but no factual Transit finding is invented without an actual source. A clearly labeled synthetic target fixture can test protocol semantics until a real second-project workload is supplied. Do not count it as demonstrated production transfer.

The concrete investigation: **design and validate a regression gate for hash aggregation under forced spill without confusing non-engaged spill, legal nondeterminism, and real wrong answers**. This needs higher-level validation methodology, project SQL constraints, and component engagement evidence. It is bounded by a fixed commit, a small fixture set, an explicit runtime, an independently derived expected result, and a time/token budget. The hypothesis is not “spilling is generally correct.”

Suggested initial corpus bound: at most 24 reviewed candidate revisions across levels and concepts, including relevant contradictory/negative material and irrelevant distractors. This is a workload limit, not a success metric. A steward may ratify a small evaluation policy for the pilot; imported ADR policy text remains reported/unratified until then. Reproduce selected factual findings before labeling them supported.

Demonstrate the eight requested behaviors in this order:

1. A fresh worker resolves its organization/project/repository/component and receives authorized validation policy, relevant project invariants, and local spill findings. Explicitly exclude unrelated security topics unless the task triggers them.
2. The bundle distinguishes supported source findings, local applicability, unvalidated higher-level principles, exceptions and pending objections. Missing conditions remain visible.
3. The worker executes the finite investigation, publishes a precise component finding with actual trigger/result/oracle provenance, and proposes its evidence attachment immediately. Empty findings remain acceptable if it discovers nothing new.
4. A second fresh agent requests context through the memory-management or distributed-execution concept and finds the same revision, without a copied claim. Measure whether it uses the finding correctly.
5. A manually authorized synthesis investigation proposes a broader statement, for example that correctness checks for condition-triggered paths need evidence that the path engaged. Label it a generalization and identify assumptions, observed operator families, and known limits.
6. Its placement under general engineering does not establish it. The finite-case evaluator refuses broad acceptance. A separately reviewed bounded policy may establish the claim only for explicitly validated contexts; the unrestricted principle stays insufficient.
7. Introduce a genuinely observed contradiction or a clearly labeled fault-injected fixture where a recorded engagement signal measured the wrong event. Review the challenge; invalidate the affected justification; verify that the broader dependent needs revalidation while an adequate alternative survives in a separate branch of the test. Never present injected evidence as a naturally discovered Wadjet defect.
8. Fresh retrieval reflects that change and excludes inaccessible material. A twin hidden-project fixture must not alter the unauthorized bundle, including counts, reasons and cache behavior.

Actual semantic contradictions are not required to occur naturally on demand. If the investigation finds no counterexample, retain that bounded negative result and use the labeled adversarial fixture solely to test propagation.

## Experiment extension

Retain the existing three-arm framing, but split structured retrieval:

| Arm | Available context |
| --- | --- |
| A | Repository/tools/instructions, no Holocron. |
| B | Ordinary document retrieval over the approved source corpus. |
| C-flat | Structured claims, evidence and qualifications with flat authorized retrieval. |
| C-hierarchy | The same claims, evidence, qualifications, grants and policies, plus hierarchical/conceptual candidate routing. |

The primary comparison for this extension is **C-hierarchy versus C-flat**. Give both the same policy obligations, source corpus, source snapshot, accessible candidate universe, context budget, tool capabilities and verification gates. Flat retrieval must search across all authorized records, not be artificially limited to one repository. Both must have equivalent applicability/orientation semantics; otherwise the experiment would conflate those improvements with hierarchy.

Use the same fixed model and harness version within each randomized block, clean sessions and equal task/resource limits. Block by task family and model; do not compare different vendors as if that identified a hierarchy effect. Freeze reviewed hierarchy/classification snapshots before evaluation. Keep held-out solutions and their classifications out of training/curation inputs. Blind correctness assessment and record failed, censored and interrupted attempts.

Include tasks where hierarchy should help (cross-level constraint, lateral finding), where it should not (a precise local lookup), and where it could harm (misleading ancestor principle or stale generalization). This provides a chance to falsify the recommendation. A small pilot estimates variance and curation burden; it is not evidence of statistical superiority. Choose confirmatory sample size from the minimum useful effect and pilot variance, not a convenient fixed task count.

Measure existing correctness/time/token/cost/redundancy/reuse outcomes, plus:

* Precision and recall against an independently reviewed, versioned relevant-card set, with applicability warnings scored separately.
* Correct use of local findings versus unjustified analogical transfer or ancestor authority bias.
* Inclusion of material contradictions, exceptions and missing conditions; omission counts are not automatically errors when a card is irrelevant.
* Useful validated propositions actually applied per total delivered model token, including tool wrappers, repeated tool catalog/context, and progressive-disclosure calls. Also report the underlying totals to prevent ratio gaming.
* Additional lookups, context-resolution failures, receipt invalidations and curation/reclassification/reorganization cost.
* Security outcomes from paired visible/hidden datasets. Confidentiality failures are release blockers, not a cost/correctness tradeoff.

Track proposal versus accepted classification, retrieved versus applied knowledge, and observed versus inferred conclusions. Raw card count, concept depth, number of graph edges and contribution volume are not success outcomes. Report acquisition, evidence review, taxonomy maintenance and execution costs together.

The first live pilot already showed high repeated-context overhead, failed launches, missing attachment and scope-provenance friction. Keep those costs in this experiment. The hierarchy treatment must not receive extra hand-curated explanations unavailable to the flat arm. If helpful explanations are part of the treatment, charge their creation and identify that as a separate package effect.

Preregister the acceptable correctness margin, minimum useful resource improvement, harmful-transfer threshold, and maintenance budget before confirmatory outcomes are inspected. Accept a result where flat retrieval is better. An inconclusive result should constrain scope rather than trigger a larger taxonomy or more infrastructure.

## Review checklist

Approve or amend the separation of the five dimensions; typed multiple-parent containment and conceptual DAGs; orientation-versus-local-applicability cards; explicit policy exceptions without factual precedence; and the H3 authorization/replay gate before multi-domain discovery. H0–H2 are the recommended first implementation slice after semantics review. H4's generalized evidence policy and any future declassification mechanism require their own concrete review before activation.
